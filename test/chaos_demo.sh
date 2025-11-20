#!/bin/bash
SESSION="crisis_chaos"
BIN="./crisis"
LOG_DIR="test_logs"
cleanup() {
    echo "Cleaning up..."
    tmux kill-session -t $SESSION 2>/dev/null
    pkill -f "$BIN start" || true
}
trap cleanup EXIT
if ! command -v tmux &> /dev/null; then
    echo "Error: tmux is required for this test."
    exit 1
fi
echo "Building CrisisMesh..."
go build -o $BIN ./cmd/crisis
mkdir -p $LOG_DIR
rm -f $LOG_DIR/*.log
rm -f *.db *.db-shm *.db-wal identity_*.json
tmux kill-session -t $SESSION 2>/dev/null
tmux new-session -d -s $SESSION
echo "Starting nodes..."
tmux rename-window -t $SESSION:0 'alice'
tmux send-keys -t $SESSION:alice "$BIN start --nick Alice --port 9001" C-m
tmux new-window -t $SESSION -n 'bob'
tmux send-keys -t $SESSION:bob "$BIN start --nick Bob --port 9002" C-m
tmux new-window -t $SESSION -n 'charlie'
tmux send-keys -t $SESSION:charlie "$BIN start --nick Charlie --port 9003" C-m
echo "Waiting 5s for mesh to stabilize..."
sleep 5
echo "Killing Bob (Simulating partition)..."
BOB_PID=$(pgrep -f "nick Bob")
if [ -z "$BOB_PID" ]; then
    echo "Error: Bob process not found!"
    exit 1
fi
kill -9 $BOB_PID
echo "Bob is dead."
echo "Alice broadcasting 'Target Charlie'..."
tmux send-keys -t $SESSION:alice "Target Charlie" C-m
echo "Waiting 2s..."
sleep 2
echo "Resurrecting Bob..."
tmux send-keys -t $SESSION:bob "$BIN start --nick Bob --port 9002" C-m
echo "Waiting 15s for gossip propagation..."
sleep 15
echo "Capturing Charlie's logs..."
tmux capture-pane -p -t $SESSION:charlie > $LOG_DIR/charlie.log
if grep -a "Target Charlie" $LOG_DIR/charlie.log; then
    echo "----------------------------------------"
    echo "SUCCESS: Message survived partition!"
    echo "----------------------------------------"
    exit 0
else
    echo "----------------------------------------"
    echo "FAIL: Message not found in Charlie's logs."
    echo "----------------------------------------"
    echo "Tail of Charlie's log:"
    tail -n 20 $LOG_DIR/charlie.log
    cleanup
    exit 1
fi
