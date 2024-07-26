package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	title string
}

func NewModel() Model {
	return Model{
		title: "hello world",
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m Model) View() string {
	return m.title
}

func main() {
	m := NewModel()
	program := tea.NewProgram(m)

	_, err := program.Run()
	if err != nil {
		log.Fatalln(err)
	}
}
