package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/common-nighthawk/go-figure"
)

type doneMsg struct{}
type tickMsg struct{}

type Model struct {
	step          int
	cpfForm       *huh.Form
	passwordForm  *huh.Form
	keepForm      *huh.Form
	punchForm     *huh.Form
	cpf           string
	password      string
	keepLogged    bool
	spinner       spinner.Model
	timerRunning  bool
	elapsed       time.Duration
	punchCount    int
	tickScheduled bool
}

var theme *huh.Theme = huh.ThemeBase()
var defaultConfirm = true
var clockWerkColor = "#E28413"

func waitCmd(seconds int) tea.Cmd {
	return tea.Tick(time.Duration(seconds)*time.Second, func(t time.Time) tea.Msg {
		return doneMsg{}
	})
}

func newCPFForm(initialValue string) *huh.Form {
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
		Value(&defaultConfirm).
		Key("next").
		Affirmative("Prosseguir").
		Negative("Voltar")

	return huh.NewForm(
		huh.NewGroup(cpfInput, nextConfirm0),
	).WithWidth(45).WithShowHelp(true).WithShowErrors(true).WithTheme(theme)
}

func newPasswordForm(initialValue string) *huh.Form {
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
		Value(&defaultConfirm).
		Negative("Voltar").
		Affirmative("Prosseguir")

	return huh.NewForm(
		huh.NewGroup(passwordInput, nextConfirm1),
	).WithWidth(45).WithShowHelp(true).WithShowErrors(true).WithTheme(theme)
}

func newKeepForm(initialValue bool) *huh.Form {
	keepConfirm := huh.NewConfirm().
		Key("keep").
		Title("Deseja se manter logado?").
		Value(&initialValue).
		Affirmative("Sim").
		Negative("Não")

	proceedConfirm := huh.NewConfirm().
		Key("next").
		Value(&defaultConfirm).
		Affirmative("Prosseguir").
		Negative("Voltar")

	return huh.NewForm(
		huh.NewGroup(keepConfirm, proceedConfirm),
	).WithWidth(45).WithShowHelp(true).WithShowErrors(true).WithTheme(theme)
}

func newPunchConfirmForm() *huh.Form {
	defaultValue := true
	confirm := huh.NewConfirm().
		Key("confirm").
		Title("Bater o ponto?").
		Value(&defaultValue).
		Affirmative("Sim").
		Negative("Cancelar")

	return huh.NewForm(
		huh.NewGroup(confirm),
	).WithWidth(45).WithShowHelp(true).WithShowErrors(true).WithTheme(theme)
}

func NewModel() Model {
	sp := spinner.New()
	sp.Spinner = spinner.Points
	sp.Style = lipgloss.NewStyle().
		PaddingLeft(1).
		Foreground(lipgloss.Color(clockWerkColor))

	return Model{
		step:         0,
		cpfForm:      newCPFForm(""),
		passwordForm: newPasswordForm(""),
		keepForm:     newKeepForm(true),
		punchForm:    nil,
		spinner:      sp,
		timerRunning: false,
		elapsed:      0,
		punchCount:   0,
		keepLogged:   true,
	}
}

