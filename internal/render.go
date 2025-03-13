package internal

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/NimbleMarkets/ntcharts/barchart"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
	"github.com/common-nighthawk/go-figure"
	"github.com/diegodario88/clockwerk/internal/core"
)

func renderWindowSizeWarn(m *clockTimer) string {
	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().
		Bold(true).
		Render("A janela do terminal é muito pequena!") + "\n")

	b.WriteString(lipgloss.NewStyle().
		Italic(true).
		Render(fmt.Sprintf("Largura = %d -> Ideal = %d", m.width, core.AppWidth)) +
		"\n")

	b.WriteString(lipgloss.NewStyle().
		Italic(true).
		Render(fmt.Sprintf("Altura = %d -> Ideal = %d", m.height, core.AppHeight)) + "\n")

	return b.String()
}

func renderCpfStep(m *clockTimer) string {
	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().
		Bold(true).
		Render("Autenticação - Etapa 1/3: Identificação") + "\n\n")

	b.WriteString(m.cpfForm.View() + "\n\n")

	b.WriteString(
		lipgloss.NewStyle().
			Width(core.AppWidth).
			AlignHorizontal(lipgloss.Center).
			Render(m.paginator.View()),
	)

	return b.String()
}

func renderPasswordStep(m *clockTimer) string {
	var b strings.Builder

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
			Width(core.AppWidth).
			AlignHorizontal(lipgloss.Center).
			Render(m.paginator.View()),
	)

	return b.String()
}

func renderKeepStep(m *clockTimer) string {
	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().
		Bold(true).
		Render("Autenticação - Etapa 3/3: Manter Logado") + "\n\n")

	b.WriteString(m.keepForm.View() + "\n\n")

	b.WriteString(
		"Informações serão persistidas em " +
			lipgloss.NewStyle().
				Bold(true).
				Italic(true).
				Render(core.GetCredentialsFilePath()),
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
			Width(core.AppWidth).
			AlignHorizontal(lipgloss.Center).
			Render(m.paginator.View()),
	)

	return b.String()
}

func renderAuthStep(m *clockTimer) string {
	var b strings.Builder

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
				Width(core.AppWidth).
				AlignHorizontal(lipgloss.Center).
				Italic(true).
				Render(fmt.Sprintf("Login: %s@%s", m.cpf, m.domain)) + "\n",
		)

		b.WriteString(
			lipgloss.NewStyle().
				Width(core.AppWidth).
				AlignHorizontal(lipgloss.Center).
				Italic(true).
				Render(fmt.Sprintf("Mensagem: %s", m.failedMsg.error)) +
				"\n\n",
		)

		b.WriteString(
			lipgloss.NewStyle().
				Width(core.AppWidth).
				AlignHorizontal(lipgloss.Center).
				Blink(true).
				Foreground(lipgloss.Color(core.ClockWerkColor)).
				Render(
					"¯\\_(ツ)_/¯",
				) + "\n\n",
		)

		b.WriteString(
			lipgloss.NewStyle().
				Width(core.AppWidth).
				AlignHorizontal(lipgloss.Center).
				Render(m.help.View(limitedHelp)),
		)
	} else {
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Autenticando...") + "\n\n")
		s := lipgloss.NewStyle().Render(m.spinner.View())
		block := lipgloss.Place(core.AppWidth, core.AppHalfHeight, lipgloss.Center, lipgloss.Center, s)
		b.WriteString(block)
	}

	return b.String()
}

func renderEventsStep(m *clockTimer) string {
	var b strings.Builder

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
				Foreground(lipgloss.Color(core.ClockWerkColor)).
				Render(
					"¯\\_(ツ)_/¯",
				) + "\n\n",
		)

		b.WriteString(
			lipgloss.NewStyle().
				Width(core.AppWidth).
				AlignHorizontal(lipgloss.Center).
				Render(m.help.View(limitedHelp)),
		)
	} else {
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Buscando últimos eventos...") + "\n\n")
		s := lipgloss.NewStyle().Render(m.spinner.View())
		block := lipgloss.Place(core.AppWidth, core.AppHalfHeight, lipgloss.Center, lipgloss.Center, s)
		b.WriteString(block)
	}

	return b.String()
}

func renderDashboardStep(m *clockTimer) string {
	var b strings.Builder

	now := time.Now()
	h := int(m.elapsed.Hours())
	mm := int(m.elapsed.Minutes()) % 60
	ss := int(m.elapsed.Seconds()) % 60
	timeStr := fmt.Sprintf("%02d:%02d:%02d", h, mm, ss)
	limitedHelp := customHelp{keys.MoveBack, keys.MoveForward, keys.Exit, keys.Quit}

	activeTabStyle := lipgloss.NewStyle().
		Bold(true).
		Italic(true).
		Foreground(lipgloss.Color(core.ClockWerkColor))

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
			Width(core.AppWidth).
			Render(" Registro de Ponto - Clockwerk") +
			"\n",
	)

	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		Height(core.AppHeight).
		Width(core.AppWidth)

	var contentBuilder strings.Builder

	contentBuilder.WriteString(
		lipgloss.NewStyle().
			Width(core.AppWidth).
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
		maybeTodayClock, exists := m.eventMsg.clocking[core.TodayKey]

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
				Width(core.AppWidth).
				Bold(true).
				Foreground(lipgloss.Color(core.ClockWerkColor)).
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
					Width(core.AppWidth).
					AlignHorizontal(lipgloss.Center).
					AlignVertical(lipgloss.Bottom).
					Render(m.help.View(m.keys)),
			)
		}
	case 1:
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

		bc := barchart.New(core.AppWidth, core.AppHalfHeight)

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
				barColor = core.SunflowerYellow
			case workedHours <= 5:
				barColor = core.MintGreen
			case workedHours <= 8.5:
				barColor = core.Forest
			default:
				barColor = core.LavaRed
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
				Width(core.AppWidth).
				AlignHorizontal(lipgloss.Center).
				Render(m.help.View(limitedHelp)),
		)
	case 2:
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		goVersion := runtime.Version()
		desktop := "não reconhecido"
		if envDesktop := os.Getenv("XDG_CURRENT_DESKTOP"); envDesktop != "" {
			desktop = envDesktop
		}
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
		contentBuilder.WriteString(fmt.Sprintf("Versão:        %s\n", core.Version))
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
				Width(core.AppWidth).
				AlignHorizontal(lipgloss.Center).
				Bold(true).
				Italic(true).
				Render("Desenvolvido por diegodario88"),
		)

		contentBuilder.WriteString("\n\n")

		contentBuilder.WriteString(
			lipgloss.NewStyle().
				Width(core.AppWidth).
				AlignHorizontal(lipgloss.Center).
				Render(m.help.View(limitedHelp)),
		)
	}

	b.WriteString(boxStyle.Render(contentBuilder.String()))

	return b.String()
}

func renderPunchStep(m *clockTimer) string {
	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Registrando evento ponto...") + "\n\n")

	s := lipgloss.NewStyle().Render(m.spinner.View())

	block := lipgloss.Place(
		core.AppWidth,
		core.AppHalfHeight,
		lipgloss.Center,
		lipgloss.Center,
		s,
	)

	b.WriteString(block)

	return b.String()
}
