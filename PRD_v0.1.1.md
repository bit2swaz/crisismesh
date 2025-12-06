# CrisisMesh - Product Requirements Document (v0.1.1)

**Date:** December 6, 2025  
**Version:** 0.1.1 (Secure Mobile-Ready Beta)  
**Status:** Implemented & Tested  
**Repository:** `github.com/bit2swaz/crisismesh`  
**Authors:** Bit2Swaz & GitHub Copilot  

---

## 1. Executive Summary

**CrisisMesh** is a decentralized, offline-first emergency communication system designed for scenarios where traditional internet and cellular infrastructure is compromised (e.g., natural disasters, censorship, remote operations). It enables devices on a local network (LAN/Wi-Fi/Hotspot) to form a self-organizing mesh network.

The system utilizes a **Store-and-Forward** (Delay Tolerant Networking - DTN) architecture. Messages are stored locally on each device and propagated via a gossip protocol to peers when they become available, ensuring eventual delivery even if the sender and recipient are never online simultaneously.

**Version 0.1.1** introduces critical enhancements:
*   **End-to-End Encryption (E2EE):** Secure Direct Messages (DMs) using Curve25519.
*   **Mobile-First Web Interface:** A responsive web UI for accessing the mesh via smartphones.
*   **Network Visualization:** A real-time graph of the mesh topology.
*   **Advanced TUI:** A polished, cyberpunk-themed terminal interface with visual alerts.

---

## 2. Why We Made This

In the modern world, we rely heavily on centralized infrastructure (ISPs, Cell Towers, Cloud Servers). When these fail, communication collapses. 

**CrisisMesh** was built to answer the question: *"How do we talk when the lights go out?"*

### Core Philosophy
1.  **Zero Config:** It should just work. No servers to set up, no IP addresses to configure manually.
2.  **Offline First:** The network *is* the database. If you have the data, you have the app.
3.  **Resilient:** Nodes can join and leave at will. The network heals itself.
4.  **Secure:** Privacy is not optional, especially in crisis scenarios.

---

## 3. System Architecture

The system runs as a single executable binary (`crisis`) that integrates networking, storage, TUI, and a Web Server.

### 3.1. Component Diagram

```mermaid
graph TD
    UserTUI[User (Terminal)] <--> TUI[Bubble Tea Interface]
    UserWeb[User (Mobile/Browser)] <--> Web[HTTP Server]
    
    subgraph "CrisisMesh Agent"
        TUI --> Engine
        Web --> Engine
        
        Engine[Gossip Engine] <--> Store[(SQLite DB)]
        Engine <--> Crypto[NaCl Box Crypto]
        
        Engine <--> Discovery[UDP Discovery]
        Engine <--> Transport[TCP Transport]
    end

    Discovery <--> LAN((Local Network))
    Transport <--> LAN
```

### 3.2. Data Flow (The Gossip Loop)

1.  **Discovery (UDP):** Nodes broadcast "Heartbeats" (ID, Nick, Port, **Public Key**) every second.
2.  **Handshake:** Nodes update their local `peers` table with the received info.
3.  **Sync (TCP):** Every 5 seconds, a node picks a random peer and exchanges a list of Message IDs.
4.  **Request (TCP):** If Node A has messages Node B is missing, Node B requests them.
5.  **Transfer:** Messages are transferred. If encrypted, they remain encrypted on the wire.
6.  **Storage:** Messages are saved to the local SQLite database (WAL mode).
7.  **Notification:** The TUI flashes, and the Web UI updates via HTMX.

---

## 4. Technical Specifications

### 4.1. Security & Encryption (New in v0.1.1)
*   **Library:** `golang.org/x/crypto/nacl/box` (Curve25519, XSalsa20, Poly1305).
*   **Identity:** 
    *   On startup, the agent generates a Curve25519 Keypair.
    *   Keys are persisted in `identity_<port>.json`.
