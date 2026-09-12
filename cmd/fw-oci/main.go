package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"n1h41/fw-oci/internal/config"
	"n1h41/fw-oci/internal/oci"
	"n1h41/fw-oci/internal/tui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	provider, err := cfg.Provider()
	if err != nil {
		fmt.Fprintf(os.Stderr, "provider: %v\n", err)
		os.Exit(1)
	}

	tenancy, err := provider.TenancyOCID()
	if err != nil {
		fmt.Fprintf(os.Stderr, "tenancy: %v\n", err)
		os.Exit(1)
	}

	client, err := oci.NewClient(provider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "client: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(tui.New(client, tenancy, cfg.CompartmentOCID), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui: %v\n", err)
		os.Exit(1)
	}
}
