package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
	"github.com/common-nighthawk/go-figure"
	"github.com/diegodario88/clockwerk/senior"
	"github.com/diegodario88/clockwerk/storage"
)

type keyMap struct {
	Punch       key.Binding
	ForgetCreds key.Binding
	Quit        key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Punch, k.ForgetCreds, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Punch, k.ForgetCreds, k.Quit},
	}
}

var keys = keyMap{
	Punch: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("<space>", "Bater o ponto"),
	),
	ForgetCreds: key.NewBinding(
		key.WithKeys("e", "E"),
		key.WithHelp("<e>", "Esquecer credenciais"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("<ctrl+c>", "Sair"),
	),
}

type tickMsg struct{}
type failedMsg struct{ error string }
type loginMsg struct{ token string }

type postClockingMsg struct {
	dateEvent string
	timeEvent string
}

type eventMsg struct {
	employeeName     string
	employeeId       string
	employeeArpId    string
	companyName      string
	companyId        string
	companyArpId     string
	cnpj             string
	cpf              string
	pis              string
	caepf            string
	cnoNumber        string
	appVersion       string
	timeZone         string
	shift            string
	timeTable        string
	signatureVersion int
	signature        string
	use              int
	clocking         map[string][]clockingMsg
}

type clockingMsg struct {
	id        string
	date      string
	time      string
	platform  string
	eventTime time.Time
}

type Model struct {
	step           int
	punchCount     int
	keepLogged     bool
	timerRunning   bool
	tickScheduled  bool
	hasAuthRecover bool
	cpf            string
	password       string
	token          string
	cpfForm        *huh.Form
	passwordForm   *huh.Form
	keepForm       *huh.Form
	punchForm      *huh.Form
	forgetForm     *huh.Form
	failedMsg      failedMsg
	loginMsg       loginMsg
	eventMsg       eventMsg
	spinner        spinner.Model
	elapsed        time.Duration
	help           help.Model
	keys           keyMap
}

var theme *huh.Theme = huh.ThemeBase()
var defaultConfirm = true
var todayKey = time.Now().Local().Format("2006-01-02")

const clockWerkColor = "#E28413"
const timeLayout = "2006-01-02 15:04:05.999 -07:00"

const defaultWidth = 80

func handleAuthentication(user string, password string) tea.Cmd {
	return func() tea.Msg {
		token, err := senior.GatewayLogin(user, password)
		if err != nil {
			return failedMsg{error: err.Error()}
		}

		return loginMsg{token: token}
	}
}

func handleGetClockingEvent(token string) tea.Cmd {
	return func() tea.Msg {
		events, err := senior.GetClockingEvents(token)
		if err != nil {
			return failedMsg{error: err.Error()}
		}
		if len(events) == 0 {
			return failedMsg{error: "lista de eventos vazia"}
		}

		grouped := make(map[string][]clockingMsg)
		for _, event := range events {
			timeStr := fmt.Sprintf("%s %s %s", event.DateEvent, event.TimeEvent, event.TimeZone)

			parsedTime, err := time.Parse(
				timeLayout,
				timeStr,
			)

			if err != nil {
				return failedMsg{
					error: fmt.Sprintf("erro parseando horário %s %s: %v",
						event.DateEvent,
						event.TimeEvent,
						err),
				}
			}

			cMsg := clockingMsg{
				id:        event.ID,
				date:      event.DateEvent,
				time:      event.TimeEvent,
				platform:  event.Platform,
				eventTime: parsedTime,
			}
			grouped[cMsg.date] = append(grouped[cMsg.date], cMsg)
		}

		for date, clockings := range grouped {
			sort.Slice(clockings, func(i, j int) bool {
				return clockings[i].eventTime.Before(clockings[j].eventTime)
			})
			grouped[date] = clockings
		}

		return eventMsg{
			employeeName:     events[0].Employee.Name,
			employeeId:       events[0].Employee.ID,
			employeeArpId:    events[0].Employee.ArpID,
			companyName:      events[0].Employee.Company.Name,
			companyId:        events[0].Employee.Company.ID,
			companyArpId:     events[0].Employee.ArpID,
			cnpj:             events[0].Employee.Company.Cnpj,
			pis:              events[0].Employee.Pis,
			caepf:            events[0].Caepf,
			appVersion:       events[0].AppVersion,
			cnoNumber:        events[0].CnoNumber,
			timeZone:         events[0].TimeZone,
			shift:            events[0].Employee.Shift,
			timeTable:        events[0].Employee.Timetable,
			signatureVersion: events[0].SignatureVersion,
			signature:        events[0].Signature,
			use:              events[0].Use,
			clocking:         grouped,
		}
	}
}

