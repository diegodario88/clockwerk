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
)

type doneMsg struct{}
type tickMsg struct{}

type Model struct {
	step         int       // 0: CPF, 1: Senha, 2: Manter Logado, 3: Autenticando, 4: Buscando Eventos, 5: Timer (registro de ponto), 6: Finalizado
	cpfForm      *huh.Form // Formulário CPF
	passwordForm *huh.Form // Formulário Senha
	keepForm     *huh.Form // Formulário Manter Logado
	cpf          string    // CPF informado
	password     string    // Senha informada
	keepLogged   bool      // Deseja manter logado?
	spinner      spinner.Model

	// Campos para o timer (registro de ponto)
	timerRunning bool          // Indica se o timer está rodando
	elapsed      time.Duration // Tempo acumulado
	punchCount   int           // Contador de batidas (deve ser 4 no total)
}

var theme *huh.Theme = huh.ThemeBase()
var defaultConfirm = true

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
		// Observe que usamos uma variável ponteiro para o valor inicial:
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

func NewModel() Model {
	sp := spinner.New()
	sp.Spinner = spinner.Points
	sp.Style = lipgloss.NewStyle().
		PaddingLeft(1).
		Foreground(lipgloss.Color("#0097F4"))

	return Model{
		step:         0,
		cpfForm:      newCPFForm(""),
		passwordForm: newPasswordForm(""),
		keepForm:     newKeepForm(false),
		spinner:      sp,
		// Timer inicia parado, sem tempo acumulado e 0 batidas
		timerRunning: false,
		elapsed:      0,
		punchCount:   0,
	}
}

func (m Model) Init() tea.Cmd {
	tea.SetWindowTitle("Clockwerk")
	return m.cpfForm.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Tratamento geral de teclas
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
			m.keepForm = newKeepForm(m.keepForm.GetBool("keep"))
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
				// Volta para a etapa de senha
				m.step = 1
				m.passwordForm = newPasswordForm(m.passwordForm.GetString("password"))
				return m, m.passwordForm.Init()
			}
			m.keepLogged = m.keepForm.GetBool("keep")
			// Após esta etapa, simulamos dois spinners e depois iniciamos o timer.
			m.step = 3
			return m, tea.Batch(waitCmd(3), m.spinner.Tick)
		}
		return m, cmd

	case 3: // Etapa 4: Spinner (simulação de autenticação)
		m.spinner, cmd = m.spinner.Update(msg)
		switch msg.(type) {
		case doneMsg:
			m.step = 4
			return m, tea.Batch(waitCmd(3), m.spinner.Tick)
		}
		return m, cmd

	case 4: // Etapa 5: Spinner (simulação de busca de eventos)
		m.spinner, cmd = m.spinner.Update(msg)
		switch msg.(type) {
		case doneMsg:
			// Ao final das simulações, iniciamos o timer (registro de ponto)
			m.step = 5
			// O timer inicia parado (não dispara o tick)
			return m, nil
		}
		return m, cmd

	case 5: // Etapa 6: Timer (registro de ponto)
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case " ":
				// Pressionar space alterna o timer
				if m.timerRunning {
					m.timerRunning = false
					m.punchCount++ // Conta uma batida ao parar
					if m.punchCount >= 4 {
						m.step = 6 // Finaliza após 4 batidas
						return m, nil
					}
				} else {
					m.timerRunning = true
					return m, tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg{} })
				}
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

		if m.timerRunning {
			return m, tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg{} })
		}
		return m, nil

	case 6: // Etapa Final: Exibe o resultado final do registro de ponto
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "q" || msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
		}
		return m, nil

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
		// Exibe o relógio no formato HH:MM:SS
		h := int(m.elapsed.Hours())
		mm := int(m.elapsed.Minutes()) % 60
		ss := int(m.elapsed.Seconds()) % 60
		timeStr := fmt.Sprintf("%02d:%02d:%02d", h, mm, ss)
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Registro de Ponto - Timer") + "\n\n")
		b.WriteString("Tempo acumulado: " + timeStr + "\n")
		b.WriteString(fmt.Sprintf("Batidas: %d/4\n", m.punchCount))
		b.WriteString("\nPressione SPACE para iniciar/parar o timer. (q para sair)")
	case 6:
		h := int(m.elapsed.Hours())
		mm := int(m.elapsed.Minutes()) % 60
		ss := int(m.elapsed.Seconds()) % 60
		timeStr := fmt.Sprintf("%02d:%02d:%02d", h, mm, ss)
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Registro Finalizado!") + "\n\n")
		b.WriteString("Tempo total registrado: " + timeStr + "\n")
		b.WriteString("\nPressione 'q' para sair.")
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
