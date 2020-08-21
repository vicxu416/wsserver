package wsserver

import (
	"context"
	"net"
	"sync"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
)

func newContext(conn net.Conn) *Context {
	onceConn := &onceCloseConn{Conn: conn}
	ctx, cancel := context.WithCancel(context.Background())
	return &Context{
		ID:        uuid.New().String(),
		Conn:      onceConn,
		Ctx:       ctx,
		ctxCancel: cancel,
	}
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
	wg      *sync.WaitGroup
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

// GetPayload get payload
func (c *Context) GetPayload() []byte {
	c.rw.RLock()
	defer c.rw.RUnlock()
	return c.payload
}

// Close close the connection from server
func (c *Context) Close() error {
	c.rw.Lock()
	defer c.rw.Unlock()
	c.payload = nil
	c.opcode = ws.OpClose
	c.ctxCancel()
	c.wg.Done()

	return c.Conn.OnceClose()
}

// WriteBinary write binary data
func (c *Context) WriteBinary(data []byte) error {
	return wsutil.WriteServerBinary(c.Conn, data)
}

// WriteText write text data
func (c *Context) WriteText(data []byte) error {
	return wsutil.WriteServerText(c.Conn, data)
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
