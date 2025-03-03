package ui

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/diegodario88/clockwerk/config"
)

func NewCPFForm(initialValue string) *huh.Form {
	validateCPF := func(s string) error {
		if len(s) != 11 {
			return fmt.Errorf("CPF deve conter 11 dígitos")
		}
		for _, c := range s {
			if c < '0' || c > '9' {
				return fmt.Errorf("Apenas números permitidos")
			}
		}
		return nil
	}
	cpfInput := huh.NewInput().
		Key("cpf").
		Title("CPF").
		Placeholder("Digite").
		Value(&initialValue).
		CharLimit(11).
		Validate(validateCPF)
	nextConfirm0 := huh.NewConfirm().
		Value(&config.DefaultConfirm).
		Key("next").
		Affirmative("Prosseguir").
		Negative("")
	return huh.NewForm(
		huh.NewGroup(cpfInput, nextConfirm0),
	).WithWidth(config.DefaultWidth).WithShowHelp(true).WithShowErrors(true).WithTheme(config.Theme)
}

func NewPasswordForm(initialValue string) *huh.Form {
	passwordInput := huh.NewInput().
		Key("password").
		Title("Senha").
		Placeholder("Digite sua senha").
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
		Value(&config.DefaultConfirm).
		Negative("Voltar").
		Affirmative("Prosseguir")
	return huh.NewForm(
		huh.NewGroup(passwordInput, nextConfirm1),
	).WithWidth(config.DefaultWidth).WithShowHelp(true).WithShowErrors(true).WithTheme(config.Theme)
}

func NewKeepForm(initialValue bool) *huh.Form {
	keepConfirm := huh.NewConfirm().
		Key("keep").
		Title("Deseja se manter logado?").
		Value(&initialValue).
		Affirmative("Sim").
		Negative("Não")
	proceedConfirm := huh.NewConfirm().
		Key("next").
		Value(&config.DefaultConfirm).
		Affirmative("Prosseguir").
		Negative("Voltar")
	return huh.NewForm(
		huh.NewGroup(keepConfirm, proceedConfirm),
	).WithWidth(config.DefaultWidth).WithShowHelp(true).WithShowErrors(true).WithTheme(config.Theme)
}

func NewPunchConfirmForm() *huh.Form {
	defaultValue := true
	confirm := huh.NewConfirm().
		Key("confirm").
		Title("Bater o ponto?").
		Value(&defaultValue).
		Affirmative("Sim").
		Negative("Cancelar")
	return huh.NewForm(
		huh.NewGroup(confirm),
	).WithWidth(config.DefaultWidth).WithShowHelp(true).WithShowErrors(true).WithTheme(config.Theme)
}

func NewForgetForm() *huh.Form {
	defaultValue := false
	confirm := huh.NewConfirm().
		Key("confirm").
		Title("Deseja esquecer suas credenciais?").
		Value(&defaultValue).
		Affirmative("Sim").
		Negative("Não")
	return huh.NewForm(
		huh.NewGroup(confirm),
	).WithWidth(config.DefaultWidth).WithShowHelp(true).WithShowErrors(true).WithTheme(config.Theme)
}
