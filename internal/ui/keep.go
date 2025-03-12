package ui

import (
	"github.com/charmbracelet/huh"
	"github.com/diegodario88/clockwerk/internal/core"
)

func NewKeepForm(initialValue bool) *huh.Form {
	keepConfirm := huh.NewConfirm().
		Key("keep").
		Title("Deseja se manter logado?").
		Description("Se preferir não repetir esses passos, considere permanecer logado.").
		Value(&initialValue).
		Affirmative("Sim").
		Negative("Não")

	proceedConfirm := huh.NewConfirm().
		Key("next").
		Value(&core.DefaultConfirm).
		Affirmative("Prosseguir").
		Negative("Voltar")

	return huh.NewForm(
		huh.NewGroup(keepConfirm, proceedConfirm),
	).
		WithWidth(core.AppWidth).
		WithShowHelp(true).
		WithShowErrors(true).
		WithTheme(core.Theme)
}
