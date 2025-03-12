package ui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/diegodario88/clockwerk/internal/core"
)

func NewForgetForm() *huh.Form {
	defaultValue := false
	confirm := huh.NewConfirm().
		Key("confirm").
		Title(
			lipgloss.NewStyle().
				Bold(true).
				Render("Deseja esquecer suas credenciais?"),
		).
		Description(
			lipgloss.NewStyle().
				Italic(true).
				Render("Ao confirmar, todas as informações de login serão deletadas"),
		).
		Value(&defaultValue).
		Affirmative("Sim").
		Negative("Não")

	return huh.NewForm(
		huh.NewGroup(confirm),
	).
		WithWidth(core.AppWidth).
		WithShowHelp(true).
		WithShowErrors(true).
		WithTheme(core.Theme)
}
