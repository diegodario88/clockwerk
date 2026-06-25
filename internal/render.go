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

	left := " Registro de Ponto - Clockwerk"

	var right string
	if m.refreshing {
		right = "⟳ atualizando… "
	} else if !m.nextRefresh.IsZero() {
		remaining := time.Until(m.nextRefresh)
		if remaining < 0 {
			remaining = 0
		}
		right = fmt.Sprintf(
			"↻ próx. atualização %02d:%02d ",
			int(remaining.Minutes()),
			int(remaining.Seconds())%60,
		)
	}

	gap := core.AppWidth - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	b.WriteString(
		lipgloss.NewStyle().
			Bold(true).
			Width(core.AppWidth).
			Render(left+strings.Repeat(" ", gap)+right) +
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
			"Colaborador:    " + m.eventMsg.employeeName,
			"Empresa:        " + m.eventMsg.companyName,
			"Data atual:     " + now.Local().Format("02/01/2006"),
			"Expediente:     " + m.eventMsg.timeTable,
		}

		if exp, ok := core.ParseTimeTable(m.eventMsg.timeTable); ok {
			var punches []time.Time
			for _, event := range m.eventMsg.clocking[core.TodayKey] {
				punches = append(punches, event.eventTime)
			}
			if predicted, ok := core.PredictExit(exp, punches, now); ok {
				lines = append(lines, "Saída prevista: "+predicted.Format("15:04"))
			}
		}

		lines = append(lines, "Registros:      "+strconv.Itoa(m.punchCount))

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
		subTabs := []string{"Semana", "Mês"}
		var subTabsLine strings.Builder
		for i, tab := range subTabs {
			if i == m.historyView {
				subTabsLine.WriteString(activeTabStyle.Render(fmt.Sprintf("[%s] ", tab)))
			} else {
				subTabsLine.WriteString(inactiveTabStyle.Render(fmt.Sprintf(" %s  ", tab)))
			}
		}

		contentBuilder.WriteString(
			lipgloss.NewStyle().Bold(true).Render("Histórico de ponto") + "\n",
		)
		contentBuilder.WriteString(
			lipgloss.NewStyle().
				Width(core.AppWidth).
				AlignHorizontal(lipgloss.Center).
				Render(subTabsLine.String()) + "\n\n",
		)

		var selected []string
		if m.historyView == 0 {
			selected = selectWeekDates(m.eventMsg.clocking)
		} else {
			selected = selectMonthDates(m.eventMsg.clocking, now)
		}

		var totalWorked, totalBalance time.Duration
		hasIncomplete := false
		for _, date := range selected {
			db := computeDayBalance(date, m.eventMsg.clocking[date], m.eventMsg.timeTable)
			totalWorked += db.worked
			if db.countsForBalance() {
				totalBalance += db.balance
			}
			if !db.complete {
				hasIncomplete = true
			}
		}

		if len(selected) == 0 {
			contentBuilder.WriteString(
				lipgloss.NewStyle().
					Italic(true).
					Render("Nenhuma marcação disponível para o período selecionado.") + "\n",
			)
		} else if m.historyView == 0 {
			contentBuilder.WriteString(
				lipgloss.NewStyle().
					Italic(true).
					Render("Últimos cinco dias úteis com marcações. A barra mostra as horas trabalhadas e o saldo do dia aparece acima de cada barra.") +
					"\n\n",
			)
			contentBuilder.WriteString(renderWeekChart(m.eventMsg.clocking, m.eventMsg.timeTable, selected))
			contentBuilder.WriteString("\n")
		} else {
			contentBuilder.WriteString(renderMonthTable(m.eventMsg.clocking, m.eventMsg.timeTable, selected))
			contentBuilder.WriteString("\n")
			if note := monthCoverageNote(m.eventMsg.clocking, now); note != "" {
				contentBuilder.WriteString(
					lipgloss.NewStyle().
						Italic(true).
						Foreground(lipgloss.Color(core.SunflowerYellow)).
						Render(note) + "\n",
				)
			}
		}

		// Rodapé de totais do período.
		balanceColor := neutralBalanceColor(totalBalance)
		contentBuilder.WriteString("\n")
		contentBuilder.WriteString(
			lipgloss.NewStyle().Bold(true).Render("Total trabalhado: ") +
				core.FormatDuration(totalWorked) +
				lipgloss.NewStyle().Bold(true).Render("    Saldo do período: ") +
				lipgloss.NewStyle().
					Foreground(lipgloss.Color(balanceColor)).
					Render(core.FormatSignedDuration(totalBalance)) + "\n",
		)

		if hasIncomplete {
			contentBuilder.WriteString(
				lipgloss.NewStyle().
					Italic(true).
					Foreground(lipgloss.Color(core.SunflowerYellow)).
					Render("⚠ Dias com marcações faltando não entram no saldo (aguardando ajuste).") + "\n",
			)
		}
		contentBuilder.WriteString("\n")

		historyHelp := customHelp{
			keys.MoveBack,
			keys.MoveForward,
			keys.ToggleHistoryView,
			keys.Exit,
			keys.Quit,
		}
		contentBuilder.WriteString(
			lipgloss.NewStyle().
				Width(core.AppWidth).
				AlignHorizontal(lipgloss.Center).
				Render(m.help.View(historyHelp)),
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

type historyDayBalance struct {
	when     time.Time
	worked   time.Duration
	balance  time.Duration
	hasExp   bool
	complete bool // >= 4 marcações
}

// countsForBalance: só dias completos e com expediente válido entram no saldo.
func (d historyDayBalance) countsForBalance() bool {
	return d.hasExp && d.complete
}

func computeDayBalance(dateKey string, clockings []clockingMsg, timeTable string) historyDayBalance {
	punches := make([]time.Time, len(clockings))
	for i, c := range clockings {
		punches[i] = c.eventTime
	}

	db := historyDayBalance{
		worked:   core.WorkedDuration(punches),
		complete: len(clockings) >= 4,
	}
	if len(clockings) > 0 {
		db.when = clockings[0].eventTime
	}
	if exp, ok := core.ExpectedDailyWork(timeTable); ok {
		db.hasExp = true
		db.balance = db.worked - exp

		// Hoje pode estar em andamento: ignora saldo negativo (não é débito real).
		if dateKey == core.TodayKey && db.balance < 0 {
			db.hasExp = false
			db.balance = 0
		}
	}

	return db
}

func parseDateKey(key string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", key)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// selectWeekDates: últimos 5 dias úteis com marcações, do mais recente ao mais antigo.
func selectWeekDates(clocking map[string][]clockingMsg) []string {
	var dates []string
	for date := range clocking {
		if hideTodayWithoutLunch(date, clocking[date]) {
			continue
		}
		if t, ok := parseDateKey(date); ok {
			wd := t.Weekday()
			if wd == time.Saturday || wd == time.Sunday {
				continue
			}
		}
		dates = append(dates, date)
	}

	sort.Sort(sort.Reverse(sort.StringSlice(dates)))
	if len(dates) > 5 {
		dates = dates[:5]
	}
	return dates
}

// selectMonthDates: dias do mês calendário de `now` com marcações, em ordem crescente.
func selectMonthDates(clocking map[string][]clockingMsg, now time.Time) []string {
	var dates []string
	for date := range clocking {
		if hideTodayWithoutLunch(date, clocking[date]) {
			continue
		}
		t, ok := parseDateKey(date)
		if !ok {
			continue
		}
		if t.Year() == now.Year() && t.Month() == now.Month() {
			dates = append(dates, date)
		}
	}

	sort.Strings(dates)
	return dates
}

// hideTodayWithoutLunch oculta o dia de hoje enquanto tiver menos de 2 marcações.
func hideTodayWithoutLunch(date string, clockings []clockingMsg) bool {
	return date == core.TodayKey && len(clockings) < 2
}

// renderWeekChart desenha o gráfico da semana com o saldo do dia acima de cada barra.
func renderWeekChart(clocking map[string][]clockingMsg, timeTable string, dates []string) string {
	bc := barchart.New(core.AppWidth, core.AppHalfHeight)

	// barWidth replica o cálculo do barchart (AutoBarWidth + barGap=1) para
	// alinhar a linha de saldo e os rótulos ao centro de cada barra.
	const barGap = 1
	n := len(dates)
	barWidth := core.AppWidth
	if n > 0 {
		barWidth = (core.AppWidth - (n-1)*barGap) / n
	}
	if barWidth < 1 {
		barWidth = 1
	}

	var balanceLine strings.Builder
	for i, date := range dates {
		db := computeDayBalance(date, clocking[date], timeTable)
		workedHours := db.worked.Hours()
		shortDate := db.when.Format("02/01")

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

		saldo := "—"
		saldoColor := core.ClockWerkColor
		if !db.complete {
			saldo = "⚠ faltam"
			saldoColor = core.SunflowerYellow
		} else if db.hasExp {
			saldo = core.FormatSignedDuration(db.balance)
			saldoColor = neutralBalanceColor(db.balance)
		}

		if i > 0 {
			balanceLine.WriteString(strings.Repeat(" ", barGap))
		}
		balanceLine.WriteString(
			lipgloss.NewStyle().
				Width(barWidth).
				Align(lipgloss.Center).
				Bold(true).
				Foreground(lipgloss.Color(saldoColor)).
				Render(saldo),
		)

		label := centerPlainText(fmt.Sprintf("%s %s", shortDate, core.FormatDuration(db.worked)), barWidth)

		bc.Push(barchart.BarData{
			Label: label,
			Values: []barchart.BarValue{
				{
					Name:  date,
					Value: workedHours,
					Style: lipgloss.NewStyle().Foreground(lipgloss.Color(barColor)),
				},
			},
		})
	}

	bc.Draw()

	return lipgloss.NewStyle().Bold(true).Render("Saldo por dia") + "\n" +
		balanceLine.String() + "\n" + bc.View()
}

// centerPlainText centraliza com espaços (sem estilo) numa largura fixa, truncando se exceder.
func centerPlainText(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	pad := width - len(s)
	left := pad / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", pad-left)
}

// renderMonthTable monta a tabela Data | Trabalhado | Saldo | Marcações do mês.
func renderMonthTable(clocking map[string][]clockingMsg, timeTable string, dates []string) string {
	var b strings.Builder

	const (
		wData   = 12
		wWorked = 12
		wSaldo  = 21
	)
	col := func(w int) lipgloss.Style { return lipgloss.NewStyle().Width(w) }

	b.WriteString(
		col(wData).Bold(true).Render("Data") +
			col(wWorked).Bold(true).Render("Trabalhado") +
			col(wSaldo).Bold(true).Render("Saldo") +
			lipgloss.NewStyle().Bold(true).Render("Marcações") + "\n",
	)

	for _, date := range dates {
		db := computeDayBalance(date, clocking[date], timeTable)

		saldo := "—"
		saldoStyle := col(wSaldo)
		if !db.complete {
			saldo = "⚠ faltam marcações"
			saldoStyle = saldoStyle.Foreground(lipgloss.Color(core.SunflowerYellow))
		} else if db.hasExp {
			saldo = core.FormatSignedDuration(db.balance)
			saldoStyle = saldoStyle.Foreground(lipgloss.Color(neutralBalanceColor(db.balance)))
		}

		marks := make([]string, 0, len(clocking[date]))
		for _, c := range clocking[date] {
			marks = append(marks, c.eventTime.Format("15:04"))
		}

		row := col(wData).Render(db.when.Format("02/01/2006")) +
			col(wWorked).Render(core.FormatDuration(db.worked)) +
			saldoStyle.Render(saldo) +
			lipgloss.NewStyle().Render(strings.Join(marks, " "))
		b.WriteString(row + "\n")
	}

	return b.String()
}

// monthCoverageNote sinaliza quando os dados não cobrem o início do mês (limite da API).
func monthCoverageNote(clocking map[string][]clockingMsg, now time.Time) string {
	earliest := ""
	for date := range clocking {
		if earliest == "" || date < earliest {
			earliest = date
		}
	}
	if earliest == "" {
		return ""
	}

	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	earliestTime, ok := parseDateKey(earliest)
	if !ok {
		return ""
	}

	if earliestTime.After(firstOfMonth) && now.Day() > 1 {
		return fmt.Sprintf(
			"Histórico parcial: dados a partir de %s (limite da API).",
			earliestTime.Format("02/01"),
		)
	}
	return ""
}

// neutralBalanceColor: verde se positivo, vermelho se negativo, neutro se zero.
func neutralBalanceColor(balance time.Duration) string {
	switch {
	case balance > 0:
		return core.MintGreen
	case balance < 0:
		return core.LavaRed
	default:
		return core.ClockWerkColor
	}
}
