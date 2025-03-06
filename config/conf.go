package config

import (
	"time"

	"github.com/charmbracelet/huh"
)

var Theme *huh.Theme = huh.ThemeBase()

var TodayKey = time.Now().Local().Format("2006-01-02")
var DefaultConfirm = true

const LavaRed = "#fc4a1a"
const MintGreen = "#a8ff78"
const ForestGreen = "#2F7336"
const SunflowerYellow = "#f7b733"
const ClockWerkColor = "#E28413"
const TimeLayout = "2006-01-02 15:04:05.999 -07:00"
const DefaultWidth = 90
const DefaultHeight = 30
const DefaultBarHeight = 15
