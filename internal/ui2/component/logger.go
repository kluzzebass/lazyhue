package component

import (
	"io"
	"log/slog"
	"os"
)

// InitLogger configures slog to write to the specified file.
// If path is empty, logging is disabled by setting the logger to a no-op handler.
// Returns an error if the file cannot be opened.
func InitLogger(path string) (*os.File, error) {
	if path == "" {
		// Disable logging by setting a handler that discards all logs
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
		return nil, nil
	}

	// Open log file for appending
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	// Configure slog to write to the file with DEBUG level
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	handler := slog.NewTextHandler(file, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Write initial message
	slog.Info("Component routing logging started")

	return file, nil
}
