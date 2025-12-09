package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ilia/ps9s/internal/aws"
	"github.com/ilia/ps9s/internal/config"
	"github.com/ilia/ps9s/internal/ui"
)

func main() {
	// Parse profiles from environment
	profiles, err := config.GetProfilesFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nPlease set PS9S_AWS_PROFILES environment variable with comma-separated profile names.\n")
		fmt.Fprintf(os.Stderr, "Example: export PS9S_AWS_PROFILES=dev,staging,prod\n")
		os.Exit(1)
	}

	// Load region mapping from config
	regionMapping, err := config.LoadRegionMapping()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load region mapping: %v\n", err)
		// Continue with empty mapping
		regionMapping = &config.RegionMapping{
			ProfileRegions: make(map[string]string),
		}
	}

	// Initialize root model with empty client pool
	// Clients will be created after region selection
	clientPool := make(map[string]*aws.Client)
	model := ui.NewModel(profiles, clientPool, regionMapping)

	// Start Bubble Tea program with alt screen
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
