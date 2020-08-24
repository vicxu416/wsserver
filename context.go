package wsserver

import (
	"context"
	"net"
	"reflect"
	"sync"
	"unsafe"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
)

// Binder define binding data interface
type Binder interface {
	Bind(data []byte) error
}

// Context represetn websocket connection context
type Context struct {
	Ctx       context.Context
	ctxCancel context.CancelFunc
	ID        string
	Conn      *onceCloseConn

	opcode  ws.OpCode
	payload []byte
	rw      sync.RWMutex
	serv    *Server
	store   map[string]interface{}
}

// Error invoke msg err handler function
func (c *Context) Error(err error) {
	c.serv.msgErrorHandlerFunc(c, err)
}

func (c *Context) read() error {
	c.rw.Lock()
	defer c.rw.Unlock()

	data, opcode, err := wsutil.ReadClientData(c.Conn)
	if err != nil {
		return err
	}
	c.opcode = opcode
	c.payload = data
	return nil
}

// Bind bind payload to target
func (c *Context) Bind(binder Binder) error {
	return binder.Bind(c.Payload())
}

// Logger return logger from websocker server options
func (c *Context) Logger() Logger {
	return c.serv.options.Logger
}

// Reset reset connection context
func (c *Context) Reset(conn net.Conn) {
	onceConn := &onceCloseConn{Conn: conn}
	ctx, cancel := context.WithCancel(context.Background())
	c.Conn = onceConn
	c.Ctx = ctx
	c.ctxCancel = cancel
	c.ID = uuid.New().String()
}

// Payload return websocket message payload
func (c *Context) Payload() []byte {
	c.rw.RLock()
	defer c.rw.RUnlock()
	return c.payload
}

// OpCode return websocket opcode
func (c *Context) OpCode() ws.OpCode {
	c.rw.RLock()
	defer c.rw.RUnlock()
	return c.opcode
}

// Close close the connection from server
func (c *Context) Close() error {
	c.rw.Lock()
	defer c.rw.Unlock()
	c.payload = nil
	c.opcode = ws.OpClose
	c.ctxCancel()
	c.serv.wg.Done()

	return c.Conn.OnceClose()
}

// WriteBinary write binary data
func (c *Context) WriteBinary(data []byte) error {
	return wsutil.WriteServerBinary(c.Conn, data)
}

// WriteText write text data
func (c *Context) WriteText(data string) error {
	return wsutil.WriteServerText(c.Conn, StringToBytes(data))
}

// ClientAddr represent client ip address
func (c *Context) ClientAddr() net.Addr {
	return c.Conn.RemoteAddr()
}

// Get get value from context
func (c *Context) Get(key string) interface{} {
	c.rw.RLock()
	defer c.rw.RUnlock()
	return c.store[key]
}

// Set set value into context
func (c *Context) Set(key string, val interface{}) {
	c.rw.Lock()
	defer c.rw.Unlock()

	if c.store == nil {
		c.store = make(map[string]interface{})
	}
	c.store[key] = val
}

type onceCloseConn struct {
	net.Conn
	once     sync.Once
	closeErr error
}

func (l *onceCloseConn) OnceClose() error {
	l.once.Do(l.close)
	return l.closeErr
}

func (l *onceCloseConn) close() {
	l.closeErr = l.Conn.Close()
}

// StringToBytes converts string to byte slice without a memory allocation.
func StringToBytes(s string) (b []byte) {
	sh := *(*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	bh.Data, bh.Len, bh.Cap = sh.Data, sh.Len, sh.Len
	return b
}
