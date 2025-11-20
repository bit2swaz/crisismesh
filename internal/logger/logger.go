package logger
import (
	"log/slog"
	"os"
)
func Init() error {
	file, err := os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewTextHandler(file, nil))
	slog.SetDefault(logger)
	return nil
}
