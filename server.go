package wsserver

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/gobwas/ws"
	"github.com/google/uuid"
)

var (
	// ErrServerClosed represent server closed error
	ErrServerClosed = errors.New("ws: Server closed")
)

const (
	// CtxRequestID define request id key
	CtxRequestID = "context-requset-id"
)

// MsgHandlerFunc define a function to handle websocket message
type MsgHandlerFunc func(*Context) error

// MiddlewareFunc define a function to process middleware
type MiddlewareFunc func(MsgHandlerFunc) MsgHandlerFunc

// ListenAndServe make websocket server serve on given address and port
func ListenAndServe(addr, port string, handlerFunc MsgHandlerFunc, options ...Option) error {
	server := New()
	server.Addr = addr
	server.Port = port
	server.MsgHandlerFunc = handlerFunc
	for _, opt := range options {
		opt(server.options)
	}
	return server.Start()
}

// NewDefault build an instance of websocket server with default options
func NewDefault() *Server {
	serv := New()
	serv.options = &DefaultOptions
	serv.msgErrorHandlerFunc = func(c *Context, err error) {
		c.WriteText(err.Error())
	}
	return serv
}

// New a web socket server
func New(options ...Option) *Server {

	serv := &Server{
		middleware:  make([]MiddlewareFunc, 0, 1),
		pool:        sync.Pool{},
		wg:          &sync.WaitGroup{},
		options:     &Options{},
		closeSignal: make(chan struct{}),
	}
	serv.pool.New = func() interface{} {
		return serv.NewContext(nil)
	}
	serv.netPoller = newNetPoller(serv)
	for _, opt := range options {
		opt(serv.options)
	}
	return serv
}

// Server websocket server
type Server struct {
	Addr           string
	Port           string
	MsgHandlerFunc MsgHandlerFunc
	middleware     []MiddlewareFunc

	pool                sync.Pool
	netPoller           *netPoller
	ln                  net.Listener
	wg                  *sync.WaitGroup
	msgErrorHandlerFunc ConnErrHandle
	options             *Options
	closeSignal         chan struct{}
}

// NewContext create a instance of connection context
func (serv *Server) NewContext(conn net.Conn) *Context {
	onceConn := &onceCloseConn{Conn: conn}
	ctx, cancel := context.WithCancel(context.Background())
	return &Context{
		ID:        uuid.New().String(),
		Conn:      onceConn,
		Ctx:       ctx,
		ctxCancel: cancel,
	}
}

// MsgErrorHandleFunc error handle function for handler error
func (serv *Server) MsgErrorHandleFunc(handle func(c *Context, err error)) {
	serv.msgErrorHandlerFunc = handle
}

// Use append middleware on msg handlerfunc
func (serv *Server) Use(middlewares ...MiddlewareFunc) {
	serv.middleware = append(serv.middleware, middlewares...)
}

// Options get servier options
func (serv *Server) Options() *Options {
	return serv.options
}

// SetOptions set options to server
func (serv *Server) SetOptions(opts *Options) {
	serv.options = opts
}

// Start start the websocket server
func (serv *Server) Start() error {
	serv.MsgHandlerFunc = applyMiddleware(serv.MsgHandlerFunc, serv.middleware...)
	if serv.Addr == "" {
		serv.Addr = "127.0.0.1"
	}
	if serv.Port == "" {
		serv.Port = "10003"
	}

	ln, err := net.Listen("tcp", serv.resolvedAddr())
	if err != nil {
		return err
	}
	serv.ln = &onceCloseListener{Listener: ln}
	serv.options.Logger.Infof("⇨ websocket server started on %s\n", colorize(serv.resolvedAddr(), colorGreen))
	return serv.serve()
}

// Shutdown graceful shutdown server
func (serv *Server) Shutdown(timeout time.Duration) error {
	close(serv.closeSignal)
	err := serv.ln.Close()

	closed := make(chan struct{})

	go func() {
		serv.wg.Wait()
		close(closed)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case <-closed:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (serv *Server) getContext() *Context {
	return serv.pool.Get().(*Context)
}

func (serv *Server) releaseContext(c *Context) {
	serv.pool.Put(c)
}

func (serv *Server) resolvedAddr() string {
	return serv.Addr + ":" + serv.Port
}

func (serv *Server) serve() error {
	var tempDelay time.Duration // how long to sleep on accept failure

	for {
		conn, err := serv.ln.Accept()
		if err != nil {
			select {
			case <-serv.closeSignal:
				return ErrServerClosed
			default:
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				serv.options.Logger.Warnf("ws: tcp listener Accept failed, err: %s; retrying in %+v", colorize(err, colorRed), tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return err
		}
		wsCtx, err := serv.upgradeToWS(conn)
		if err != nil {
			serv.options.Logger.Warnf("ws: upgrade client connection to websocket failed, ignore this connection, err: %s", colorize(err, colorRed))
			continue
		}
		if err := serv.netPoller.register(wsCtx); err != nil {
			serv.options.Logger.Warnf("ws: register client connection to net poll failed, ignore this connection, err: %s", colorize(err, colorRed))
			continue
		}
		serv.options.ConnOpendHook(wsCtx)
	}
}

func (serv *Server) upgradeToWS(conn net.Conn) (*Context, error) {
	if _, err := ws.Upgrade(conn); err != nil {
		return nil, err
	}
	wsCtx := serv.getContext()
	wsCtx.Reset(conn)
	wsCtx.serv = serv
	serv.wg.Add(1)
	return wsCtx, nil
}

func (serv *Server) handleMessage(c *Context) {
	go func() {
		if err := c.read(); err != nil {
			serv.options.Logger.Warnf("ws: read payload from websocket connection failed, ignore this message, err:%+v", err)
			return
		}
		if err := serv.MsgHandlerFunc(c); err != nil {
			serv.msgErrorHandlerFunc(c, err)
		}
	}()
}

func applyMiddleware(h MsgHandlerFunc, middlewares ...MiddlewareFunc) MsgHandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

type onceCloseListener struct {
	net.Listener
	once     sync.Once
	closeErr error
}

func (l *onceCloseListener) OnceClose() error {
	l.once.Do(l.close)
	return l.closeErr
}

func (l *onceCloseListener) close() {
	l.closeErr = l.Listener.Close()
}
