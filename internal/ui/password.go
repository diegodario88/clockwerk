package ui

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/diegodario88/clockwerk/internal/core"
)

func NewPasswordForm(initialValue string) *huh.Form {
	passwordInput := huh.NewInput().
		Key("password").
		Title("Senha").
		Description("Insira a sua senha. É a mesma usada nas aplicações SeniorX").
		Placeholder("******").
		Value(&initialValue).
		Validate(func(s string) error {
			if len(s) < 3 {
				return fmt.Errorf("Password deve conter 3 ou mais caracteres")
			}
			return nil
		}).
		EchoMode(huh.EchoModePassword)

	nextConfirm1 := huh.NewConfirm().
		Key("next").
		Value(&core.DefaultConfirm).
		Negative("Voltar").
		Affirmative("Prosseguir")

	return huh.NewForm(
		huh.NewGroup(passwordInput, nextConfirm1),
	).
		WithWidth(core.AppWidth).
		WithShowHelp(true).
		WithShowErrors(true).
		WithTheme(core.Theme)
}
