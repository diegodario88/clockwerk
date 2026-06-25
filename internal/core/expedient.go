package core

import (
	"strconv"
	"strings"
	"time"
)

// ParseTimeTable converte uma string de expediente no formato
// "08:00-12:00-14:00-18:00" em uma lista de minutos-do-dia.
//
// Exige uma quantidade par de horários (>= 2), representando pares de
// entrada/saída de cada bloco. Caso a string não seja interpretável
// (vazia, ímpar ou formato inesperado), retorna ok=false.
func ParseTimeTable(s string) ([]int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, false
	}

	parts := strings.Split(s, "-")
	if len(parts) < 2 || len(parts)%2 != 0 {
		return nil, false
	}

	minutes := make([]int, 0, len(parts))
	for _, p := range parts {
		hm := strings.SplitN(strings.TrimSpace(p), ":", 2)
		if len(hm) != 2 {
			return nil, false
		}

		h, err := strconv.Atoi(strings.TrimSpace(hm[0]))
		if err != nil || h < 0 || h > 23 {
			return nil, false
		}

		m, err := strconv.Atoi(strings.TrimSpace(hm[1]))
		if err != nil || m < 0 || m > 59 {
			return nil, false
		}

		minutes = append(minutes, h*60+m)
	}

	return minutes, true
}

// AssumedBreak é a duração presumida de cada intervalo ainda não tirado no dia.
// O expediente do Senior costuma propor intervalos maiores (ex.: 2h de almoço),
// mas o horário é flexível e na prática o intervalo gira em torno de 1h. Os
// intervalos já realizados usam sempre a duração REAL das marcações; só os
// intervalos futuros (ainda não iniciados) caem nessa estimativa.
const AssumedBreak = time.Hour

// PredictExit calcula o horário previsto de saída.
//
// O expediente serve apenas para derivar QUANTO tempo de trabalho o usuário
// precisa cumprir (soma dos blocos de trabalho). A previsão ancora na PRIMEIRA
// marcação do dia e soma: trabalho total + intervalos. Os intervalos já
// ocorridos contam pela duração real das marcações; o intervalo em andamento
// conta pelo maior entre o tempo já decorrido e AssumedBreak; e cada intervalo
// futuro ainda não iniciado conta como AssumedBreak.
//
// Exemplo com expediente 08:00-12:00-14:00-18:00 (8h de trabalho, 1 intervalo):
//   - [08:30]                       -> 17:30 (8h + 1h de almoço presumido)
//   - [08:30, 12:15, 13:15]         -> 17:30 (almoço real de 1h)
//   - [08:30, 12:15, 13:45]         -> 18:00 (almoço real de 1h30)
//
// Quando a jornada já foi concluída (batidas >= horários do expediente),
// retorna a última batida como saída efetiva. Sem batidas, projeta a partir do
// início do expediente.
//
// exp deve ser a lista de minutos-do-dia (par) retornada por ParseTimeTable.
// punches devem ser as marcações de HOJE, ordenadas de forma crescente.
func PredictExit(exp []int, punches []time.Time, now time.Time) (time.Time, bool) {
	if len(exp) < 2 || len(exp)%2 != 0 {
		return time.Time{}, false
	}

	// Tempo de trabalho exigido = soma dos blocos de trabalho (pares do
	// expediente). Número de intervalos esperados = blocos de trabalho - 1.
	workBlocks := len(exp) / 2
	var work time.Duration
	for i := 0; i+1 < len(exp); i += 2 {
		work += time.Duration(exp[i+1]-exp[i]) * time.Minute
	}
	expectedBreaks := workBlocks - 1

	n := len(punches)

	if n == 0 {
		startMin := exp[0]
		anchor := time.Date(
			now.Year(), now.Month(), now.Day(),
			startMin/60, startMin%60, 0, 0, now.Location(),
		)
		return anchor.Add(work + time.Duration(expectedBreaks)*AssumedBreak), true
	}

	if n >= len(exp) {
		return punches[n-1], true
	}

	// Soma a duração dos intervalos. Os índices ímpares de marcação iniciam um
	// intervalo (saída para almoço etc.); só existem expectedBreaks intervalos.
	maxBreakStart := 2*expectedBreaks - 1
	var breaks time.Duration
	breaksSeen := 0
	for i := 1; i < n && i <= maxBreakStart; i += 2 {
		breaksSeen++
		if i+1 < n {
			breaks += punches[i+1].Sub(punches[i]) // intervalo concluído (real)
		} else {
			elapsed := now.Sub(punches[i]) // intervalo em andamento
			if elapsed < AssumedBreak {
				elapsed = AssumedBreak
			}
			breaks += elapsed
		}
	}

	remainingBreaks := expectedBreaks - breaksSeen
	if remainingBreaks > 0 {
		breaks += time.Duration(remainingBreaks) * AssumedBreak
	}

	return punches[0].Add(work + breaks), true
}
