// Package main is the entry point for the lazyhue application.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kluzzebass/lazyhue/internal/app"
	"github.com/kluzzebass/lazyhue/internal/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Load credentials
	creds, err := config.LoadCredentials()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading credentials: %v\n", err)
		os.Exit(1)
	}

	// Load UI state
	uiState, err := config.LoadUIState()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading UI state: %v\n", err)
		os.Exit(1)
	}

	// Create the application model
	model := app.New(cfg, creds, uiState)

	// Create the program with options
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
