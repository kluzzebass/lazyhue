// Package debug provides a global debug logger that writes to a file.
package debug

import (
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	mu      sync.Mutex
	file    *os.File
	enabled bool
)

// Init initializes the debug logger with the given file path.
// If path is empty, debug logging is disabled.
func Init(path string) error {
	mu.Lock()
	defer mu.Unlock()

	if path == "" {
		enabled = false
		return nil
	}

	var err error
	file, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	enabled = true

	// Write initial message directly (we already hold the lock)
	timestamp := time.Now().Format("15:04:05.000")
	fmt.Fprintf(file, "[%s] Debug logging started at %s\n", timestamp, time.Now().Format(time.RFC3339))
	file.Sync()

	return nil
}

// Close closes the debug log file.
func Close() {
	mu.Lock()
	defer mu.Unlock()

	if file != nil {
		file.Close()
		file = nil
	}
	enabled = false
}

// Log writes a debug message if logging is enabled.
func Log(format string, args ...any) {
	mu.Lock()
	defer mu.Unlock()

	if !enabled || file == nil {
		return
	}

	timestamp := time.Now().Format("15:04:05.000")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(file, "[%s] %s\n", timestamp, msg)
	file.Sync() // Ensure it's written immediately
}

// Enabled returns whether debug logging is enabled.
func Enabled() bool {
	mu.Lock()
	defer mu.Unlock()
	return enabled
}
