# CrisisMesh - Product Requirements Document (v0.1.0)

**Date:** November 20, 2025
**Version:** 0.1.0 (Hackathon MVP)
**Status:** Implemented
**Repository:** `github.com/bit2swaz/crisismesh`

---

## 1. Executive Summary

**CrisisMesh** is a decentralized, offline-first emergency communication system designed for disaster scenarios where traditional internet and cellular infrastructure is compromised. It enables devices on a local network (LAN/Wi-Fi/Hotspot) to form a self-organizing mesh network.

The system uses a **Store-and-Forward** (Delay Tolerant Networking - DTN) approach. Messages are stored locally on each device and forwarded to peers when they become available, ensuring eventual delivery even if the sender and recipient are never online simultaneously.

The MVP is a **Go-based CLI agent** with a "Cyberpunk/Military" Terminal User Interface (TUI), focusing on reliability, ease of deployment, and clear visualization of mesh mechanics.

---

## 2. System Architecture

The system runs as a single executable binary (`crisis`) that integrates networking, storage, and UI components.

### 2.1. High-Level Data Flow

1.  **Discovery**: Agent broadcasts UDP heartbeats. Peers listen and update their local `peers` table.
2.  **Connection**: Agent establishes TCP connections to discovered peers.
3.  **Gossip (Sync)**: Periodically, Agent A sends a list of known Message IDs to Agent B.
4.  **Request**: Agent B identifies missing messages and requests them from Agent A.
5.  **Transfer**: Agent A sends the full message payloads.
6.  **Storage**: Agent B persists messages to SQLite.
7.  **Presentation**: The TUI updates to show new messages or peer status.

### 2.2. Component Diagram

```mermaid
graph TD
    User[User (TUI)] <--> Agent
    Agent <--> SQLite[(SQLite DB)]
    Agent <--> Network((LAN / Mesh))

    subgraph Agent Components
        Discovery[Discovery Service (UDP)]
        Transport[Transport Manager (TCP)]
        Gossip[Gossip Engine]
        Store[Storage Layer]
        UI[Bubble Tea TUI]
    end

    Discovery --> Store
    Gossip --> Transport
    Gossip --> Store
    UI --> Gossip
    UI --> Store
```

---

## 3. Technical Specifications

### 3.1. Discovery Layer
*   **Mechanism**: UDP Broadcast.
*   **Ports**: Broadcasts to `255.255.255.255` on ports `9000` through `9005` to ensure discovery across multiple local instances.
*   **Heartbeat Frequency**: 1 second.
*   **Packet Structure**:
    ```json
    {
      "type": "HEARTBEAT",
      "id": "node-uuid-v4",
      "nick": "Alice",
      "port": 9000,
      "ts": 1700000000
    }
    ```
*   **Peer Liveness**: Peers are marked `IsActive=true` on heartbeat. A background "Reaper" process marks them `IsActive=false` if no heartbeat is received for >10 seconds.

### 3.2. Transport Layer
*   **Protocol**: TCP.
*   **Framing**: Length-Prefixed JSON.
    *   **Header**: 4 bytes (Big Endian uint32) indicating payload length.
    *   **Payload**: JSON byte array.
    *   **Max Frame Size**: 10 MB.
*   **Connection Management**: Connections are ephemeral for the MVP; established for gossip exchange and then closed, or kept alive depending on the `TransportManager` state (currently ephemeral for simplicity).

### 3.3. Application Protocol
*   **Packet Types**:
    *   `SYNC`: Contains a list of Message IDs held by the sender.
    *   `REQ`: Request for specific Message IDs.
    *   `MSG`: Delivery of a full `Message` object.
    *   `SAFE`: High-priority status broadcast.
*   **Gossip Interval**: Every 5 seconds, the agent picks a random active peer to sync with.

### 3.4. Storage Layer
*   **Database**: SQLite (embedded via CGO).
*   **ORM**: GORM.
*   **Schema**:
    *   **`peers` Table**:
        *   `id` (PK, string): Node UUID.
        *   `nick` (string): Display name.
        *   `addr` (string): IP:Port.
        *   `last_seen` (datetime): Timestamp of last heartbeat.
        *   `is_active` (bool): Online status.
    *   **`messages` Table**:
        *   `id` (PK, string): Unique hash.
        *   `sender_id` (string): Originator UUID.
        *   `recipient_id` (string): Target UUID or "BROADCAST".
        *   `content` (string): Text payload.
        *   `priority` (int): 1 (Normal), 2 (High/SAFE).
        *   `timestamp` (int64): Creation time.
        *   `ttl` (int): Time-To-Live (hops remaining).
        *   `hop_count` (int): Number of hops traversed.
        *   `status` (string): Delivery status (e.g., "stored").

