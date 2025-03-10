package main

import (
	_ "embed"
	"fmt"
	"log"
	"math"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NimbleMarkets/ntcharts/barchart"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
	"github.com/common-nighthawk/go-figure"
	"github.com/diegodario88/clockwerk/config"
	"github.com/diegodario88/clockwerk/senior"
	"github.com/diegodario88/clockwerk/storage"
	"github.com/diegodario88/clockwerk/ui"
	"github.com/godbus/dbus/v5"
)

type keyMap struct {
	Punch       key.Binding
	ForgetCreds key.Binding
	Quit        key.Binding
	Exit        key.Binding
	MoveBack    key.Binding
	MoveForward key.Binding
	Retry       key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.MoveBack,
		k.MoveForward,
		k.Punch,
		k.ForgetCreds,
		k.Exit,
		k.Quit,
	}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			k.MoveBack,
			k.MoveForward,
			k.Punch,
			k.ForgetCreds,
			k.Exit,
			k.Quit,
		},
	}
}

var keys = keyMap{
	MoveBack: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "Anterior"),
	),
	MoveForward: key.NewBinding(
		key.WithKeys("right", "l", "tab"),
		key.WithHelp("→/l", "Próxima"),
	),
	Punch: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("<space>", "Ponto"),
	),
	ForgetCreds: key.NewBinding(
		key.WithKeys("e", "E"),
		key.WithHelp("<e>", "Esquecer"),
	),
	Retry: key.NewBinding(
		key.WithKeys("r", "R"),
		key.WithHelp("<r>", "Tentar novamente"),
	),
	Exit: key.NewBinding(
		key.WithKeys("q", "Q"),
		key.WithHelp("<q>", "Fechar"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("<ctrl+c>", "Sair"),
	),
}

type customHelp []key.Binding

func (c customHelp) ShortHelp() []key.Binding {
	return []key.Binding(c)
}

