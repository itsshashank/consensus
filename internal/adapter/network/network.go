package network

import (
	"log/slog"
	"sync"

	"github.com/itsshashank/consensus/types"
)

type Network struct {
	mx    sync.Mutex
	Nodes map[int]chan types.Message
}

func New() *Network {
	return &Network{
		mx:    sync.Mutex{},
		Nodes: make(map[int]chan types.Message),
	}
}

func (n *Network) Register(id int, ch chan types.Message) {
	n.mx.Lock()
	defer n.mx.Unlock()
	n.Nodes[id] = ch
}

func (n *Network) Send(msg types.Message) {
	go func() {
		defer func() {
			n.mx.Unlock()
			if r := recover(); r != nil {
				slog.Error("Recovered from send on closed channel", "Node", msg.To, "Error", r)
			}
		}()
		n.mx.Lock()
		ch, ok := n.Nodes[msg.To]
		if !ok {
			slog.Error("skipping send as node is down", "Node", msg.To, "From", msg.From)
			return
		}
		ch <- msg
	}()
}

func (n *Network) Unregister(id int) {
	n.mx.Lock()
	defer n.mx.Unlock()
	delete(n.Nodes, id)
}
