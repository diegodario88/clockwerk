package internal

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/diegodario88/clockwerk/internal/core"
	"github.com/diegodario88/clockwerk/internal/ui"
)

func dispatchWindowSizeChange(msg tea.WindowSizeMsg, m *clockTimer) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	if m.width < core.AppWidth || m.height < core.AppHeight {
		m.tooSmall = true
	} else {
		m.tooSmall = false
	}

	return m, nil
}

func dispatchInputCpf(msg tea.Msg, m *clockTimer) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
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
}

func dispatchInputPassword(msg tea.Msg, m *clockTimer) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
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
}

func dispatchInputKeep(msg tea.Msg, m *clockTimer) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
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
}

func dispatchAuthSpinner(msg tea.Msg, m *clockTimer) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)

	switch msg := msg.(type) {
	case FailedMsg:
		m.failedMsg = msg
		return m, nil

	case LoginMsg:
		m.loginMsg = msg
		m.step = 4
		m.token = msg.token
		m.failedMsg = FailedMsg{error: ""}

		if m.keepLogged {
			creds := core.UserCredentials{
				Domain:   m.domain,
				CPF:      m.cpf,
				Password: m.password,
				Token:    msg.token,
			}
			if err := core.SaveCredentials(creds); err != nil {
				log.Printf("Erro ao salvar credenciais: %v", err)
			}
		}
		return m, tea.Batch(handleGetClockingEvent(m.token), m.spinner.Tick)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Retry):
			if m.failedMsg.error != "" {
				m.failedMsg = FailedMsg{error: ""}
				m.step = 0
				m.paginator.PrevPage()
				m.paginator.PrevPage()
				m.cpfForm = ui.NewCPFForm(m.cpfForm.GetString("domain"), m.cpfForm.GetString("cpf"))
				return m, m.cpfForm.Init()
			}
		}
	}

	return m, cmd
}

func dispatchEventsSpinner(msg tea.Msg, m *clockTimer) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)

	switch msg := msg.(type) {
	case FailedMsg:
		if strings.Contains(msg.error, "Unauthorized") && !m.hasAuthRecover {
			core.DeleteCredentials()
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
		maybeTodayClock, exists := m.eventMsg.clocking[core.TodayKey]
		if exists {
			m.punchCount = len(maybeTodayClock)
			if len(maybeTodayClock)%2 != 0 {
				m.timerRunning = true
				loc := time.FixedZone("UTC-3", -3*3600)
				nowInUTC3 := time.Now().In(loc)
				formatted := nowInUTC3.Format(core.TimeLayout)
				parsedTime, err := time.ParseInLocation(core.TimeLayout, formatted, loc)
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
				m.failedMsg = FailedMsg{error: ""}
				m.cpfForm = ui.NewCPFForm(m.cpfForm.GetString("domain"), m.cpfForm.GetString("cpf"))
				return m, m.cpfForm.Init()
			}
		}
	}

	return m, cmd
}

func dispatchDashboard(msg tea.Msg, m *clockTimer) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.forgetForm != nil && m.activeTab == 0 {
		updatedForm, c := m.forgetForm.Update(msg)
		if f, ok := updatedForm.(*huh.Form); ok {
			m.forgetForm = f
		}
		cmd = c
		if m.forgetForm.State == huh.StateCompleted {
			if m.forgetForm.GetBool("confirm") {
				if err := core.DeleteCredentials(); err != nil {
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
			maybeTodayClock, exists := m.eventMsg.clocking[core.TodayKey]
			if exists {
				lastPunchTime := maybeTodayClock[m.punchCount-1].eventTime
				currentElapsed := time.Since(lastPunchTime)

				if currentElapsed.Hours() >= 4 {
					if m.lastNotification.IsZero() || time.Since(m.lastNotification) >= 20*time.Minute {
						ce := currentElapsed
						go func(elapsed time.Duration) {
							message, urgency := handleCreateMessageNotification(elapsed)
							handleDesktopNotification("Alerta Clockwerk", message, urgency, m.dbusCon)
						}(ce)
						m.lastNotification = time.Now()
					}
				}
			}

			return m, tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg{} })
		}
		return m, nil

	case FailedMsg:
		m.step = 4
		m.failedMsg = msg
		return m, nil
	}

	return m, nil
}

func dispatchPunch(msg tea.Msg, m *clockTimer) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)

	switch msg.(type) {
	case PostClockingMsg:
		m.step = 4
		return m, tea.Batch(handleGetClockingEvent(m.token), m.spinner.Tick)
	}

	return m, cmd
}
