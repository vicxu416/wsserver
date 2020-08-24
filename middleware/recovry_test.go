package middleware

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vicxu416/wsserver"

	"github.com/gorilla/websocket"
)

func NewServer(addr, port string) *wsserver.Server {
	wsServer := wsserver.NewDefault()
	wsServer.Addr = addr
	wsServer.Port = port
	wsServer.MsgHandlerFunc = func(c *wsserver.Context) error {
		panic(errors.New("testing error"))
	}
	return wsServer
}

func Dail(addr, port string) (*websocket.Conn, error) {
	conn, _, err := websocket.DefaultDialer.Dial("ws://"+addr+":"+port, nil)
	return conn, err
}
func TestRecovery(t *testing.T) {
	var (
		addr = "127.0.0.1"
		port = "13333"
	)
	serv := NewServer(addr, port)
	buf := new(bytes.Buffer)
	writer := io.MultiWriter(buf, os.Stdout)
	serv.Options().Logger.SetOutput(writer)
	serv.Use(Recover())
	go func() {
		serv.Start()
	}()
	time.Sleep(50 * time.Millisecond)

	conn, err := Dail(addr, port)
	assert.NoError(t, err)
	msg := "hello!"
	err = conn.WriteMessage(websocket.TextMessage, []byte(msg))
	assert.Nil(t, err)
	time.Sleep(50 * time.Millisecond)
	assert.Contains(t, buf.String(), "PANIC RECOVER")
}
