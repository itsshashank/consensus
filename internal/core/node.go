package core

import (
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/itsshashank/consensus/types"
)

type Node struct {
	Mu            sync.Mutex
	ID            int
	State         types.State
	CurrentTerm   int
	VotedFor      int
	VoteCount     int
	Log           []types.LogEntry
	CommitIndex   int
	Peers         []int
	MsgCh         chan types.Message
	Network       Network
	ElectionReset chan struct{}
	AckCount      map[int]int
	Storage       Storage
}

func NewNode(id int, peers []int, net Network, store Storage) *Node {
	ch := make(chan types.Message, 100)
	net.Register(id, ch)
	term, votedFor, log, _ := store.LoadState()

	return &Node{
		ID:            id,
		State:         types.Follower,
		CurrentTerm:   term,
		VotedFor:      votedFor,
		Log:           log,
		CommitIndex:   -1,
		Peers:         peers,
		MsgCh:         ch,
		Network:       net,
		ElectionReset: make(chan struct{}),
		AckCount:      map[int]int{},
		Storage:       store,
	}
}

func (n *Node) Run() {
	slog.Info(fmt.Sprintf("[Node %d] Starting up", n.ID))
	timer := time.NewTimer(randomTimeout())

	for {
		select {
		case msg := <-n.MsgCh:
			n.handleMessage(msg)

		case <-timer.C:
			n.Mu.Lock()
			if n.State != types.Leader {
				slog.Info(fmt.Sprintf("[Node %d] Election timeout. Starting election.", n.ID))
				n.Mu.Unlock()
				n.startElection()
			} else {
				n.Mu.Unlock()
			}
			timer.Reset(randomTimeout())

		case <-n.ElectionReset:
			timer.Reset(randomTimeout())
		}
	}
}

func (n *Node) startElection() {
	n.Mu.Lock()
	n.State = types.Candidate
	n.CurrentTerm++
	n.VotedFor = n.ID
	n.VoteCount = 1
	term := n.CurrentTerm
	_ = n.Storage.SaveState(n.CurrentTerm, n.VotedFor, n.Log)
	n.Mu.Unlock()

	slog.Info(fmt.Sprintf("[Node %d] Became Candidate (Term %d)", n.ID, term))

	for _, peer := range n.Peers {
		n.Network.Send(types.Message{
			From: n.ID,
			To:   peer,
			Type: "RequestVote",
			Term: term,
		})
	}
}

func (n *Node) becomeLeader() {
	n.Mu.Lock()
	n.State = types.Leader
	n.Mu.Unlock()

	slog.Info(fmt.Sprintf("[Node %d] Became Leader (Term %d)", n.ID, n.CurrentTerm))
	n.sendHeartbeats()
}

func (n *Node) sendHeartbeats() {
	go func() {
		for {
			n.Mu.Lock()
			if n.State != types.Leader {
				n.Mu.Unlock()
				return
			}
			term := n.CurrentTerm
			n.Mu.Unlock()

			for _, peer := range n.Peers {
				n.Network.Send(types.Message{
					From:         n.ID,
					To:           peer,
					Type:         "AppendEntries",
					Term:         term,
					Command:      "",
					PrevLogIndex: len(n.Log) - 1,
					PrevLogTerm: func() int {
						if len(n.Log) > 0 {
							return n.Log[len(n.Log)-1].Term
						}
						return -1
					}(),
				})
			}
			time.Sleep(150 * time.Millisecond)
		}
	}()
}

func (n *Node) ProposeCommand(cmd string) {
	n.Mu.Lock()
	if n.State != types.Leader {
		n.Mu.Unlock()
		slog.Info(fmt.Sprintf("[Node %d] Not leader, rejecting command: %s", n.ID, cmd))
		return
	}

	entry := types.LogEntry{Term: n.CurrentTerm, Command: cmd}
	n.Log = append(n.Log, entry)
	term := n.CurrentTerm
	_ = n.Storage.SaveState(n.CurrentTerm, n.VotedFor, n.Log)
	n.Mu.Unlock()

	slog.Info(fmt.Sprintf("[Node %d] Appended command to own log: %s", n.ID, cmd))

	for _, peer := range n.Peers {
		n.Network.Send(types.Message{
			From:         n.ID,
			To:           peer,
			Type:         "AppendEntries",
			Term:         term,
			Command:      cmd,
			PrevLogIndex: len(n.Log) - 2,
			PrevLogTerm: func() int {
				if len(n.Log) > 1 {
					return n.Log[len(n.Log)-2].Term
				}
				return -1
			}(),
			Entries: []types.LogEntry{entry},
		})
	}
}

func (n *Node) handleMessage(msg types.Message) {
	n.Mu.Lock()
	defer n.Mu.Unlock()

	switch msg.Type {
	case "RequestVote":
		if msg.Term > n.CurrentTerm {
			n.CurrentTerm = msg.Term
			n.State = types.Follower
			n.VotedFor = msg.From
			n.Network.Send(types.Message{
				From: n.ID,
				To:   msg.From,
				Type: "Vote",
				Term: n.CurrentTerm,
			})
			slog.Info(fmt.Sprintf("[Node %d] Voted for %d (Term %d)", n.ID, msg.From, n.CurrentTerm))
			_ = n.Storage.SaveState(n.CurrentTerm, n.VotedFor, n.Log)
		}

	case "Vote":
		if n.State == types.Candidate && msg.Term == n.CurrentTerm {
			n.VoteCount++
			if n.VoteCount > len(n.Peers)/2 {
				go n.becomeLeader()
			}
		}

	case "AppendEntries":
		if msg.Term >= n.CurrentTerm {
			n.CurrentTerm = msg.Term
			n.State = types.Follower

			if msg.PrevLogIndex >= 0 {
				if msg.PrevLogIndex >= len(n.Log) || n.Log[msg.PrevLogIndex].Term != msg.PrevLogTerm {
					return
				}
			}

			for _, entry := range msg.Entries {
				if len(n.Log) > msg.PrevLogIndex+1 {
					n.Log = n.Log[:msg.PrevLogIndex+1]
				}
				n.Log = append(n.Log, entry)
				slog.Info(fmt.Sprintf("[Node %d] Appended entry: %v", n.ID, entry))
			}

			if msg.CommitIndex > n.CommitIndex {
				lastIndex := len(n.Log) - 1
				if msg.CommitIndex < lastIndex {
					n.CommitIndex = msg.CommitIndex
				} else {
					n.CommitIndex = lastIndex
				}
			}
			_ = n.Storage.SaveState(n.CurrentTerm, n.VotedFor, n.Log)
			n.Network.Send(types.Message{
				From:        n.ID,
				To:          msg.From,
				Type:        "AppendEntriesResponse",
				Term:        n.CurrentTerm,
				CommitIndex: n.CommitIndex,
			})
		}
	case "AppendEntriesResponse":
		n.AckCount[msg.CommitIndex]++
		if n.AckCount[msg.CommitIndex] > len(n.Peers)/2 && n.CommitIndex < msg.CommitIndex {
			n.CommitIndex = msg.CommitIndex
			slog.Info(fmt.Sprintf("[Node %d] Committed index %d", n.ID, msg.CommitIndex))
			_ = n.Storage.SaveState(n.CurrentTerm, n.VotedFor, n.Log)
		}
	}
}

func (n *Node) Stop() {
	n.Network.Unregister(n.ID)
	close(n.MsgCh)
	slog.Info("Node crashed!", "Node", n.ID)
}

func randomTimeout() time.Duration {
	return time.Duration(250+rand.Intn(150)) * time.Millisecond
}
