package wsserver

import (
	"sync"

	"github.com/mailru/easygo/netpoll"
)

func newNetPoller(serv *Server) *netPoller {
	poller, _ := netpoll.New(nil)

	descPool := &netPoller{
		serv:   serv,
		Poller: poller,
	}
	descPool.pool.New = func() interface{} {
		return &descriptor{}
	}
	return descPool
}

type netPoller struct {
	netpoll.Poller
	serv *Server
	pool sync.Pool
}

func (poller *netPoller) get(c *Context) (*descriptor, error) {
	desc := poller.pool.Get().(*descriptor)
	desc.c = c
	readDesc, err := netpoll.HandleRead(desc.c.Conn.Conn)
	if err != nil {
		return nil, err
	}
	desc.desc = readDesc

	return desc, nil
}

func (poller *netPoller) register(c *Context) error {
	d, err := poller.get(c)
	if err != nil {
		return err
	}
	return poller.Start(d.desc, func(event netpoll.Event) {
		if event&netpoll.EventHup != 0 || event&netpoll.EventReadHup != 0 {
			if err := d.c.Close(); err != nil {
				poller.serv.options.Logger.Warnf("ws: close the websocket connection failed, err:%+v", err)
			}
			poller.Stop(d.desc)
			poller.serv.options.ConnClosedHook(d.c)
			poller.serv.releaseContext(d.c)
			poller.release(d)
			return
		}
		poller.serv.handleMessage(d.c)
	})
}

func (poller *netPoller) release(d *descriptor) {
	poller.pool.Put(d)
}

type descriptor struct {
	desc *netpoll.Desc
	c    *Context
}
