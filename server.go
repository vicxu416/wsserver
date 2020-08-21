package wsserver

import (
	"net"
	"sync"
	"time"

	"github.com/gobwas/ws"

	"github.com/mailru/easygo/netpoll"
)

// HandleFunc hanle websocket message
type HandleFunc func(*Context) error

// ListenAndServe make websocket server serve on given address and port
func ListenAndServe(addr, port string, options ...Option) error {
	return nil
}

// Server websocket server
type Server struct {
	Addr    string
	Port    string
	Handler HandleFunc

	poller             netpoll.Poller
	ln                 net.Listener
	wg                 *sync.WaitGroup
	HandlerErrorHandle ConnErrHandle
	shutdownSignal     chan struct{}
	options            Options
}

// ErrorHandle error handle function for handler error
func (serv *Server) ErrorHandle(handle func(ctx *Context, err error)) {
	serv.HandlerErrorHandle = handle
}

// Start start the websocket server
func (serv *Server) Start() error {
	return nil
}

// Shutdown graceful shutdown server
func (serv *Server) Shutdown(timeout time.Duration) error {
	err := serv.ln.Close()
	serv.wg.Wait()
	return err
}

func (serv *Server) resolvedAddr() string {
	return serv.Addr + ":" + serv.Port
}

func (serv *Server) serve() {
	for {
		conn, err := serv.ln.Accept()
		if err != nil {
			serv.options.ErrHandler(err)
		}
		ctx, err := serv.upgradeToWS(conn)
		if err != nil {
			serv.options.ErrHandler(err)
		}
		if err := serv.registerNetpoll(ctx); err != nil {
			serv.options.ErrHandler(err)
		}
	}
}

func (serv *Server) upgradeToWS(conn net.Conn) (*Context, error) {
	if _, err := ws.Upgrade(conn); err != nil {
		return nil, err
	}
	wsCtx := newContext(conn)
	serv.wg.Add(1)
	return wsCtx, nil
}

func (serv *Server) registerNetpoll(c *Context) error {
	// register read event on websocket connection and make a file descriptor
	desc, err := netpoll.HandleRead(c.Conn)
	if err != nil {
		return err
	}

	// add file descriptor to epoll observation list
	serv.poller.Start(desc, func(event netpoll.Event) {
		// when receive closed or receive/send is closed
		if event&netpoll.EventHup != 0 || event&netpoll.EventReadHup != 0 {
			if err := c.Close(); err != nil {
				serv.options.ErrHandler(err)
			}
			serv.poller.Stop(desc)
			return
		}
		serv.handleMessage(c)
	})
	return nil
}

func (serv *Server) handleNetEvent(event netpoll.Event) {

}

func (serv *Server) handleMessage(c *Context) {
	go func() {
		if err := c.read(); err != nil {
			serv.options.ErrHandler(err)
		}
		if err := serv.Handler(c); err != nil {
			serv.HandlerErrorHandle(c, err)
		}
	}()
}