func (c customHelp) FullHelp() [][]key.Binding {
	return [][]key.Binding{c}
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

type ClockTimer struct {
	step             int
	punchCount       int
	activeTab        int
	keepLogged       bool
	timerRunning     bool
	tickScheduled    bool
	hasAuthRecover   bool
	shoudNotify      bool
	domain           string
	cpf              string
	password         string
	token            string
	cpfForm          *huh.Form
	passwordForm     *huh.Form
	keepForm         *huh.Form
	punchForm        *huh.Form
	forgetForm       *huh.Form
	failedMsg        failedMsg
	loginMsg         loginMsg
	eventMsg         eventMsg
	spinner          spinner.Model
	paginator        paginator.Model
	elapsed          time.Duration
	lastNotification time.Time
	help             help.Model
	keys             keyMap
	width            int
	height           int
	tooSmall         bool
	dbusCon          *dbus.Conn
}

var (
	version           = "development"
	clockwerkIconPath string
	createIconOnce    sync.Once
	cleanupIconOnce   sync.Once
)

//go:embed assets/clockwerk.png
var clockwerkIcon []byte

func getNotificationMessage(elapsed time.Duration) (message string, urgency string) {
	hours := elapsed.Hours()
	formattedTime := formatHoursAsHHMM(hours)
	text := fmt.Sprintf("Você está trabalhando há %s sem intervalo.\n", formattedTime)

	if hours >= 6 {
		alert := "🚨 Conforme Art. 71 da CLT, é obrigatório um intervalo mínimo de 1 hora para jornadas acima de 6 horas."
		return text + alert, "critical"
	}
	if hours >= 5 {
		alert := "⚠️ Conforme Art. 71 da CLT, é recomendável um intervalo de 15 minutos para jornadas entre 4 e 6 horas."
		return text + alert, "normal"
	}

	alert := "💡 Pausas curtas são recomendadas para preservar sua saúde e produtividade."

	return text + alert, "low"
}

func sendNotification(title, message, urgency string, con *dbus.Conn) {
	if runtime.GOOS != "linux" {
		log.Printf("Notificações não suportadas neste sistema (%s)", runtime.GOOS)
		return
	}

	createIconOnce.Do(func() {
		if _, err := os.Stat("/tmp/clockwerk_icon.png"); err == nil {
			clockwerkIconPath = "/tmp/clockwerk_icon.png"
			return
		}

		tmpFile, err := os.Create("/tmp/clockwerk_icon.png")
		if err != nil {
			log.Printf("Erro ao criar arquivo temporário: %v", err)
			return
		}
		defer tmpFile.Close()

		if _, err := tmpFile.Write(clockwerkIcon); err != nil {
			log.Printf("Erro ao escrever ícone: %v", err)
			return
		}

		clockwerkIconPath = tmpFile.Name()
	})

	expireTime := "10000"
	category := "productivity.timetracking"
	if urgency == "critical" {
		expireTime = "30000"
	}

	obj := con.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")
	call := obj.Call(
		"org.freedesktop.Notifications.Notify",
		0,
		"Clockwerk",
		uint32(0),
		clockwerkIconPath,
		title,
		message,
		[]string{},
		map[string]dbus.Variant{
			"urgency":        dbus.MakeVariant(urgency),
			"expire_timeout": dbus.MakeVariant(expireTime),
			"category":       dbus.MakeVariant(category),
		},
		int32(5000),
	)
	if call.Err != nil {
		log.Printf("Erro ao enviar notificação: %v", call.Err)
	}
}

func formatHoursAsHHMM(hours float64) string {
	totalMinutes := int(math.Round(hours * 60))
	h := totalMinutes / 60
	m := totalMinutes % 60
	return fmt.Sprintf("%dh%02dm", h, m)
}

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
				config.TimeLayout,
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

func NewClockTimer(con *dbus.Conn) ClockTimer {
	sp := spinner.New()
	sp.Spinner = spinner.Points
	sp.Style = lipgloss.NewStyle().
		PaddingLeft(1).
		Foreground(lipgloss.Color(config.ClockWerkColor))

	p := paginator.New()
	p.Type = paginator.Dots
	p.ActiveDot = lipgloss.NewStyle().
		Foreground(lipgloss.Color(config.ClockWerkColor)).
		Padding(0, 1).
		Render("● ")
	p.InactiveDot = lipgloss.NewStyle().
		Padding(0, 1).
		Render("○ ")
	p.SetTotalPages(3)

	helpModel := help.New()

	creds, err := storage.LoadCredentials()
	initialStep := 0
	initialDomain := ""
	initialCPF := ""
	initialPassword := ""
	initialToken := ""

	if err == nil && creds.Domain != "" && creds.CPF != "" && creds.Password != "" &&
		creds.Token != "" {
		initialStep = 4
		initialDomain = creds.Domain
		initialCPF = creds.CPF
		initialPassword = creds.Password
		initialToken = creds.Token
	}

	return ClockTimer{
		step:         initialStep,
		domain:       initialDomain,
		cpf:          initialCPF,
		password:     initialPassword,
		token:        initialToken,
		cpfForm:      ui.NewCPFForm(initialDomain, initialCPF),
		passwordForm: ui.NewPasswordForm(initialPassword),
		keepForm:     ui.NewKeepForm(true),
		punchForm:    nil,
		forgetForm:   nil,
		spinner:      sp,
		paginator:    p,
		timerRunning: false,
		elapsed:      0,
		punchCount:   0,
		keepLogged:   true,
		help:         helpModel,
		keys:         keys,
		activeTab:    0,
		dbusCon:      con,
	}
}

func (m ClockTimer) Init() tea.Cmd {
	tea.SetWindowTitle("Clockwerk")

	config.Theme.Focused.Base = lipgloss.NewStyle().
		PaddingLeft(1).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color(config.ClockWerkColor)).
		BorderLeft(true)

	if m.step == 0 {
		return m.cpfForm.Init()
	} else if m.step == 4 {
		return tea.Batch(handleGetClockingEvent(m.token), m.spinner.Tick)
	}

	return nil
}

