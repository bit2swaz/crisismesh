package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bit2swaz/crisismesh/internal/core"
	"github.com/bit2swaz/crisismesh/internal/engine"
	"github.com/bit2swaz/crisismesh/internal/store"
	"github.com/bit2swaz/crisismesh/internal/transport"
)

func main() {
	port := 9002
	target := "127.0.0.1:9007"
	nick := "TestBot"

	// 1. Setup
	dbPath := fmt.Sprintf("crisis_%d.db", port)
	db, err := store.Init(dbPath)
	if err != nil {
		log.Fatalf("Failed to init DB: %v", err)
	}

	identityPath := fmt.Sprintf("identity_%d.json", port)
	id, err := core.LoadOrGenerateIdentity(identityPath)
	if err != nil {
		log.Fatalf("Failed to load identity: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tm := transport.NewManager()
	eng := engine.NewGossipEngine(db, tm, id.NodeID, nick, port, id.PubKey, id.PrivKey)

	// 2. Start Engine
	fmt.Printf("Bot starting on port %d...\n", port)
	if err := eng.Start(ctx); err != nil {
		log.Fatalf("Failed to start engine: %v", err)
	}

	// 3. Connect to Target
	time.Sleep(2 * time.Second)
	fmt.Printf("Connecting to %s...\n", target)
	if err := eng.ManualConnect(target); err != nil {
		log.Printf("Failed to connect (is the main app running?): %v", err)
		return
	}

	// 4. Send Message (Triggers Flash)
	time.Sleep(2 * time.Second)
	msg := "Hello! I am a bot. This message should FLASH."
	fmt.Printf("Sending message: %q\n", msg)
	if err := eng.PublishText(msg); err != nil {
		log.Printf("Failed to publish: %v", err)
	}

	// 5. Wait (Triggers Active Sort)
	fmt.Println("Staying online for 10 seconds (Check 'TestBot' is at top of list)...")
	time.Sleep(10 * time.Second)

	// 6. Exit (Triggers Inactive Sort)
	fmt.Println("Bot shutting down (Check 'TestBot' moves down/goes inactive)...")
	os.Exit(0)
}
