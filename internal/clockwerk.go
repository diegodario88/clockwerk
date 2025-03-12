package internal

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/diegodario88/clockwerk/internal/core"
	"github.com/diegodario88/clockwerk/internal/ui"
)

type tickMsg struct{}

type clockingMsg struct {
	id        string
	date      string
	time      string
	platform  string
	eventTime time.Time
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

type clockTimer struct {
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
	failedMsg        FailedMsg
	loginMsg         LoginMsg
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
}

func NewClockTimer() clockTimer {
	sp := spinner.New()
	sp.Spinner = spinner.Points
	sp.Style = lipgloss.NewStyle().
		PaddingLeft(1).
		Foreground(lipgloss.Color(core.ClockWerkColor))

	p := paginator.New()
	p.Type = paginator.Dots
	p.ActiveDot = lipgloss.NewStyle().
		Foreground(lipgloss.Color(core.ClockWerkColor)).
		Padding(0, 1).
		Render("● ")
	p.InactiveDot = lipgloss.NewStyle().
		Padding(0, 1).
		Render("○ ")
	p.SetTotalPages(3)

	helpModel := help.New()

	creds, err := core.LoadCredentials()
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

	return clockTimer{
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
	}
}

func (m clockTimer) Init() tea.Cmd {
	tea.SetWindowTitle("Clockwerk")

	core.Theme.Focused.Base = lipgloss.NewStyle().
		PaddingLeft(1).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color(core.ClockWerkColor)).
		BorderLeft(true)

	core.Theme.Focused.FocusedButton = core.Theme.Focused.FocusedButton.
		Background(lipgloss.Color(core.AmberFlare))

	if m.step == 0 {
		return m.cpfForm.Init()
	} else if m.step == 4 {
		return tea.Batch(handleGetClockingEvent(m.token), m.spinner.Tick)
	}

	return nil
}

func (m clockTimer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return dispatchWindowSizeChange(msg, &m)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		}
	}

	switch m.step {
	case 0:
		return dispatchInputCpf(msg, &m)
	case 1:
		return dispatchInputPassword(msg, &m)
	case 2:
		return dispatchInputKeep(msg, &m)
	case 3:
		return dispatchAuthSpinner(msg, &m)
	case 4:
		return dispatchEventsSpinner(msg, &m)
	case 5:
		return dispatchDashboard(msg, &m)
	case 6:
		return dispatchPunch(msg, &m)
	default:
		return m, nil
	}
}

func (m clockTimer) View() string {
	if m.tooSmall {
		return renderWindowSizeWarn(&m)
	}

	switch m.step {
	case 0:
		return renderCpfStep(&m)
	case 1:
		return renderPasswordStep(&m)
	case 2:
		return renderKeepStep(&m)
	case 3:
		return renderAuthStep(&m)
	case 4:
		return renderEventsStep(&m)
	case 5:
		return renderDashboardStep(&m)
	case 6:
		return renderPunchStep(&m)
	}

	return "Não deveria ser exibido isso ..."
}
