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

// Mensagem que indica o término da simulação HTTP (3s)
type doneMsg struct{}

// waitCmd aguarda X segundos e retorna um doneMsg.
func waitCmd(seconds int) tea.Cmd {
	return tea.Tick(time.Duration(seconds)*time.Second, func(t time.Time) tea.Msg {
		return doneMsg{}
	})
}

// Model representa o estado da aplicação.
type Model struct {
	step         int       // Etapa atual (0: CPF, 1: Senha, 2: Confirmação, 3: Spinner, 4: Resultado)
	cpfForm      *huh.Form // Formulário da Etapa 1
	passwordForm *huh.Form // Formulário da Etapa 2
	confirmForm  *huh.Form // Formulário da Etapa 3
	cpf          string    // CPF informado
	password     string    // Senha informada
	keepLogged   bool      // Deseja manter logado?
	spinner      spinner.Model
}

// NewModel inicializa os formulários e o spinner.
func NewModel() Model {
	var theme *huh.Theme = huh.ThemeBase()

	// Etapa 1: CPF
	cpfInput := huh.NewInput().
		Key("cpf").
		Title("CPF").
		Placeholder("Digite").
		CharLimit(11).
		Validate(func(s string) error {
			if len(s) != 11 {
				return fmt.Errorf("CPF deve conter 11 dígitos")
			}
			for _, c := range s {
				if c < '0' || c > '9' {
					return fmt.Errorf("Apenas números permitidos")
				}
			}
			return nil
		})
	nextConfirm0 := huh.NewConfirm().
		Key("next").
		Affirmative("Próximo").
		Negative("")

	cpfForm := huh.NewForm(
		huh.NewGroup(cpfInput, nextConfirm0),
	).WithWidth(45).WithShowHelp(true).WithShowErrors(true).WithTheme(theme)

	// Etapa 2: Senha
	passwordInput := huh.NewInput().
		Key("password").
		Title("Senha").
		Placeholder("Digite sua senha").
		EchoMode(huh.EchoModePassword)

	nextConfirm1 := huh.NewConfirm().
		Key("next").
		Affirmative("Próximo").
		Negative("")

	passwordForm := huh.NewForm(
		huh.NewGroup(passwordInput, nextConfirm1),
	).WithWidth(45).WithShowHelp(true).WithShowErrors(true).WithTheme(theme)

	// Etapa 3: Confirmação de manter logado
	keepConfirm := huh.NewConfirm().
		Key("keep").
		Title("Deseja se manter logado?").
		Affirmative("Sim").
		Negative("Não")

	proceedConfirm := huh.NewConfirm().
		Key("proceed").
		Affirmative("Prosseguir").
		Negative("")

	confirmForm := huh.NewForm(
		huh.NewGroup(keepConfirm, proceedConfirm),
	).WithWidth(45).WithShowHelp(true).WithShowErrors(true).WithTheme(theme)

	// Configura o spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#0097F4"))

	return Model{
		step:         0,
		cpfForm:      cpfForm,
		passwordForm: passwordForm,
		confirmForm:  confirmForm,
		spinner:      sp,
	}
}

func (m Model) Init() tea.Cmd {
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
			m.password = m.passwordForm.GetString("password")
			m.step = 2
			return m, m.confirmForm.Init()
		}
		return m, cmd

	case 2: // Etapa 3: Confirmação de manter logado
		newForm, c := m.confirmForm.Update(msg)
		if f, ok := newForm.(*huh.Form); ok {
			m.confirmForm = f
		}
		cmd = c
		if m.confirmForm.State == huh.StateCompleted {
			m.keepLogged = m.confirmForm.GetBool("keep")
			m.step = 3
			// Inicia a simulação HTTP (spinner por 3s)
			return m, tea.Batch(waitCmd(3), m.spinner.Tick)
		}
		return m, cmd

	case 3: // Etapa 4: Spinner
		m.spinner, cmd = m.spinner.Update(msg)
		// Ao receber doneMsg, passa para a etapa final
		switch msg.(type) {
		case doneMsg:
			m.step = 4
			return m, nil
		}
		return m, cmd

	case 4: // Etapa 5: Exibe os dados
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

// View renderiza a interface de acordo com a etapa atual.
func (m Model) View() string {
	var b strings.Builder

	switch m.step {
	case 0:
		b.WriteString(
			lipgloss.NewStyle().
				Bold(true).
				Render("Autenticação - Etapa 1/3: Identificação") +
				"\n\n",
		)
		b.WriteString(m.cpfForm.View())
	case 1:
		loginEmail := fmt.Sprintf("%s@gazin.com.br", m.cpf)
		b.WriteString(
			lipgloss.NewStyle().Bold(true).Render("Autenticação - Etapa 2/3: Senha") + "\n\n",
		)
		b.WriteString(
			lipgloss.NewStyle().
				Render("Login: "+loginEmail) +
				"\n\n",
		)
		b.WriteString(m.passwordForm.View())
	case 2:
		b.WriteString(
			lipgloss.NewStyle().Bold(true).Render("Login - Etapa 3: Manter Logado") + "\n\n",
		)
		b.WriteString(m.confirmForm.View())
	case 3:
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Autenticando...") + "\n\n")
		b.WriteString(m.spinner.View())
	case 4:
		loginEmail := fmt.Sprintf("%s@gazin.com.br", m.cpf)
		keep := "Não"
		if m.keepLogged {
			keep = "Sim"
		}
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Login Concluído!") + "\n\n")
		b.WriteString(
			fmt.Sprintf(
				"CPF: %s\nLogin: %s\nSenha: %s\nManter Logado: %s\n\nPressione 'q' para sair.",
				m.cpf,
				loginEmail,
				m.password,
				keep,
			),
		)
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
