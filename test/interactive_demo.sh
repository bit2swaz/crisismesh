#!/bin/bash

SESSION="crisis_interactive"
BIN="./crisis"

# Cleanup function
cleanup() {
    # We don't kill the session on exit because we want to attach to it.
    # But we should ensure we don't have lingering processes if we fail before attach.
    return
}

if ! command -v tmux &> /dev/null; then
    echo "Error: tmux is required for this test."
    exit 1
fi

echo "Building CrisisMesh..."
go build -o $BIN ./cmd/crisis

echo "Cleaning up old data..."
rm -f *.db *.db-shm *.db-wal identity_*.json

# Kill existing session if it exists
tmux kill-session -t $SESSION 2>/dev/null

# Create new session
echo "Starting 3-node mesh in tmux..."
tmux new-session -d -s $SESSION

# Configure tmux for this session
# 1. Enable mouse support (clickable windows/panes)
tmux set-option -t $SESSION mouse on
# 2. Change prefix to Ctrl+A to avoid VS Code conflict (Ctrl+B toggles sidebar)
tmux set-option -t $SESSION prefix C-a
tmux unbind-key -t $SESSION C-b
tmux bind-key -t $SESSION C-a send-prefix

# Window 0: Alice
tmux rename-window -t $SESSION:0 'Alice'
tmux send-keys -t $SESSION:Alice "$BIN start --nick Alice --port 9000" C-m

# Window 1: Bob
tmux new-window -t $SESSION -n 'Bob'
tmux send-keys -t $SESSION:Bob "$BIN start --nick Bob --port 9001" C-m

# Window 2: Charlie
tmux new-window -t $SESSION -n 'Charlie'
tmux send-keys -t $SESSION:Charlie "$BIN start --nick Charlie --port 9002" C-m

# Select Alice window
tmux select-window -t $SESSION:Alice

echo "---------------------------------------------------"
echo "Interactive Demo Started!"
echo "---------------------------------------------------"
echo "IMPORTANT: Prefix key changed to Ctrl+A (to avoid VS Code conflict)"
echo "Commands:"
echo "  - Switch windows: Ctrl+A then 0, 1, or 2"
echo "  - OR CLICK the window names at the bottom with your mouse"
echo "  - Quit app: Ctrl+C in each window"
echo "  - Detach session: Ctrl+A then d"
echo "---------------------------------------------------"
echo "Attaching to session..."

# Attach to the session
tmux attach-session -t $SESSION
