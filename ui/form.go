package ui

import (
	"fmt"
	"regexp"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/diegodario88/clockwerk/config"
)

func NewCPFForm(initialDomainValue string, initialCpfValue string) *huh.Form {
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

	validateDomain := func(s string) error {
		if s == "" {
			return fmt.Errorf("o domínio não pode estar vazio")
		}
		re := regexp.MustCompile(`^([a-zA-Z0-9]+(-[a-zA-Z0-9]+)*\.)+[a-zA-Z]{2,}$`)
		if !re.MatchString(s) {
			return fmt.Errorf("o domínio não é válido")
		}
		return nil
	}

	domainInput := huh.NewInput().
		Key("domain").
		Title("Domínio").
		Description("Insira o domínio da sua empresa").
		Placeholder("exemplo.com.br").
		Value(&initialDomainValue).
		Validate(validateDomain)

	cpfInput := huh.NewInput().
		Key("cpf").
		Title("CPF").
		Description("Insira o número do seu CPF").
		Placeholder("22920181017").
		Value(&initialCpfValue).
		CharLimit(11).
		Validate(validateCPF)

	nextConfirm0 := huh.NewConfirm().
		Value(&config.DefaultConfirm).
		Key("next").
		Affirmative("Prosseguir").
		Negative("")

	return huh.NewForm(
		huh.NewGroup(domainInput, cpfInput, nextConfirm0),
	).
		WithWidth(config.DefaultWidth).
		WithShowHelp(true).
		WithShowErrors(true).
		WithTheme(config.Theme)
}

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
		Value(&config.DefaultConfirm).
		Negative("Voltar").
		Affirmative("Prosseguir")

	return huh.NewForm(
		huh.NewGroup(passwordInput, nextConfirm1),
	).
		WithWidth(config.DefaultWidth).
		WithShowHelp(true).
		WithShowErrors(true).
		WithTheme(config.Theme)
}

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
		Value(&config.DefaultConfirm).
		Affirmative("Prosseguir").
		Negative("Voltar")

	return huh.NewForm(
		huh.NewGroup(keepConfirm, proceedConfirm),
	).
		WithWidth(config.DefaultWidth).
		WithShowHelp(true).
		WithShowErrors(true).
		WithTheme(config.Theme)
}

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
		WithWidth(config.DefaultWidth).
		WithShowHelp(true).
		WithShowErrors(true).
		WithTheme(config.Theme)
}

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
		WithWidth(config.DefaultWidth).
		WithShowHelp(true).
		WithShowErrors(true).
		WithTheme(config.Theme)
}