func (m ClockTimer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.width < config.DefaultWidth || m.height < config.DefaultHeight {
			m.tooSmall = true
		} else {
			m.tooSmall = false
		}
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
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
			m.domain = m.cpfForm.GetString("domain")
			m.step = 1
			m.paginator.NextPage()
			m.passwordForm = ui.NewPasswordForm(m.passwordForm.GetString("password"))
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
				m.paginator.PrevPage()
				m.cpfForm = ui.NewCPFForm(m.cpfForm.GetString("domain"), m.cpfForm.GetString("cpf"))
				return m, m.cpfForm.Init()
			}
			m.password = m.passwordForm.GetString("password")
			m.step = 2
			m.paginator.NextPage()
			m.keepForm = ui.NewKeepForm(m.keepLogged)
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
				m.paginator.PrevPage()
				m.passwordForm = ui.NewPasswordForm(m.passwordForm.GetString("password"))
				m.keepLogged = m.keepForm.GetBool("keep")
				return m, m.passwordForm.Init()
			}
			m.keepLogged = m.keepForm.GetBool("keep")
			m.step = 3
			return m, tea.Batch(
				handleAuthentication(fmt.Sprintf("%s@%s", m.cpf, m.domain), m.password),
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
			m.token = msg.token
			m.failedMsg = failedMsg{error: ""}

			if m.keepLogged {
				creds := storage.UserCredentials{
					Domain:   m.domain,
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
			switch {
			case key.Matches(msg, m.keys.Retry):
				if m.failedMsg.error != "" {
					m.failedMsg = failedMsg{error: ""}
					m.step = 0
					m.paginator.PrevPage()
					m.paginator.PrevPage()
					m.cpfForm = ui.NewCPFForm(m.cpfForm.GetString("domain"), m.cpfForm.GetString("cpf"))
					return m, m.cpfForm.Init()
				}
			}
		}
		return m, cmd

	case 4: // Etapa 5: Spinner (busca de eventos
		m.spinner, cmd = m.spinner.Update(msg)
		switch msg := msg.(type) {
		case failedMsg:
			if strings.Contains(msg.error, "Unauthorized") && !m.hasAuthRecover {
				storage.DeleteCredentials()
				m.step = 3
				m.hasAuthRecover = true
				return m, tea.Batch(
					handleAuthentication(fmt.Sprintf("%s@%s", m.cpf, m.domain), m.password),
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
			maybeTodayClock, exists := m.eventMsg.clocking[config.TodayKey]
			if exists {
				m.punchCount = len(maybeTodayClock)
				if len(maybeTodayClock)%2 != 0 {
					m.timerRunning = true
					loc := time.FixedZone("UTC-3", -3*3600)
					nowInUTC3 := time.Now().In(loc)
					formatted := nowInUTC3.Format(config.TimeLayout)
					parsedTime, err := time.ParseInLocation(config.TimeLayout, formatted, loc)
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
			switch {
			case key.Matches(msg, m.keys.Retry):
				if m.failedMsg.error != "" {
					m.step = 0
					m.failedMsg = failedMsg{error: ""}
					m.cpfForm = ui.NewCPFForm(m.cpfForm.GetString("domain"), m.cpfForm.GetString("cpf"))
					return m, m.cpfForm.Init()
				}
			}
		}
		return m, cmd

	case 5:
		// Tratamento para o formulário de esquecer credenciais
		if m.forgetForm != nil && m.activeTab == 0 {
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
				if m.timerRunning {
					return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
						return tickMsg{}
					})
				}
				return m, nil
			}

			return m, cmd
		}

		// Tratamento para o formulário de confirmação de ponto
		if m.punchForm != nil && m.activeTab == 0 {
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
				if m.activeTab != 0 {
					return m, nil
				}
				m.punchForm = ui.NewPunchConfirmForm()
				return m, m.punchForm.Init()
			case key.Matches(msg, m.keys.ForgetCreds):
				if m.activeTab != 0 {
					return m, nil
				}
				m.forgetForm = ui.NewForgetForm()
				return m, m.forgetForm.Init()
			case key.Matches(msg, m.keys.MoveBack):
				if m.activeTab > 0 {
					m.activeTab--
				} else {
					m.activeTab = 2
				}
				return m, nil
			case key.Matches(msg, m.keys.MoveForward):
				m.activeTab = (m.activeTab + 1) % 3
				return m, nil
			case key.Matches(msg, m.keys.Quit):
				return m, tea.Quit
			case key.Matches(msg, m.keys.Exit):
				return m, tea.Quit
			}
		case tickMsg:
			if m.timerRunning {
				m.elapsed += time.Second
				maybeTodayClock, exists := m.eventMsg.clocking[config.TodayKey]
				if exists {
					lastPunchTime := maybeTodayClock[m.punchCount-1].eventTime
					currentElapsed := time.Since(lastPunchTime)

					if currentElapsed.Hours() >= 4 {
						if m.lastNotification.IsZero() || time.Since(m.lastNotification) >= 20*time.Minute {
							ce := currentElapsed
							go func(elapsed time.Duration) {
								message, urgency := getNotificationMessage(elapsed)
								sendNotification("Alerta Clockwerk", message, urgency, m.dbusCon)
							}(ce)
							m.lastNotification = time.Now()
						}
					}
				}

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

func (m ClockTimer) View() string {
	var b strings.Builder
	if m.tooSmall {
		b.WriteString(lipgloss.NewStyle().
			Bold(true).
			Render("A janela do terminal é muito pequena!") + "\n")
		b.WriteString(lipgloss.NewStyle().
			Italic(true).
			Render(fmt.Sprintf("Largura = %d -> Ideal = %d", m.width, config.DefaultWidth)) +
			"\n")
		b.WriteString(lipgloss.NewStyle().
			Italic(true).
			Render(fmt.Sprintf("Altura = %d -> Ideal = %d", m.height, config.DefaultHeight)) + "\n")
		return b.String()
	}

	switch m.step {
	case 0:
		b.WriteString(lipgloss.NewStyle().
			Bold(true).
			Render("Autenticação - Etapa 1/3: Identificação") + "\n\n")
		b.WriteString(m.cpfForm.View() + "\n\n")
		b.WriteString(
			lipgloss.NewStyle().
				Width(config.DefaultWidth).
				AlignHorizontal(lipgloss.Center).
				Render(m.paginator.View()),
		)

	case 1:
		loginEmail := fmt.Sprintf("%s@%s", m.cpf, m.domain)
		b.WriteString(
			lipgloss.NewStyle().Bold(true).Render("Autenticação - Etapa 2/3: Senha") + "\n\n",
		)
		b.WriteString(
			"Domínio: " +
				lipgloss.NewStyle().Italic(true).Render(m.domain) + "\n",
		)
		b.WriteString(
			"CPF:     " +
				lipgloss.NewStyle().Italic(true).Render(m.cpf) + "\n",
		)
		b.WriteString(
			"Login:   " +
				lipgloss.NewStyle().Italic(true).Render(loginEmail) + "\n\n",
		)
		b.WriteString(m.passwordForm.View() + "\n\n")
		b.WriteString(
			lipgloss.NewStyle().
				Width(config.DefaultWidth).
				AlignHorizontal(lipgloss.Center).
				Render(m.paginator.View()),
		)

	case 2:
		b.WriteString(lipgloss.NewStyle().
			Bold(true).
			Render("Autenticação - Etapa 3/3: Manter Logado") + "\n\n")
		b.WriteString(m.keepForm.View() + "\n\n")
		b.WriteString(
			"Informações serão persistidas em " +
				lipgloss.NewStyle().
					Bold(true).
					Italic(true).
					Render(storage.GetCredentialsFilePath()),
		)
		b.WriteString("\n")
		b.WriteString(
			lipgloss.NewStyle().
				Italic(true).
				Render("* Armazenaremos suas credenciais de forma criptografada utilizando AES-256."),
		)
		b.WriteString("\n\n")
		b.WriteString(
			lipgloss.NewStyle().
				Width(config.DefaultWidth).
				AlignHorizontal(lipgloss.Center).
				Render(m.paginator.View()),
		)

	case 3:
		limitedHelp := customHelp{keys.Retry, keys.Quit}

		if m.failedMsg.error != "" {
			b.WriteString(
				lipgloss.NewStyle().
					Bold(true).
					Render("Ops.. Falha no processo de autenticação") +
					"\n\n",
			)
			b.WriteString(
				lipgloss.NewStyle().
					Width(config.DefaultWidth).
					AlignHorizontal(lipgloss.Center).
					Italic(true).
					Render(fmt.Sprintf("Login: %s@%s", m.cpf, m.domain)) + "\n",
			)
			b.WriteString(
				lipgloss.NewStyle().
					Width(config.DefaultWidth).
					AlignHorizontal(lipgloss.Center).
					Italic(true).
					Render(fmt.Sprintf("Mensagem: %s", m.failedMsg.error)) +
					"\n\n",
			)
			b.WriteString(
				lipgloss.NewStyle().
					Width(config.DefaultWidth).
					AlignHorizontal(lipgloss.Center).
					Blink(true).
					Foreground(lipgloss.Color(config.ClockWerkColor)).
					Render(
						"¯\\_(ツ)_/¯",
					) + "\n\n",
			)
			b.WriteString(
				lipgloss.NewStyle().
					Width(config.DefaultWidth).
					AlignHorizontal(lipgloss.Center).
					Render(m.help.View(limitedHelp)),
			)
		} else {
			b.WriteString(lipgloss.NewStyle().Bold(true).Render("Autenticando...") + "\n\n")
			b.WriteString(
				lipgloss.NewStyle().
					Width(config.DefaultWidth).
					AlignHorizontal(lipgloss.Center).
					Render(m.spinner.View()),
			)
		}

	case 4:
		limitedHelp := customHelp{keys.Retry, keys.Quit}
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
					Render(fmt.Sprintf("Login: %s@%s", m.cpf, m.domain)) + "\n",
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
					Foreground(lipgloss.Color(config.ClockWerkColor)).
					Render(
						"¯\\_(ツ)_/¯",
					) + "\n\n",
			)
			b.WriteString(
				lipgloss.NewStyle().
					Width(config.DefaultWidth).
					AlignHorizontal(lipgloss.Center).
					Render(m.help.View(limitedHelp)),
			)
		} else {
			b.WriteString(lipgloss.NewStyle().Bold(true).Render("Buscando últimos eventos...") + "\n\n")
			b.WriteString(
				lipgloss.NewStyle().
					Width(config.DefaultWidth).
					AlignHorizontal(lipgloss.Center).
					Render(m.spinner.View()),
			)
		}

	case 5:
		now := time.Now()
		h := int(m.elapsed.Hours())
		mm := int(m.elapsed.Minutes()) % 60
		ss := int(m.elapsed.Seconds()) % 60
		timeStr := fmt.Sprintf("%02d:%02d:%02d", h, mm, ss)
		limitedHelp := customHelp{keys.MoveBack, keys.MoveForward, keys.Exit, keys.Quit}

		// Renderiza a linha de abas:
		activeTabStyle := lipgloss.NewStyle().
			Bold(true).
			Italic(true).
			Foreground(lipgloss.Color(config.ClockWerkColor))

		inactiveTabStyle := lipgloss.NewStyle().
			Foreground(lipgloss.NoColor{})

		tabs := []string{"Timer", "Histórico", "Sobre"}
		var tabsLine strings.Builder
		for i, tab := range tabs {
			if i == m.activeTab {
				tabsLine.WriteString(activeTabStyle.Render(fmt.Sprintf("[%s] ", tab)))
			} else {
				tabsLine.WriteString(inactiveTabStyle.Render(fmt.Sprintf(" %s  ", tab)))
			}
		}

		b.WriteString(
			lipgloss.NewStyle().
				Bold(true).
				Width(config.DefaultWidth).
				Render(" Registro de Ponto - Clockwerk") +
				"\n",
		)

		boxStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			Height(config.DefaultHeight).
			Width(config.DefaultWidth)

		var contentBuilder strings.Builder

		contentBuilder.WriteString(
			lipgloss.NewStyle().
				Width(config.DefaultWidth).
				AlignHorizontal(lipgloss.Center).
				Render(
					tabsLine.String(),
				),
		)

		contentBuilder.WriteString("\n\n")

		switch m.activeTab {
		case 0:
			lines := []string{
				"Colaborador:   " + m.eventMsg.employeeName,
				"Empresa:       " + m.eventMsg.companyName,
				"Data atual:    " + now.Local().Format("02/01/2006"),
				"Expediente:    " + m.eventMsg.timeTable,
				"Registros:     " + strconv.Itoa(m.punchCount),
			}

			contentBuilder.WriteString(strings.Join(lines, "\n"))
			maybeTodayClock, exists := m.eventMsg.clocking[config.TodayKey]

			if exists {
				t := tree.Root(".")
				for _, event := range maybeTodayClock {
					t.Child(strings.Split(event.time, ".")[0] + " " + event.platform)
				}
				contentBuilder.WriteString("\n" + t.String() + "\n")
			} else {
				contentBuilder.WriteString("\n")
			}

			asciiArt := figure.NewFigure(timeStr, "starwars", true).String()
			lines = strings.Split(asciiArt, "\n")
			maxWidth := 0

			for _, line := range lines {
				if len(line) > maxWidth {
					maxWidth = len(line)
				}
			}

			for i, line := range lines {
				lines[i] = lipgloss.NewStyle().Width(maxWidth).Render(line)
			}

			alignedArt := strings.Join(lines, "\n")
			contentBuilder.WriteString(
				lipgloss.NewStyle().
					Width(config.DefaultWidth).
					Bold(true).
					Foreground(lipgloss.Color(config.ClockWerkColor)).
					AlignHorizontal(lipgloss.Center).
					AlignVertical(lipgloss.Bottom).
					Render(alignedArt),
			)

			if m.forgetForm != nil {
				contentBuilder.WriteString("\n")
				contentBuilder.WriteString(m.forgetForm.View())
			} else if m.punchForm != nil {
				contentBuilder.WriteString(m.punchForm.View())
			} else {
				contentBuilder.WriteString("\n")
				contentBuilder.WriteString(
					lipgloss.NewStyle().
						Width(config.DefaultWidth).
						AlignHorizontal(lipgloss.Center).
						AlignVertical(lipgloss.Bottom).
						Render(m.help.View(m.keys)),
				)
			}
		case 1: // Aba "Histórico": conteúdo de exemplo para registros passados
			var dates []string
			for date := range m.eventMsg.clocking {
				dates = append(dates, date)
			}

			sort.Slice(dates, func(i, j int) bool {
				t1, err1 := time.Parse("02/01/2006", dates[i])
				t2, err2 := time.Parse("02/01/2006", dates[j])
				if err1 != nil || err2 != nil {
					return dates[i] > dates[j]
				}
				return t1.After(t2)
			})

			bc := barchart.New(config.DefaultWidth, config.DefaultBarHeight)

			for _, date := range dates {
				clockings := m.eventMsg.clocking[date]
				workedHours := 0.0
				copyClocking := make([]clockingMsg, len(clockings))
				copy(copyClocking, clockings)

				if len(copyClocking)%2 != 0 {
					copyClocking = copyClocking[:len(copyClocking)-1]
				}

				for i := 0; i < len(copyClocking)-1; i += 2 {
					startTime := copyClocking[i].eventTime
					endTime := copyClocking[i+1].eventTime
					workedHours += endTime.Sub(startTime).Hours()
				}

				shortDate := clockings[0].eventTime.Format("02/01")
				formattedTime := formatHoursAsHHMM(workedHours)

				var barColor string
				switch {
				case workedHours < 2:
					barColor = config.SunflowerYellow
				case workedHours <= 5:
					barColor = config.MintGreen
				case workedHours <= 8.5:
					barColor = config.ForestGreen
				default:
					barColor = config.LavaRed
				}

				bar := barchart.BarData{
					Label: fmt.Sprintf("%s (%s)", shortDate, formattedTime),
					Values: []barchart.BarValue{
						{
							Name:  date,
							Value: workedHours,
							Style: lipgloss.NewStyle().Foreground(lipgloss.Color(barColor)),
						},
					},
				}
				bc.Push(bar)
			}

			bc.Draw()
			contentBuilder.WriteString(
				lipgloss.NewStyle().
					Bold(true).
					Render("Histórico de ponto") + "\n",
			)
			contentBuilder.WriteString(
				lipgloss.NewStyle().
					Italic(true).
					Render("Confira um resumo dos registros dos últimos cinco dias (quando disponíveis). O gráfico mostra o total de horas trabalhadas em cada dia.") +
					"\n\n",
			)
			contentBuilder.WriteString("\n")
			contentBuilder.WriteString(bc.View())
			contentBuilder.WriteString("\n\n")
			contentBuilder.WriteString(
				lipgloss.NewStyle().
					Width(config.DefaultWidth).
					AlignHorizontal(lipgloss.Center).
					Render(m.help.View(limitedHelp)),
			)
		case 2: // Aba "Sobre": informações sobre o aplicativo
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)
			goVersion := runtime.Version()
			desktop := os.Getenv("XDG_CURRENT_DESKTOP")
			osInfo := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
			debugMode := "desativado"
			if len(os.Getenv("DEBUG")) > 0 {
				debugMode = "ativado"
			}

			contentBuilder.WriteString(
				lipgloss.NewStyle().Bold(true).Render("Sobre o Clockwerk") + "\n",
			)
			contentBuilder.WriteString(
				"Clockwerk é um exemplo de aplicativo TUI para controle de ponto.\n\n",
			)
			contentBuilder.WriteString(fmt.Sprintf("Versão:        %s\n", version))
			contentBuilder.WriteString(fmt.Sprintf("Depuração:     %s\n", debugMode))
			contentBuilder.WriteString(fmt.Sprintf("Go:            %s\n", goVersion))
			contentBuilder.WriteString(fmt.Sprintf("Sistema:       %s\n", osInfo))
			contentBuilder.WriteString(fmt.Sprintf("Desktop:       %s\n", desktop))
			contentBuilder.WriteString(fmt.Sprintf("CPU Núcleos:   %d\n", runtime.NumCPU()))
			contentBuilder.WriteString(fmt.Sprintf("Goroutines:    %d\n", runtime.NumGoroutine()))
			contentBuilder.WriteString(
				fmt.Sprintf("Uso Memória:   %.2f MB\n", float64(memStats.HeapAlloc)/1024/1024),
			)
			contentBuilder.WriteString("\n\n")

			contentBuilder.WriteString(
				lipgloss.NewStyle().
					Width(config.DefaultWidth).
					AlignHorizontal(lipgloss.Center).
					Bold(true).
					Italic(true).
					Render("Desenvolvido por diegodario88"),
			)

			contentBuilder.WriteString("\n\n")

			contentBuilder.WriteString(
				lipgloss.NewStyle().
					Width(config.DefaultWidth).
					AlignHorizontal(lipgloss.Center).
					Render(m.help.View(limitedHelp)),
			)
		}
		// Renderiza todo o conteúdo ao final
		b.WriteString(boxStyle.Render(contentBuilder.String()))
	case 6:
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Registrando evento ponto...") + "\n\n")
		b.WriteString(m.spinner.View())
	}

	return b.String()
}

func main() {
	var hasDebug = false
	if len(os.Getenv("DEBUG")) > 0 {
		hasDebug = true
	}

	if hasDebug {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
	}

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	clockTimer := NewClockTimer(conn)
	program := tea.NewProgram(clockTimer, tea.WithAltScreen())

	if _, err := program.Run(); err != nil {
		if hasDebug {
			log.Printf("Erro ao executar o programa: %s\n", err)
		}
		os.Exit(1)
	}

	program.Quit()
}
