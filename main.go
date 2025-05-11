package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/diegodario88/clockwerk/internal"
)

func main() {
	var hasDebug = false
	if len(os.Getenv("DEBUG")) > 0 {
		hasDebug = true
	}

	if hasDebug {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
	}

	clockTimer := internal.NewClockTimer()
	program := tea.NewProgram(clockTimer, tea.WithAltScreen())

	if _, err := program.Run(); err != nil {
		if hasDebug {
			log.Printf("Erro ao executar o programa: %s\n", err)
		}
		os.Exit(1)
	}

	program.Quit()
}
