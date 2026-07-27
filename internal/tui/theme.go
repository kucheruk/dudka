package tui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func init() {
	// Force truecolor DESIGN tokens even when Terminal.app reports a light
	// background or a weak color profile (otherwise charcoal styles wash out).
	if os.Getenv("DUDKA_COLOR_PROFILE") == "ascii" {
		lipgloss.SetColorProfile(termenv.Ascii)
		return
	}
	lipgloss.SetColorProfile(termenv.TrueColor)
}

// DESIGN.md charcoal / silkscreen / LED tokens for truecolor terminals (P046).
var (
	colorPanel      = lipgloss.Color("#1A1A1A")
	colorPanelDeep  = lipgloss.Color("#0E0E0E")
	colorSilk       = lipgloss.Color("#F2F2F2")
	colorSilkDim    = lipgloss.Color("#8A8A8A")
	colorLED        = lipgloss.Color("#FF4500")
	colorSegment    = lipgloss.Color("#FF3B30")
	colorStepRed    = lipgloss.Color("#FF3B30")
	colorStepOrange = lipgloss.Color("#FF9A00")
	colorStepYellow = lipgloss.Color("#FFD600")
	colorOK         = lipgloss.Color("#FFD600")
	colorDanger     = lipgloss.Color("#FF3B30")
)

func styleStatus() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(colorSegment).
		Background(colorPanel).
		Bold(true).
		Padding(0, 1)
}

func styleLabel() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(colorSilkDim).
		Background(colorPanelDeep).
		Bold(true)
}

func styleBody() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(colorSilk).
		Background(colorPanelDeep)
}

func styleDim() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(colorSilkDim).
		Background(colorPanelDeep)
}

func stylePeerActive() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(colorLED).
		Background(colorPanelDeep).
		Bold(true)
}

func styleCompose() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(colorSilk).
		Background(colorPanelDeep).
		Padding(0, 1)
}

func styleAction() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(colorPanelDeep).
		Background(colorLED).
		Bold(true).
		Padding(0, 1)
}

func styleErr() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(colorDanger).
		Background(colorPanel)
}

func styleCanvas(width, height int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Background(colorPanel).
		Foreground(colorSilk)
}

// StepPads returns a 4-pad progress strip for percent 0..100 (DESIGN step-row).
func StepPads(percent int) string {
	lit := percent / 25
	if percent > 0 && lit == 0 {
		lit = 1
	}
	if lit > 4 {
		lit = 4
	}
	if lit < 0 {
		lit = 0
	}
	colors := []lipgloss.Color{colorStepRed, colorStepOrange, colorStepYellow, colorSilk}
	var b stringsBuilder
	for i := 0; i < 4; i++ {
		st := lipgloss.NewStyle().Background(colorPanelDeep)
		if i < lit {
			st = st.Foreground(colors[i]).Background(colors[i])
			b.WriteString(st.Render("▀"))
		} else {
			st = st.Foreground(colorSilkDim)
			b.WriteString(st.Render("·"))
		}
	}
	return b.String()
}

type stringsBuilder struct{ b []byte }

func (s *stringsBuilder) WriteString(v string) { s.b = append(s.b, v...) }
func (s *stringsBuilder) String() string       { return string(s.b) }
