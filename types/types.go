package types

type State int

const (
	Follower State = iota
	Candidate
	Leader
)

type Message struct {
	From         int
	To           int
	Type         string
	Term         int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	CommitIndex  int
	Command      string
}

type LogEntry struct {
	Term    int
	Command string
}
