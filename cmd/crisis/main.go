package main

import (
	"log"

	"github.com/bit2swaz/crisismesh/internal/logger"
)

func main() {
	// Initialize file logger immediately
	if err := logger.Init(); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	Execute()
}