func (m Model) Init() tea.Cmd {
	tea.SetWindowTitle("Clockwerk")
	return m.cpfForm.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Interrupt
		}
	}

	var cmd tea.Cmd

	switch m.step {
	case 0: // Etapa 1: CPF
		newForm, c := m.cpfForm.Update(msg)
		if f, ok := newForm.(*huh.Form); ok {
			m.cpfForm = f
		}
		cmd = c

		if m.cpfForm.State == huh.StateCompleted {
			m.cpf = m.cpfForm.GetString("cpf")
			m.step = 1
			// Passa valor anterior (se houver) para o formulário de senha
			m.passwordForm = newPasswordForm(m.passwordForm.GetString("password"))
			return m, m.passwordForm.Init()
		}
		return m, cmd

	case 1: // Etapa 2: Senha
		newForm, c := m.passwordForm.Update(msg)
		if f, ok := newForm.(*huh.Form); ok {
			m.passwordForm = f
		}
		cmd = c

		if m.passwordForm.State == huh.StateCompleted {
			if !m.passwordForm.GetBool("next") {
				// Volta para CPF (mantendo o valor digitado)
				m.step = 0
				m.cpfForm = newCPFForm(m.cpfForm.GetString("cpf"))
				return m, m.cpfForm.Init()
			}
			m.password = m.passwordForm.GetString("password")
			m.step = 2
			m.keepForm = newKeepForm(m.keepLogged)
			return m, m.keepForm.Init()
		}
		return m, cmd

	case 2: // Etapa 3: Manter Logado
		newForm, c := m.keepForm.Update(msg)
		if f, ok := newForm.(*huh.Form); ok {
			m.keepForm = f
		}
		cmd = c

		if m.keepForm.State == huh.StateCompleted {
			if !m.keepForm.GetBool("next") {
				m.step = 1
				m.passwordForm = newPasswordForm(m.passwordForm.GetString("password"))
				m.keepLogged = m.keepForm.GetBool("keep")
				return m, m.passwordForm.Init()
			}
			m.keepLogged = m.keepForm.GetBool("keep")
			m.step = 3
			return m, tea.Batch(waitCmd(1), m.spinner.Tick)
		}
		return m, cmd

	case 3: // Etapa 4: Spinner (simulação de autenticação)
		m.spinner, cmd = m.spinner.Update(msg)
		switch msg.(type) {
		case doneMsg:
			m.step = 4
			return m, tea.Batch(waitCmd(1), m.spinner.Tick)
		}
		return m, cmd

	case 4: // Etapa 5: Spinner (simulação de busca de eventos)
		m.spinner, cmd = m.spinner.Update(msg)
		switch msg.(type) {
		case doneMsg:
			m.step = 5
			return m, nil
		}
		return m, cmd

	case 5:
		if m.punchForm != nil {
			updatedForm, c := m.punchForm.Update(msg)
			if f, ok := updatedForm.(*huh.Form); ok {
				m.punchForm = f
			}
			cmd = c

			if m.punchForm.State == huh.StateCompleted {
				if m.punchForm.GetBool("confirm") {
					// Se confirmado, Inicie aqui um spinner para simular um HTTP POST.
					// Se tudo ocorrer bem, alterna o estado do timer e registra a batida.
					m.step = 6
					m.punchForm = nil

					return m, tea.Batch(waitCmd(1), m.spinner.Tick)
				} else {
					m.punchForm = nil
					if m.timerRunning {
						return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
							return tickMsg{}
						})
					}
				}
			}

			return m, cmd
		}

		// Se o formulário de confirmação não está ativo, então:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case " ":
				// Ao pressionar SPACE e sem formulário ativo, dispara o formulário de confirmação.
				m.punchForm = newPunchConfirmForm()
				return m, m.punchForm.Init()

			case "q", "ctrl+c":
				return m, tea.Quit
			}

		case tickMsg:
			if m.timerRunning {
				m.elapsed += time.Second
				return m, tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg{} })
			}
			return m, nil
		}

		return m, nil

	case 6:
		m.spinner, cmd = m.spinner.Update(msg)
		switch msg.(type) {
		case doneMsg:
			m.step = 5
			m.timerRunning = !m.timerRunning
			m.punchCount++

			if m.timerRunning {
				return m, tea.Batch(cmd, tea.Tick(time.Second, func(t time.Time) tea.Msg {
					m.tickScheduled = false
					return tickMsg{}
				}))
			}

			return m, nil
		}

		return m, cmd

	default:
		return m, nil
	}
}

func (m Model) View() string {
	var b strings.Builder

	switch m.step {
	case 0:
		b.WriteString(lipgloss.NewStyle().
			Bold(true).
			Render("Autenticação - Etapa 1/3: Identificação") + "\n\n")
		b.WriteString(m.cpfForm.View())
	case 1:
		loginEmail := fmt.Sprintf("%s@gazin.com.br", m.cpf)
		b.WriteString(
			lipgloss.NewStyle().Bold(true).Render("Autenticação - Etapa 2/3: Senha") + "\n\n",
		)
		b.WriteString(lipgloss.NewStyle().Italic(true).Render("Login: "+loginEmail) + "\n\n")
		b.WriteString(m.passwordForm.View())
	case 2:
		b.WriteString(lipgloss.NewStyle().
			Bold(true).
			Render("Autenticação - Etapa 3/3: Manter Logado") + "\n\n")
		b.WriteString(m.keepForm.View())
	case 3:
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Autenticando...") + "\n\n")
		b.WriteString(m.spinner.View())
	case 4:
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Buscando últimos eventos...") + "\n\n")
		b.WriteString(m.spinner.View())
	case 5:
		now := time.Now()
		h := int(m.elapsed.Hours())
		mm := int(m.elapsed.Minutes()) % 60
		ss := int(m.elapsed.Seconds()) % 60
		timeStr := fmt.Sprintf("%02d:%02d:%02d", h, mm, ss)
		b.WriteString(
			lipgloss.NewStyle().Bold(true).Width(30).Render("Registro de Ponto - Timer") + "\n\n",
		)
		b.WriteString(
			lipgloss.NewStyle().Render("Data atual: " + now.Local().Format("02/01/2006") + "\n"),
		)
		b.WriteString(fmt.Sprintf("Registros de ponto: %d\n", m.punchCount))
		b.WriteString(lipgloss.NewStyle().Render("Tempo de trabalho:\n\n"))
		b.WriteString(
			lipgloss.NewStyle().
				Width(200).
				Height(10).
				Bold(true).
				Foreground(lipgloss.Color(clockWerkColor)).
				PaddingLeft(2).
				Render(figure.NewFigure(timeStr, "starwars", true).String() + "\n"),
		)
		if m.punchForm != nil {
			b.WriteString("\nConfirma o registro de ponto?\n")
			b.WriteString(m.punchForm.View())
		} else {
			b.WriteString("\nPressione SPACE para iniciar/parar o timer. (q para sair)")
		}
	case 6:
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Registrando evento ponto...") + "\n\n")
		b.WriteString(m.spinner.View())
	}

	return b.String()
}

func main() {
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()

	if _, err := tea.NewProgram(NewModel(), tea.WithAltScreen()).Run(); err != nil {
		log.Printf("Erro ao executar o programa: %s\n", err)
		os.Exit(1)
	}
}