func handlePostClockingEvent(token string, event eventMsg) tea.Cmd {
	return func() tea.Msg {
		cResp, err := senior.PostClockingEvent(token, senior.ClockingRequest{
			ClockingInfo: senior.ClockingInfo{
				Company: senior.ClockingCompany{
					ID:         event.companyId,
					ArpID:      event.companyArpId,
					Identifier: event.cnpj,
					Caepf:      event.caepf,
					CnoNumber:  event.cnoNumber,
				},
				Employee: senior.ClockingEmployee{
					ID:    event.employeeId,
					ArpID: event.employeeArpId,
					Cpf:   event.cpf,
					Pis:   event.pis,
				},
				Signature: senior.ClockingSignature{
					SignatureVersion: event.signatureVersion,
					Signature:        event.signature,
				},
				AppVersion: event.appVersion,
				TimeZone:   event.timeZone,
				Use:        fmt.Sprintf("%02d", event.use),
			},
		})

		if err != nil {
			log.Println(err.Error())
			return failedMsg{error: err.Error()}
		}

		return postClockingMsg{
			dateEvent: cResp.Result.EventImported.DateEvent,
			timeEvent: cResp.Result.EventImported.DateEvent,
		}
	}
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
		Negative("")
	return huh.NewForm(
		huh.NewGroup(cpfInput, nextConfirm0),
	).WithWidth(defaultWidth).WithShowHelp(true).WithShowErrors(true).WithTheme(theme)
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
	).WithWidth(defaultWidth).WithShowHelp(true).WithShowErrors(true).WithTheme(theme)
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
	).WithWidth(defaultWidth).WithShowHelp(true).WithShowErrors(true).WithTheme(theme)
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
	).WithWidth(defaultWidth).WithShowHelp(true).WithShowErrors(true).WithTheme(theme)
}

func newForgetForm() *huh.Form {
	defaultValue := false
	confirm := huh.NewConfirm().
		Key("confirm").
		Title("Deseja esquecer suas credenciais?").
		Value(&defaultValue).
		Affirmative("Sim").
		Negative("Não")
	return huh.NewForm(
		huh.NewGroup(confirm),
	).WithWidth(defaultWidth).WithShowHelp(true).WithShowErrors(true).WithTheme(theme)
}

func NewModel() Model {
	sp := spinner.New()
	sp.Spinner = spinner.Points
	sp.Style = lipgloss.NewStyle().
		PaddingLeft(1).
		Foreground(lipgloss.Color(clockWerkColor))

	helpModel := help.New()

	creds, err := storage.LoadCredentials()
	initialStep := 0
	initialCPF := ""
	initialPassword := ""
	initialToken := ""

	if err == nil && creds.CPF != "" && creds.Password != "" && creds.Token != "" {
		initialStep = 4
		initialCPF = creds.CPF
		initialPassword = creds.Password
		initialToken = creds.Token
	}

	return Model{
		step:         initialStep,
		cpf:          initialCPF,
		password:     initialPassword,
		token:        initialToken,
		cpfForm:      newCPFForm(initialCPF),
		passwordForm: newPasswordForm(initialPassword),
		keepForm:     newKeepForm(true),
		punchForm:    nil,
		forgetForm:   nil,
		spinner:      sp,
		timerRunning: false,
		elapsed:      0,
		punchCount:   0,
		keepLogged:   true,
		help:         helpModel,
		keys:         keys,
	}
}

