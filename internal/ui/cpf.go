package ui

import (
	"fmt"
	"regexp"

	"github.com/charmbracelet/huh"
	"github.com/diegodario88/clockwerk/internal/core"
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
		Value(&core.DefaultConfirm).
		Key("next").
		Affirmative("Prosseguir").
		Negative("")

	return huh.NewForm(
		huh.NewGroup(domainInput, cpfInput, nextConfirm0),
	).
		WithWidth(core.AppWidth).
		WithShowHelp(true).
		WithShowErrors(true).
		WithTheme(core.Theme)
}
