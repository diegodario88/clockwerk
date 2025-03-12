package internal

import (
	_ "embed"
	"fmt"
	"log"
	"math"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/diegodario88/clockwerk/internal/core"
	"github.com/godbus/dbus/v5"
)

//go:embed ui/res/clockwerk.png
var clockwerkIcon []byte

var (
	versioR           = "development"
	clockwerkIconPath string
	createIconOnce    sync.Once
	cleanupIconOnce   sync.Once
)

type FailedMsg struct{ error string }
type LoginMsg struct{ token string }

type PostClockingMsg struct {
	dateEvent string
	timeEvent string
}

func handleCreateMessageNotification(elapsed time.Duration) (message string, urgency string) {
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

func handleDesktopNotification(title, message, urgency string) {
	if runtime.GOOS != "linux" {
		return
	}

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		panic(err)
	}
	defer conn.Close()

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

	obj := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")
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
		token, err := core.GatewayLogin(user, password)
		if err != nil {
			return FailedMsg{error: err.Error()}
		}

		return LoginMsg{token: token}
	}
}

func handleGetClockingEvent(token string) tea.Cmd {
	return func() tea.Msg {
		events, err := core.GetClockingEvents(token)
		if err != nil {
			return FailedMsg{error: err.Error()}
		}
		if len(events) == 0 {
			return FailedMsg{error: "lista de eventos vazia"}
		}

		grouped := make(map[string][]clockingMsg)
		for _, event := range events {
			timeStr := fmt.Sprintf("%s %s %s", event.DateEvent, event.TimeEvent, event.TimeZone)

			parsedTime, err := time.Parse(
				core.TimeLayout,
				timeStr,
			)

			if err != nil {
				return FailedMsg{
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
		cResp, err := core.PostClockingEvent(token, core.ClockingRequest{
			ClockingInfo: core.ClockingInfo{
				Company: core.ClockingCompany{
					ID:         event.companyId,
					ArpID:      event.companyArpId,
					Identifier: event.cnpj,
					Caepf:      event.caepf,
					CnoNumber:  event.cnoNumber,
				},
				Employee: core.ClockingEmployee{
					ID:    event.employeeId,
					ArpID: event.employeeArpId,
					Cpf:   event.cpf,
					Pis:   event.pis,
				},
				Signature: core.ClockingSignature{
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
			return FailedMsg{error: err.Error()}
		}

		return PostClockingMsg{
			dateEvent: cResp.Result.EventImported.DateEvent,
			timeEvent: cResp.Result.EventImported.DateEvent,
		}
	}
}