func (m Model) Init() tea.Cmd {
	tea.SetWindowTitle("Clockwerk")

	theme.Focused.Base = lipgloss.NewStyle().
		PaddingLeft(1).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color(clockWerkColor)).
		BorderLeft(true)

	if m.step == 0 {
		return m.cpfForm.Init()
	} else if m.step == 4 {
		return tea.Batch(handleGetClockingEvent(m.token), m.spinner.Tick)
	}

	return nil
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
			return m, tea.Batch(
				handleAuthentication(fmt.Sprintf("%s@gazin.com.br", m.cpf), m.password),
				m.spinner.Tick,
			)
		}
		return m, cmd

	case 3: // Etapa 4: Spinner (autenticação)
		m.spinner, cmd = m.spinner.Update(msg)
		switch msg := msg.(type) {
		case failedMsg:
			m.failedMsg = msg
			return m, nil
		case loginMsg:
			m.loginMsg = msg
			m.step = 4
			m.failedMsg = failedMsg{error: ""}

			if m.keepLogged {
				creds := storage.UserCredentials{
					CPF:      m.cpf,
					Password: m.password,
					Token:    msg.token,
				}
				if err := storage.SaveCredentials(creds); err != nil {
					log.Printf("Erro ao salvar credenciais: %v", err)
				}
			}
			return m, tea.Batch(handleGetClockingEvent(m.token), m.spinner.Tick)
		case tea.KeyMsg:
			switch msg.String() {
			case "r":
				fallthrough
			case "R":
				if m.failedMsg.error != "" {
					m.step = 0
					m.cpfForm = newCPFForm(m.cpfForm.GetString("cpf"))
					return m, m.cpfForm.Init()
				}
			}
		}
		return m, cmd

	case 4: // Etapa 5: Spinner (busca de eventos)
		m.spinner, cmd = m.spinner.Update(msg)
		switch msg := msg.(type) {
		case failedMsg:
			if strings.Contains(msg.error, "Unauthorized") && !m.hasAuthRecover {
				storage.DeleteCredentials()
				m.step = 3
				m.hasAuthRecover = true
				return m, tea.Batch(
					handleAuthentication(fmt.Sprintf("%s@gazin.com.br", m.cpf), m.password),
					m.spinner.Tick,
				)
			}
			m.failedMsg = msg
			return m, nil
		case eventMsg:
			m.step = 5
			m.eventMsg = msg
			m.elapsed = 0
			m.timerRunning = false
			maybeTodayClock, exists := m.eventMsg.clocking[todayKey]
			if exists {
				m.punchCount = len(maybeTodayClock)
				if len(maybeTodayClock)%2 != 0 {
					m.timerRunning = true
					loc := time.FixedZone("UTC-3", -3*3600)
					nowInUTC3 := time.Now().In(loc)
					formatted := nowInUTC3.Format(timeLayout)
					parsedTime, err := time.ParseInLocation(timeLayout, formatted, loc)
					if err != nil {
						log.Println("Erro ao parsear o tempo:", err)
					}
					maybeTodayClock = append(maybeTodayClock, clockingMsg{eventTime: parsedTime})
				}

				for i := 0; i < len(maybeTodayClock)-1; i += 2 {
					startTime := maybeTodayClock[i].eventTime
					endTime := maybeTodayClock[i+1].eventTime
					m.elapsed += endTime.Sub(startTime)
				}
			}

			if m.timerRunning {
				return m, tea.Batch(cmd, tea.Tick(time.Second, func(t time.Time) tea.Msg {
					m.tickScheduled = false
					return tickMsg{}
				}))
			}
			return m, nil
		case tea.KeyMsg:
			switch msg.String() {
			case "r":
				fallthrough
			case "R":
				if m.failedMsg.error != "" {
					m.step = 0
					m.cpfForm = newCPFForm(m.cpfForm.GetString("cpf"))
					return m, m.cpfForm.Init()
				}
			}
		}
		return m, cmd

	case 5:
		// Tratamento para o formulário de esquecer credenciais
		if m.forgetForm != nil {
			updatedForm, c := m.forgetForm.Update(msg)
			if f, ok := updatedForm.(*huh.Form); ok {
				m.forgetForm = f
			}
			cmd = c
			if m.forgetForm.State == huh.StateCompleted {
				if m.forgetForm.GetBool("confirm") {
					if err := storage.DeleteCredentials(); err != nil {
						log.Printf("Erro ao deletar credenciais: %v", err)
					}
				}
				m.forgetForm = nil
				return m, nil
			}
			return m, cmd
		}

		// Tratamento para o formulário de confirmação de ponto
		if m.punchForm != nil {
			updatedForm, c := m.punchForm.Update(msg)
			if f, ok := updatedForm.(*huh.Form); ok {
				m.punchForm = f
			}
			cmd = c
			if m.punchForm.State == huh.StateCompleted {
				if m.punchForm.GetBool("confirm") {
					m.step = 6
					m.punchForm = nil
					return m, tea.Batch(
						handlePostClockingEvent(m.token, m.eventMsg),
						m.spinner.Tick,
					)
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
			switch {
			case key.Matches(msg, m.keys.Punch):
				// Ao pressionar SPACE e sem formulário ativo, dispara o formulário de confirmação.
				m.punchForm = newPunchConfirmForm()
				return m, m.punchForm.Init()
			case key.Matches(msg, m.keys.ForgetCreds):
				// Ao pressionar E, dispara o formulário para esquecer credenciais
				m.forgetForm = newForgetForm()
				return m, m.forgetForm.Init()
			case key.Matches(msg, m.keys.Quit):
				return m, tea.Quit
			}
		case tickMsg:
			if m.timerRunning {
				m.elapsed += time.Second
				return m, tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg{} })
			}
			return m, nil
		case failedMsg:
			m.step = 4
			m.failedMsg = msg
			return m, nil
		}
		return m, nil

	case 6:
		m.spinner, cmd = m.spinner.Update(msg)
		switch msg.(type) {
		case postClockingMsg:
			m.step = 4
			return m, tea.Batch(handleGetClockingEvent(m.token), m.spinner.Tick)
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
		if m.failedMsg.error != "" {
			b.WriteString(
				lipgloss.NewStyle().
					Bold(true).
					Render("Ops.. Falha no processo de autenticação") +
					"\n\n",
			)
			b.WriteString(
				lipgloss.NewStyle().
					Italic(true).
					Render(fmt.Sprintf("Login: %s@gazin.com.br", m.cpf)) + "\n",
			)
			b.WriteString(
				lipgloss.NewStyle().
					Italic(true).
					Render(fmt.Sprintf("Mensagem: %s", m.failedMsg.error)) +
					"\n\n",
			)
			b.WriteString(
				lipgloss.NewStyle().
					PaddingLeft(2).
					Blink(true).
					Foreground(lipgloss.Color(clockWerkColor)).
					Render(
						"¯\\_(ツ)_/¯",
					) + "\n\n",
			)
			b.WriteString("Pressione R para tentar novamente.")
		} else {
			b.WriteString(lipgloss.NewStyle().Bold(true).Render("Autenticando...") + "\n\n")
			b.WriteString(m.spinner.View())
		}

	case 4:
		if m.failedMsg.error != "" {
			b.WriteString(
				lipgloss.NewStyle().
					Bold(true).
					Render("Ops.. Falha no processo de busca de eventos") +
					"\n\n",
			)
			b.WriteString(
				lipgloss.NewStyle().
					Italic(true).
					Render(fmt.Sprintf("Login: %s@gazin.com.br", m.cpf)) + "\n",
			)
			b.WriteString(
				lipgloss.NewStyle().
					Italic(true).
					Render(fmt.Sprintf("Mensagem: %s", m.failedMsg.error)) +
					"\n\n",
			)
			b.WriteString(
				lipgloss.NewStyle().
					PaddingLeft(2).
					Blink(true).
					Foreground(lipgloss.Color(clockWerkColor)).
					Render(
						"¯\\_(ツ)_/¯",
					) + "\n\n",
			)
			b.WriteString("Pressione R para tentar novamente.")
		} else {
			b.WriteString(lipgloss.NewStyle().Bold(true).Render("Buscando últimos eventos...") + "\n\n")
			b.WriteString(m.spinner.View())
		}

	case 5:
		now := time.Now()
		h := int(m.elapsed.Hours())
		mm := int(m.elapsed.Minutes()) % 60
		ss := int(m.elapsed.Seconds()) % 60
		timeStr := fmt.Sprintf("%02d:%02d:%02d", h, mm, ss)
		b.WriteString(
			lipgloss.NewStyle().
				Bold(true).
				Width(defaultWidth).
				Render("Registro de Ponto - Clockwerk") +
				"\n\n",
		)
		b.WriteString(
			lipgloss.NewStyle().Render("Colaborador: " + m.eventMsg.employeeName + "\n"),
		)
		b.WriteString(
			lipgloss.NewStyle().Render("Empresa: " + m.eventMsg.companyName + "\n"),
		)
		b.WriteString(
			lipgloss.NewStyle().Render("Data atual: " + now.Local().Format("02/01/2006") + "\n"),
		)
		b.WriteString(
			lipgloss.NewStyle().Render("Expediente: " + m.eventMsg.timeTable + "\n"),
		)
		b.WriteString(fmt.Sprintf("Registros computados: %d\n", m.punchCount))

		maybeTodayClock, exists := m.eventMsg.clocking[todayKey]

		if exists {
			t := tree.Root(".")
			for _, event := range maybeTodayClock {
				t.Child(strings.Split(event.time, ".")[0] + " " + event.platform)
			}
			b.WriteString(t.String() + "\n")
		}

		b.WriteString(
			lipgloss.NewStyle().
				Width(defaultWidth).
				Bold(true).
				Foreground(lipgloss.Color(clockWerkColor)).
				PaddingLeft(2).
				Render(figure.NewFigure(timeStr, "starwars", true).String() + "\n"),
		)

		if m.forgetForm != nil {
			b.WriteString("\n")
			b.WriteString(m.forgetForm.View())
		} else if m.punchForm != nil {
			b.WriteString("\nConfirma o registro de ponto?\n")
			b.WriteString(m.punchForm.View())
		} else {
			b.WriteString("\n")
			b.WriteString(m.help.View(m.keys))
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
