package core

import (
	"fmt"
	"time"
)

// WorkedDuration soma a duração dos pares de marcações (ordenadas). Uma
// marcação ímpar (jornada em andamento) é descartada.
func WorkedDuration(punches []time.Time) time.Duration {
	n := len(punches)
	if n%2 != 0 {
		n--
	}

	var worked time.Duration
	for i := 0; i+1 < n; i += 2 {
		worked += punches[i+1].Sub(punches[i])
	}

	return worked
}

// ExpectedDailyWork soma a jornada diária esperada (blocos de trabalho do
// timeTable). ok=false quando o expediente não é interpretável.
func ExpectedDailyWork(timeTable string) (time.Duration, bool) {
	exp, ok := ParseTimeTable(timeTable)
	if !ok {
		return 0, false
	}

	var work time.Duration
	for i := 0; i+1 < len(exp); i += 2 {
		work += time.Duration(exp[i+1]-exp[i]) * time.Minute
	}

	return work, true
}

// FormatDuration formata uma duração sem sinal no padrão "8h00m".
func FormatDuration(d time.Duration) string {
	totalMinutes := int(d.Round(time.Minute) / time.Minute)
	if totalMinutes < 0 {
		totalMinutes = -totalMinutes
	}
	return fmt.Sprintf("%dh%02dm", totalMinutes/60, totalMinutes%60)
}

// FormatSignedDuration formata uma duração com sinal: "+1h30m", "-0h45m".
func FormatSignedDuration(d time.Duration) string {
	sign := "+"
	if d < 0 {
		sign = "-"
	}
	return sign + FormatDuration(d)
}
