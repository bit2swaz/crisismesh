package logger

import (
	"log/slog"
	"os"
)

// Init configures the default slog logger to write to debug.log.
// It ensures no logs are written to stdout.
func Init() error {
	file, err := os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	// Create a new logger that writes to the file
	logger := slog.New(slog.NewTextHandler(file, nil))

	// Set it as the default logger
	slog.SetDefault(logger)

	return nil
}
