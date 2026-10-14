package warp

import (
	tea "github.com/charmbracelet/bubbletea"
	portalis "github.com/starframe-dev/portalis"
	warp "github.com/starframe-dev/warp"
)

// Panel wraps portalis.Emulator into a warp.Panel
type Panel struct {
	emulator *portalis.Emulator
}

// NewPanel creates a new warp panel backed by the given emulator
func NewPanel(emulator *portalis.Emulator) *Panel {
	return &Panel{emulator: emulator}
}

// Emulator returns the underlying terminal emulator
func (p *Panel) Emulator() *portalis.Emulator {
	return p.emulator
}

// View renders the panel content at the given size
func (p *Panel) View(width, height int) string {
	return p.emulator.View(width, height)
}

// Update forwards messages to the emulator
// It translates warp.ResizeMsg into term.ResizeMsg
func (p *Panel) Update(msg tea.Msg) tea.Cmd {
	if r, ok := msg.(warp.ResizeMsg); ok {
		msg = portalis.ResizeMsg{Width: r.Width, Height: r.Height}
	}
	return p.emulator.Update(msg)
}

var _ warp.Panel = (*Panel)(nil)
