package core

import "github.com/itsshashank/consensus/types"

type Network interface {
	Send(msg types.Message)
	Register(id int, ch chan types.Message)
	Unregister(id int)
}

type Storage interface {
	SaveState(term int, votedFor int, log []types.LogEntry) error
	LoadState() (term int, votedFor int, log []types.LogEntry, err error)
}
