package tui

import (
	"fmt"
	"strings"

	"cx/config"
)

// ListView renders the host list
type ListView struct {
	hosts    []config.Host
	cursor   int
	selected int
}

// NewListView creates a new list view
func NewListView(hosts []config.Host) ListView {
	return ListView{
		hosts:    hosts,
		cursor:   0,
		selected: -1,
	}
}

// SetHosts updates the host list
func (l *ListView) SetHosts(hosts []config.Host) {
	l.hosts = hosts
	if l.cursor >= len(hosts) {
		l.cursor = max(0, len(hosts)-1)
	}
}

// CursorUp moves the cursor up
func (l *ListView) CursorUp() {
	if l.cursor > 0 {
		l.cursor--
	}
}

// CursorDown moves the cursor down
func (l *ListView) CursorDown() {
	if l.cursor < len(l.hosts)-1 {
		l.cursor++
	}
}

// SelectedHost returns the currently selected host
func (l *ListView) SelectedHost() *config.Host {
	if l.cursor >= 0 && l.cursor < len(l.hosts) {
		return &l.hosts[l.cursor]
	}
	return nil
}

// View renders the list
func (l *ListView) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🖥  SSH Hosts"))
	b.WriteString("\n\n")

	if len(l.hosts) == 0 {
		b.WriteString(blurredStyle.Render("  No hosts configured.\n"))
		b.WriteString(blurredStyle.Render("  Press 'a' to add a new host.\n"))
	} else {
		for i, host := range l.hosts {
			cursor := "  "
			style := listItemStyle

			if i == l.cursor {
				cursor = "▸ "
				style = selectedItemStyle
			}

			// Format host entry
			alias := hostAliasStyle.Render(host.Alias)
			if i == l.cursor {
				alias = selectedItemStyle.Render(host.Alias)
			}

			details := formatHostDetails(host)

			line := fmt.Sprintf("%s%s %s", cursor, alias, hostDetailsStyle.Render(details))
			b.WriteString(style.Render(line))
			b.WriteString("\n")
		}
	}

	// Help text
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  a: add • e: edit • d: delete • p: push • enter: connect • q: quit"))

	return b.String()
}

// formatHostDetails formats the host details for display
func formatHostDetails(h config.Host) string {
	var parts []string

	if h.User != "" && h.HostName != "" {
		parts = append(parts, fmt.Sprintf("%s@%s", h.User, h.HostName))
	} else if h.HostName != "" {
		parts = append(parts, h.HostName)
	}

	if h.Port != "" && h.Port != "22" {
		parts = append(parts, fmt.Sprintf(":%s", h.Port))
	}

	if len(parts) == 0 {
		return ""
	}

	return "(" + strings.Join(parts, "") + ")"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
