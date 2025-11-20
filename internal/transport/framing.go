package transport

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

// WriteFrame writes a length-prefixed frame to the connection.
// Format: [4 bytes length][payload]
func WriteFrame(conn net.Conn, data []byte) error {
	length := uint32(len(data))
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, length)

	// Write header
	if _, err := conn.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write payload
	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("failed to write payload: %w", err)
	}

	return nil
}

// ReadFrame reads a length-prefixed frame from the connection.
func ReadFrame(conn net.Conn) ([]byte, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	length := binary.BigEndian.Uint32(header)

	// Sanity check for max frame size (e.g., 10MB) to prevent OOM attacks
	if length > 10*1024*1024 {
		return nil, fmt.Errorf("frame too large: %d bytes", length)
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, fmt.Errorf("failed to read payload: %w", err)
	}

	return payload, nil
}
