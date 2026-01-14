package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/kluzzebass/lazyhue/internal/app"
	"github.com/kluzzebass/lazyhue/internal/config"
	"github.com/kluzzebass/lazyhue/internal/ui/component"
)

func main() {
	// Parse command-line flags
	debugLog := flag.String("debug-log", "", "path to debug log file for component routing (empty to disable)")
	flag.Parse()

	// Initialize component routing logger
	logFile, err := component.InitLogger(*debugLog)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not initialize logging: %v\n", err)
		os.Exit(1)
	}
	if logFile != nil {
		defer logFile.Close()
	}

	// Initialize zone manager for mouse support
	zone.NewGlobal()

	// Load credentials
	creds, err := config.LoadCredentials()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load credentials: %v\n", err)
		creds = config.NewCredentialStore()
	}

	// Create the application model
	model := app.New(creds)

	// Create and run the program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseAllMotion(),
	)

	// Set up signal handling to save state on SIGINT, SIGTERM, SIGHUP
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		<-sigChan
		// Send quit message to trigger graceful shutdown with state save
		p.Send(app.SignalQuitMsg{})
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