*   **Key Exchange:** Public Keys are broadcast in the UDP Heartbeat payload.
*   **Encryption Flow:**
    1.  User types `/dm Bob Secret`.
    2.  Agent looks up Bob's Public Key.
    3.  Agent encrypts "Secret" using `box.SealAnonymous` (Ephemeral Sender Key + Bob's Public Key).
    4.  Ciphertext is sent over the wire.
    5.  Bob receives ciphertext, decrypts with his Private Key using `box.OpenAnonymous`.
    6.  Plaintext is stored in Bob's DB (for display).

### 4.2. Web Interface (New in v0.1.1)
*   **Stack:** Go `net/http`, `html/template`, HTMX, Tailwind CSS (CDN), Vis.js.
*   **Endpoints:**
    *   `/`: Main Chat Interface (Mobile Optimized).
    *   `/map`: Network Topology Visualization.
    *   `/api/messages`: JSON endpoint for message history.
    *   `/api/graph`: JSON endpoint for node/link data.
*   **Features:**
    *   **Responsive:** Works on iOS/Android browsers.
    *   **Real-time:** HTMX polling updates chat without full reloads.
    *   **Visuals:** Interactive physics-based graph of the mesh.

### 4.3. Terminal User Interface (TUI)
*   **Framework:** Bubble Tea + Lipgloss.
*   **Features:**
    *   **Tabs:** `F1` (Comms), `F2` (Network), `F3` (Guide).
    *   **Visual Alerts:** Screen flashes on new messages (<500ms).
    *   **Status Bar:** Real-time peer count and spinner.
    *   **Help:** Context-aware keybinding help (`?`).

### 4.4. Storage Layer
*   **Database:** SQLite (Embedded).
*   **Mode:** Write-Ahead Logging (WAL) enabled for concurrency.
*   **Schema:**
    *   `peers`: ID, Nick, Addr, **PubKey**, LastSeen, IsActive.
    *   `messages`: ID, Sender, Recipient, Content, Timestamp, **IsEncrypted**.

---

## 5. How to Use

### 5.1. Installation
```bash
# Clone the repo
git clone https://github.com/bit2swaz/crisismesh.git
cd crisismesh

# Build
go build -o crisis ./cmd/crisis
```

### 5.2. Running a Node
```bash
# Start a node (Default Port 9000)
./crisis start --nick Alice

# Start a second node (Port 9001)
./crisis start --nick Bob --port 9001
```

### 5.3. Commands
| Command | Description | Example |
| :--- | :--- | :--- |
| `/connect <ip:port>` | Manually connect to a peer | `/connect 192.168.1.5:9000` |
| `/dm <nick> <msg>` | Send an **Encrypted** Direct Message | `/dm Bob The eagle has landed` |
| `/safe` | Broadcast a high-priority SAFE status | `/safe` |
| `Ctrl+C` | Quit | |
| `F1-F3` | Switch Tabs | |

### 5.4. Accessing the Web UI
Open your browser (or mobile phone on the same Wi-Fi) and navigate to:
`http://<YOUR_PC_IP>:8080` (for the first node)
`http://<YOUR_PC_IP>:8081` (for the second node, etc.)

---

## 6. Testing Strategy

### 6.1. Automated Tests
We maintain a suite of unit tests covering core logic:
*   **Crypto:** Verifies Key Generation and Encryption/Decryption cycles.
*   **Engine:** Verifies Gossip propagation and Sync logic.
*   **TUI:** Verifies visual logic (Flash timing, Peer sorting).

Run tests with:
```bash
go test -v ./...
```

### 6.2. Manual Verification (The "Bot")
A helper script spins up a bot to simulate a peer joining, sending a message, and leaving.
```bash
./scripts/run_bot.sh
```

---

## 7. What's Next (Roadmap)

*   **File Sharing:** Chunking and transferring images/files over the mesh.
*   **Mesh Routing:** Multi-hop routing for DMs (currently DMs require direct gossip propagation, though the data floods, the decryption is point-to-point).
*   **Android App:** Wrapping the Web UI in a native WebView or building a Flutter client.
*   **LoRa Integration:** Support for 915MHz hardware radios for long-range, off-grid comms.

---

## 8. License
MIT License. Free to use, modify, and distribute for humanitarian and educational purposes.
