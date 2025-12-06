#!/bin/bash

# Cleanup function to kill background processes on exit
cleanup() {
    echo ""
    echo "🛑 Stopping CrisisMesh nodes..."
    # Kill all child processes of this script
    pkill -P $$
    exit
}
trap cleanup SIGINT SIGTERM

# Kill any existing instances
pkill -f "crisis start"

# Build the binary
echo "🔨 Building CrisisMesh..."
go build -o crisis ./cmd/crisis

# Get IP for display
IP=$(hostname -I | awk '{print $1}')

echo "---------------------------------------------------"
echo "CRISISMESH TUI DEMO"
echo "---------------------------------------------------"
echo "1. RIGHT SCREEN (Civilian):"
echo "   Open Chrome and go to: http://localhost:8081"
echo "   (Or http://$IP:8081 if using mobile)"
echo "   * Enable Mobile Emulation Mode in DevTools *"
echo ""
echo "2. LEFT SCREEN (Commander):"
echo "   The TUI will launch below."
echo "---------------------------------------------------"
echo "Starting Civilian Node (Bob) in background..."

# Start Node 2 (Civilian) - HEADLESS
# We use CRISIS_HEADLESS=true so it doesn't try to take over the terminal
export CRISIS_HEADLESS=true
./crisis start --nick Civilian --port 9001 --web-port 8081 > civilian.log 2>&1 &
CIVILIAN_PID=$!

# Wait a moment for Bob to start
sleep 2

echo "Starting Commander Node (Alice) in foreground..."
echo "Press Ctrl+C to exit both nodes."
sleep 2

# Unset HEADLESS for Alice so she gets the TUI
unset CRISIS_HEADLESS

# Start Node 1 (Commander) - TUI
./crisis start --nick Commander --port 9000 --web-port 8080

# When Alice exits, the trap will catch it and kill Bob
