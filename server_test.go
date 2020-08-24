package wsserver

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func NewServer(addr, port string) *Server {
	wsServer := NewDefault()
	wsServer.Addr = addr
	wsServer.Port = port
	wsServer.MsgHandlerFunc = func(c *Context) error {
		resp := fmt.Sprintf("echo: %s", c.Payload())
		return c.WriteText(resp)
	}
	return wsServer
}

func Dail(addr, port string) (*websocket.Conn, error) {
	conn, _, err := websocket.DefaultDialer.Dial("ws://"+addr+":"+port, nil)
	return conn, err
}

var (
	addr = "127.0.0.1"
	port = "1333"
)

func TestEchoHandle(t *testing.T) {
	serv := NewServer(addr, port)
	go func() {
		serv.Start()
	}()
	time.Sleep(50 * time.Millisecond)

	conn, err := Dail(addr, port)
	assert.NoError(t, err)
	msg := "hello!"
	err = conn.WriteMessage(websocket.TextMessage, []byte(msg))

	assert.Nil(t, err)

	_, data, err := conn.ReadMessage()
	assert.Nil(t, err)

	assert.Contains(t, string(data), msg)
	err = conn.Close()
	assert.Nil(t, err)

	err = serv.Shutdown(1 * time.Second)
	assert.Nil(t, err)
}

func TestGracefulShutdown(t *testing.T) {
	serv := NewServer(addr, port)
	go func() {
		serv.Start()
	}()
	time.Sleep(50 * time.Millisecond)
	now := time.Now()
	conn, err := Dail(addr, port)
	assert.NoError(t, err)

	go func() {
		time.Sleep(1 * time.Second)
		conn.Close()
	}()
	err = serv.Shutdown(5 * time.Second)
	assert.Nil(t, err)
	assert.True(t, time.Now().Add(-1).After(now))
}

func Benchmark(b *testing.B) {
	serv := NewServer(addr, port)
	go func() {
		serv.Start()
	}()
	time.Sleep(50 * time.Millisecond)
	conns := make([]*websocket.Conn, 0, 10)
	for i := 0; i < 100; i++ {
		conn, err := Dail(addr, port)
		if err == nil {
			msg := "hello!"
			err := conn.WriteMessage(websocket.TextMessage, []byte(msg))
			assert.NoError(b, err)
			conns = append(conns, conn)
		}

	}
	b.Logf("num of gorountine:%d", runtime.NumGoroutine())
}
