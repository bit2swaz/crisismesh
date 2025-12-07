package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/bit2swaz/crisismesh/internal/logger"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			// Restore terminal if it was in raw mode
			fmt.Print("\033[?25h") // Show cursor

			// Log the panic
			slog.Error("CRITICAL PANIC", "error", r)

			// Print to stderr
			fmt.Fprintf(os.Stderr, "\n\nCRITICAL ERROR: %v\n", r)
			fmt.Fprintf(os.Stderr, "Check debug.log for details.\n")
			os.Exit(1)
		}
	}()

	if err := logger.Init(); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	Execute()
}
