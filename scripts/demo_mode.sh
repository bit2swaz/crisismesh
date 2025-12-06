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

# Get all IPs
IPS=$(hostname -I)

echo "---------------------------------------------------"
echo "CRISISMESH DEMO MODE (Headless/WSL Friendly)"
echo "---------------------------------------------------"
echo "📱 MOBILE ACCESS INSTRUCTIONS:"
echo "To access the Civilian interface from your phone,"
echo "ensure you are on the same Wi-Fi and try these links:"
echo ""
for ip in $IPS; do
    echo "  👉 http://$ip:8081"
done
echo ""
echo "⚠️  TROUBLESHOOTING:"
echo "1. If on WSL (Windows), these 172.x.x.x IPs are internal."
echo "   You must use your Windows PC's actual LAN IP (e.g., 192.168.x.x)."
echo "   And ensure you have allowed the port in Windows Firewall."
echo "---------------------------------------------------"

# Start Node 1 (Commander)
echo "🚀 Starting Node 1 (Commander)..."
echo "   Gossip: 9000 | Web: 8080"
./crisis start --nick Commander --port 9000 --web-port 8080 > commander.log 2>&1 &

# Start Node 2 (Civilian)
echo "🚀 Starting Node 2 (Civilian)..."
echo "   Gossip: 9001 | Web: 8081"
./crisis start --nick Civilian --port 9001 --web-port 8081 > civilian.log 2>&1 &

echo ""
echo "✅ Nodes are running in the background."
echo "📄 Logs are being written to 'commander.log' and 'civilian.log'"
echo "   View logs with: tail -f commander.log"
echo ""
echo "Press Ctrl+C to stop everything."

# Wait indefinitely so the script doesn't exit
wait