---

## 4. User Interface (TUI) Specification

The UI is built with **Bubble Tea** and **Lipgloss**, featuring a "Cyberpunk/Military" aesthetic (Green on Black).

### 4.1. Visual Style
*   **Primary Color**: Green (`#00FF00` / ANSI 2).
*   **Background**: Black (`#000000` / ANSI 0).
*   **Alert Color**: Red (`#FF0000` / ANSI 196).
*   **Font**: Monospaced (Terminal default).

### 4.2. Layout Components
1.  **Header (HUD)**:
    *   **Logo**: ASCII Art `CRISISMESH` (Left aligned).
    *   **Stats**: Node ID (truncated), Live Clock (HH:MM:SS), Uptime (Right aligned).
    *   **Style**: Bordered, Green foreground.

2.  **Tab Bar**:
    *   Displays available views: `COMMs`, `NETWORK`, `GUIDE`.
    *   **Active Tab**: Highlighted with bold text and green border.
    *   **Inactive Tab**: Gray text.

3.  **Main Content Area**:
    *   **COMMs View**: Scrollable viewport of chat history.
    *   **NETWORK View**: Table of peers (`PEER ID`, `NICK`, `LAST SEEN`).
    *   **GUIDE View**: Static help text.

4.  **Status Bar (Footer)**:
    *   **Left**: Connection Status (`ONLINE` if peers > 0, else `ISOLATED`).
    *   **Right**: System Stats (`RAM: 12MB`, `TAB: Switch`).
    *   **Style**: Green background, Black text (Inverted).

5.  **Input Bar**:
    *   Standard text input field at the bottom.
    *   Placeholder: "Type a message...".

### 4.3. Interaction & Keybindings
*   **Navigation**:
    *   `Tab`: Cycle forward through tabs (COMMs -> NETWORK -> GUIDE -> COMMs).
    *   `Shift+Tab`: Cycle backward through tabs.
*   **Messaging**:
    *   `Enter`: Send message or execute command.
*   **System**:
    *   `Ctrl+C` / `Esc`: Quit application.

### 4.4. Slash Commands
| Command | Arguments | Description |
| :--- | :--- | :--- |
| `/connect` | `<ip:port>` | Manually connect to a peer (bypasses UDP discovery). |
| `/safe` | None | Broadcasts a high-priority "I AM SAFE" message. |

### 4.5. Visual Alerts
*   **SAFE Broadcast**: When a message with `Priority=2` is received, the entire TUI border flashes red for 1 second (10 ticks of 100ms).

---

## 5. CLI Reference

### 5.1. Usage
```bash
./crisis start [flags]
```

### 5.2. Flags
| Flag | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `--nick` | string | (Required) | Display name for the user. |
| `--port` | int | `9000` | TCP port to listen on. |
| `--room` | string | `lobby` | Mesh partition key (peers must match). |

---

## 6. Security & Privacy

### 6.1. Current State (MVP)
*   **Transport Security**: Plaintext TCP. No encryption.
*   **Identity**: Self-generated UUIDs. No cryptographic verification.
*   **Privacy**: Messages are stored in plaintext in the local SQLite database.

### 6.2. Risks
*   **Spoofing**: A malicious actor can impersonate a Node ID.
*   **Eavesdropping**: Anyone on the LAN can capture TCP packets.
*   **Tampering**: Messages can be modified in transit.

### 6.3. Future Mitigation (Roadmap)
*   **Encryption**: Implement X25519 key exchange for End-to-End Encryption (E2EE).
*   **Signing**: Sign messages with Ed25519 keys to ensure integrity and non-repudiation.
*   **Trust**: Implement a "Web of Trust" or QR-code based key verification.

---

## 7. Performance & Constraints

*   **Max Message Size**: ~10 MB (enforced by transport framing).
*   **Max Peers**: Tested with ~10 peers. UDP broadcast storms may occur with >50 peers.
*   **Storage**: Limited only by disk space. SQLite handles thousands of messages efficiently.
*   **Network**: Requires an open TCP port and UDP broadcast capability on the LAN.

---

## 8. Future Roadmap

*   **Encryption**: End-to-End encryption using X25519.
*   **Mobile Apps**: Android/iOS wrappers for the Go agent.
*   **Smart Routing**: Optimize message paths based on hop count and link quality (Dijkstra/A*).
*   **Attachments**: Support for images and small files.
*   **Mesh Visualization**: Web-based "God View" for demo purposes.
