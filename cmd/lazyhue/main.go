// Package main is the entry point for the lazyhue application.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kluzzebass/lazyhue/internal/app"
	"github.com/kluzzebass/lazyhue/internal/config"
	lazydebug "github.com/kluzzebass/lazyhue/internal/debug"
	"github.com/spf13/cobra"
)

var (
	debugLog string
)

var rootCmd = &cobra.Command{
	Use:   "lazyhue",
	Short: "A TUI for controlling Philips Hue lights",
	Long:  `LazyHue is a terminal user interface for discovering, configuring, and controlling Philips Hue bridges and lights.`,
	Run:   run,
}

func init() {
	rootCmd.Flags().StringVar(&debugLog, "debug-log", "", "Path to debug log file (enables debug output)")
}

func run(cmd *cobra.Command, args []string) {
	// Capture panics and print stack trace
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "\n=== PANIC ===\n%v\n\n%s\n", r, debug.Stack())
			os.Exit(1)
		}
	}()

	// Initialize debug logging if requested
	if debugLog != "" {
		if err := lazydebug.Init(debugLog); err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing debug log: %v\n", err)
			os.Exit(1)
		}
		defer lazydebug.Close()
	}

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
		tea.WithoutCatchPanics(),  // Let panics bubble up with stack traces
	)

	// Handle SIGTERM/SIGINT to save state before exiting
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigChan
		// Signal received - send quit message to save state and exit
		p.Send(app.SignalQuitMsg{})
	}()

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	// Top-level panic capture
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "\n=== PANIC (main) ===\n%v\n\n%s\n", r, debug.Stack())
			os.Exit(1)
		}
	}()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
