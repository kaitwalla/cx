package tui

import (
	"fmt"
	"strings"

	"cx/config"
)

// PushOption represents what can be pushed to remote
type PushOption int

const (
	PushDeployKey PushOption = iota // Copy public key to authorized_keys
	PushSSHConfig                   // Copy ~/.ssh/config to remote
	PushSSHKeys                     // Copy private keys to remote
	PushOptionCount
)

// PushView handles the push configuration screen
type PushView struct {
	host       *config.Host
	cursor     int
	selected   []bool
	keyOptions []string
	keyIndex   int
	err        string
	status     string
}

// NewPushView creates a new push view for a host
func NewPushView(host *config.Host) PushView {
	keys, _ := config.ListKeyFiles()

	return PushView{
		host:       host,
		cursor:     0,
		selected:   make([]bool, PushOptionCount),
		keyOptions: keys,
		keyIndex:   0,
	}
}

// CursorUp moves cursor up
func (p *PushView) CursorUp() {
	if p.cursor > 0 {
		p.cursor--
	}
}

// CursorDown moves cursor down
func (p *PushView) CursorDown() {
	if p.cursor < int(PushOptionCount)-1 {
		p.cursor++
	}
}

// Toggle toggles the current option
func (p *PushView) Toggle() {
	p.selected[p.cursor] = !p.selected[p.cursor]
}

// SetError sets an error message
func (p *PushView) SetError(err string) {
	p.err = err
}

// SetStatus sets a status message
func (p *PushView) SetStatus(status string) {
	p.status = status
}

// GetSelectedOptions returns which options are selected
func (p *PushView) GetSelectedOptions() []PushOption {
	var opts []PushOption
	for i, selected := range p.selected {
		if selected {
			opts = append(opts, PushOption(i))
		}
	}
	return opts
}

// HasSelections returns true if any options are selected
func (p *PushView) HasSelections() bool {
	for _, selected := range p.selected {
		if selected {
			return true
		}
	}
	return false
}

// View renders the push view
func (p *PushView) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf("📤 Push to %s", p.host.Alias)))
	b.WriteString("\n\n")

	options := []struct {
		name string
		desc string
	}{
		{"Deploy public key", "Add your public key to remote authorized_keys"},
		{"Push SSH config", "Copy ~/.ssh/config to remote server"},
		{"Push SSH keys", "Copy private keys to remote (for jumping)"},
	}

	for i, opt := range options {
		cursor := "  "
		checkbox := "[ ]"

		if i == p.cursor {
			cursor = "▸ "
		}
		if p.selected[i] {
			checkbox = "[✓]"
		}

		style := listItemStyle
		if i == p.cursor {
			style = selectedItemStyle
		}

		line := fmt.Sprintf("%s%s %s", cursor, checkbox, opt.name)
		b.WriteString(style.Render(line))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render(fmt.Sprintf("      %s", opt.desc)))
		b.WriteString("\n\n")
	}

	if p.err != "" {
		b.WriteString(errorStyle.Render("  ✗ " + p.err))
		b.WriteString("\n\n")
	}

	if p.status != "" {
		b.WriteString(successStyle.Render("  ✓ " + p.status))
		b.WriteString("\n\n")
	}

	b.WriteString(helpStyle.Render("  space: toggle • enter: push selected • esc: cancel"))

	return b.String()
}
