# consensus

A minimal Raft-style consensus algorithm implemented in Go.

## Architecture Overview

Each `Node` in the system can be in one of three states:
- `Follower`
- `Candidate`
- `Leader`

Nodes communicate using channels and a simulated `Network` layer. The leader replicates log entries to the followers. Consensus is reached once a majority of nodes acknowledge the same log entry.

Key components:
- `Node`: Maintains its term, log, commit index, and handles messages.
- `Network`: Simulates message delivery between nodes.
- `LogEntry`: Commands appended to the replicated log.
- `Election`: Triggered by timeouts, candidates request votes.
- `AppendEntries`: Sent by leaders to replicate logs and maintain authority.

## Building and Running

```bash
git clone https://github.com/yourusername/consensus.git
cd consensus
go run cmd/main.go
```
### Logs
```bash
[Node 1] Starting up
[Node 2] Starting up
[Node 3] Starting up
[Node 1] Election timeout. Starting election.
[Node 1] Became Candidate (Term 1)
[Node 2] Voted for 1 (Term 1)
[Node 3] Voted for 1 (Term 1)
[Node 1] Became Leader (Term 1)
[Node 1] Appended command to own log: set x=1
[Node 2] Appended entry: {Term:1 Command:set x=1}
[Node 3] Appended entry: {Term:1 Command:set x=1}
[Node 1] Committed index 0
```