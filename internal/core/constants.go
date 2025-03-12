package core

import (
	"time"

	"github.com/charmbracelet/huh"
)

var (
	Theme          *huh.Theme = huh.ThemeBase()
	TodayKey                  = time.Now().Local().Format("2006-01-02")
	DefaultConfirm            = true
	Version                   = "development"
)

const (
	LavaRed         = "#fc4a1a"
	MintGreen       = "#a8ff78"
	Forest          = "#2F7336"
	SunflowerYellow = "#f7b733"
	AmberFlare      = "#F0A322"
	ClockWerkColor  = "#E28413"
	TimeLayout      = "2006-01-02 15:04:05.999 -07:00"
	AppWidth        = 90
	AppHeight       = 30
	AppHalfHeight   = 15
)
