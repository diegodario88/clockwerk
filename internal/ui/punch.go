package ui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/diegodario88/clockwerk/internal/core"
)

func NewPunchConfirmForm() *huh.Form {
	defaultValue := true
	confirm := huh.NewConfirm().
		Key("confirm").
		Title(
			lipgloss.NewStyle().
				Bold(true).
				Render("Deseja bater o ponto?"),
		).
		Description(
			lipgloss.NewStyle().
				Italic(true).
				Render("Ao confirmar, enviaremos uma http request para senior"),
		).
		Value(&defaultValue).
		Affirmative("Sim").
		Negative("Cancelar")

	return huh.NewForm(
		huh.NewGroup(confirm),
	).
		WithWidth(core.AppWidth).
		WithShowHelp(true).
		WithShowErrors(true).
		WithTheme(core.Theme)
}
