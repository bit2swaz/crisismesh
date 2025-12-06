#!/bin/bash
set -e

echo "Building Bot..."
go build -o bin/bot ./cmd/bot

echo "---------------------------------------------------"
echo "MANUAL TUI TEST HELPER"
echo "---------------------------------------------------"
echo "1. Ensure your main CrisisMesh app is running in another terminal:"
echo "   go run ./cmd/crisis start --nick User --port 9007"
echo ""
echo "2. Press ENTER to start the Bot."
echo "   The bot will:"
echo "   - Connect to your node."
echo "   - Send a message (Test FLASH)."
echo "   - Stay online for 10s (Test SORTING - Active)."
echo "   - Disconnect (Test SORTING - Inactive)."
echo "---------------------------------------------------"
read -p "Press ENTER when ready..."

./bin/bot
