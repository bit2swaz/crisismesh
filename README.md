# CrisisMesh

**Offline, Self-Forming Emergency Communication Network**

![Architecture](untitled-2025-11-20-2204.png)

CrisisMesh is a decentralized, offline-first messaging system designed for disaster and low-connectivity scenarios. Devices on the same local network (Wi-Fi/hotspot/ethernet) form a self-organizing mesh. Messages are delivered hop-by-hop using a store-and-forward (Delay Tolerant Networking, DTN) approach, enabling eventual delivery even when the sender and recipient are never directly connected at the same time.

## Features

*   **Decentralized Mesh**: No central server required. Peers discover each other automatically.
*   **Store-and-Forward**: Messages are stored locally and forwarded when a path to the destination becomes available.
*   **Offline-First**: Designed to work without Internet access.
*   **Peer Discovery**: Automatic discovery via UDP broadcast (ports 9000-9005).
*   **Terminal UI (TUI)**: Interactive "hacker terminal" interface for sending messages and viewing network status.
*   **"I AM SAFE" Broadcast**: High-priority broadcast to notify all nearby peers of safety status.
*   **Persistence**: SQLite-backed storage ensures messages survive restarts.

## Architecture

The system consists of a single Go binary acting as an agent.

*   **Discovery**: Listens for UDP heartbeats to find peers on the LAN.
*   **Transport**: TCP-based transport for reliable message exchange.
*   **Gossip Engine**: Manages the synchronization of message inventories and requests missing messages.
*   **Storage**: SQLite database stores messages, peer tables, and deduplication records.
*   **TUI**: Bubble Tea-based interface for user interaction.

## Getting Started

### Prerequisites

*   Go 1.21 or higher
*   GCC (for SQLite CGO)

### Installation

Clone the repository and build the binary:

```bash
git clone https://github.com/bit2swaz/crisismesh.git
cd crisismesh
go mod download
go build -o crisis cmd/crisis/main.go
```

### Running the Agent

To start a node, use the `start` command. You must provide a unique nickname.

```bash
./crisis start --nick Alice --port 9000
```

**Flags:**
*   `--nick`: Your display name (required).
*   `--port`: The TCP port to listen on (default: 9000).
*   `--room`: The mesh room name (default: "lobby"). Only peers in the same room will connect.

## Demo Scenario (3-Node Mesh)

To simulate a mesh network on a single machine, you can run multiple agents on different ports.

**Terminal 1 (Alice):**
```bash
./crisis start --nick Alice --port 9000
```

**Terminal 2 (Bob):**
```bash
./crisis start --nick Bob --port 9001
```

**Terminal 3 (Charlie):**
```bash
./crisis start --nick Charlie --port 9002
```

### Testing Multi-Hop Delivery

1.  Start Alice, Bob, and Charlie.
2.  Wait for them to discover each other (they will appear in the "Peers" list in the TUI).
3.  **Simulate a partition**: Stop Bob (Ctrl+C).
4.  From Alice, send a message to Bob. The message will be queued.
5.  Restart Bob.
6.  Alice's agent will detect Bob and deliver the queued message.

*(Note: True multi-hop requires network-level partitioning, which can be simulated by blocking direct ports between specific nodes, but the store-and-forward behavior can be observed by stopping/starting nodes).*

## Usage (TUI)

Once the agent is running, you will see the TUI:

*   **Type a message**: Enter text in the input box and press Enter to broadcast to all peers.
*   **Commands**:
    *   `/connect <ip:port>`: Manually connect to a peer (useful if UDP broadcast is blocked).
    *   `/safe`: Broadcast a high-priority "I AM SAFE" status.
*   **View**: The main area shows the chat log. The sidebar shows active peers.

## Future Roadmap

*   **Encryption**: End-to-end encryption using X25519.
*   **Mobile Apps**: Android/iOS wrappers for the Go agent.
*   **Smart Routing**: Optimize message paths based on hop count and link quality.
*   **Attachments**: Support for images and small files.

## License

MIT
