package storage

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/itsshashank/consensus/types"
)

type FileStorage struct {
	mu   sync.Mutex
	file string
}

type PersistentState struct {
	Term     int
	VotedFor int
	Log      []types.LogEntry
}

func NewFileStorage(path string) *FileStorage {
	return &FileStorage{file: path}
}

func (fs *FileStorage) SaveState(term int, votedFor int, log []types.LogEntry) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	state := PersistentState{
		Term:     term,
		VotedFor: votedFor,
		Log:      log,
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(fs.file, data, 0644)
}

func (fs *FileStorage) LoadState() (int, int, []types.LogEntry, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, err := os.ReadFile(fs.file)
	if os.IsNotExist(err) {
		return 0, -1, []types.LogEntry{}, nil // default fresh state
	}
	if err != nil {
		return 0, -1, nil, err
	}

	var state PersistentState
	if err := json.Unmarshal(data, &state); err != nil {
		return 0, -1, nil, err
	}
	return state.Term, state.VotedFor, state.Log, nil
}
