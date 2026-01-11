package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/kluzzebass/lazyhue/internal/app2"
	"github.com/kluzzebass/lazyhue/internal/config"
)

func main() {
	// Initialize zone manager for mouse support
	zone.NewGlobal()

	// Load credentials
	creds, err := config.LoadCredentials()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load credentials: %v\n", err)
		creds = config.NewCredentialStore()
	}

	// Create the application model
	model := app2.New(creds)

	// Create and run the program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseAllMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
