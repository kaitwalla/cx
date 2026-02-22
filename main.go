package main

import (
	"fmt"
	"os"

	"cx/tui"
	"cx/update"

	tea "github.com/charmbracelet/bubbletea"
)

// version is set at build time via -ldflags "-X main.version=..."
var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "update":
			if err := update.SelfUpdate(); err != nil {
				fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
				os.Exit(1)
			}
			return
		case "version":
			fmt.Printf("cx version %s\n", version)
			return
		case "help", "-h", "--help":
			printHelp()
			return
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
			printHelp()
			os.Exit(1)
		}
	}

	p := tea.NewProgram(tui.NewApp(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("cx - SSH host manager")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  cx          Launch interactive host selector")
	fmt.Println("  cx update   Update to latest release")
	fmt.Println("  cx version  Show version info")
	fmt.Println("  cx help     Show this help")
}
