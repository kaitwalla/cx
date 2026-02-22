package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"cx/config"
)

// Version is set by main to display in the UI
var Version = "dev"

// ListView renders the host list
type ListView struct {
	hosts      []config.Host
	filtered   []config.Host
	cursor     int
	selected   int
	filter     string
	actionMode bool
}

// NewListView creates a new list view
func NewListView(hosts []config.Host) ListView {
	return ListView{
		hosts:      hosts,
		filtered:   hosts,
		cursor:     0,
		selected:   -1,
		filter:     "",
		actionMode: false,
	}
}

// SetHosts updates the host list
func (l *ListView) SetHosts(hosts []config.Host) {
	l.hosts = hosts
	l.applyFilter()
}

// applyFilter filters hosts based on current filter string
func (l *ListView) applyFilter() {
	if l.filter == "" {
		l.filtered = l.hosts
	} else {
		l.filtered = nil
		filterLower := strings.ToLower(l.filter)
		for _, h := range l.hosts {
			// Match against alias, hostname, or user
			if strings.Contains(strings.ToLower(h.Alias), filterLower) ||
				strings.Contains(strings.ToLower(h.HostName), filterLower) ||
				strings.Contains(strings.ToLower(h.User), filterLower) {
				l.filtered = append(l.filtered, h)
			}
		}
	}
	// Reset cursor if out of bounds
	if l.cursor >= len(l.filtered) {
		l.cursor = max(0, len(l.filtered)-1)
	}
}

// SetFilter updates the filter and refilters hosts
func (l *ListView) SetFilter(f string) {
	l.filter = f
	l.applyFilter()
}

// AppendFilter adds a character to the filter
func (l *ListView) AppendFilter(c rune) {
	l.filter += string(c)
	l.applyFilter()
}

// BackspaceFilter removes the last character from filter
func (l *ListView) BackspaceFilter() {
	if len(l.filter) > 0 {
		_, size := utf8.DecodeLastRuneInString(l.filter)
		l.filter = l.filter[:len(l.filter)-size]
		l.applyFilter()
	}
}

// ClearFilter clears the filter
func (l *ListView) ClearFilter() {
	l.filter = ""
	l.applyFilter()
}

// Filter returns the current filter string
func (l *ListView) Filter() string {
	return l.filter
}

// SetActionMode enables or disables action mode
func (l *ListView) SetActionMode(enabled bool) {
	l.actionMode = enabled
}

// InActionMode returns whether action mode is active
func (l *ListView) InActionMode() bool {
	return l.actionMode
}

// CursorUp moves the cursor up
func (l *ListView) CursorUp() {
	if l.cursor > 0 {
		l.cursor--
	}
}

// CursorDown moves the cursor down
func (l *ListView) CursorDown() {
	if l.cursor < len(l.filtered)-1 {
		l.cursor++
	}
}

// SelectedHost returns the currently selected host
func (l *ListView) SelectedHost() *config.Host {
	if l.cursor >= 0 && l.cursor < len(l.filtered) {
		return &l.filtered[l.cursor]
	}
	return nil
}

// View renders the list
func (l *ListView) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf("🖥  SSH Hosts %s", versionStyle.Render("v"+Version))))
	b.WriteString("\n\n")

	// Show filter or action mode indicator
	if l.actionMode {
		b.WriteString(filterStyle.Render("  ; "))
		b.WriteString(actionModeStyle.Render("action mode"))
		b.WriteString("\n\n")
	} else if l.filter != "" {
		b.WriteString(filterStyle.Render(fmt.Sprintf("  / %s", l.filter)))
		b.WriteString(blurredStyle.Render("_"))
		b.WriteString("\n\n")
	}

	if len(l.hosts) == 0 {
		b.WriteString(blurredStyle.Render("  No hosts configured.\n"))
		b.WriteString(blurredStyle.Render("  Press ;a to add a new host.\n"))
	} else if len(l.filtered) == 0 {
		b.WriteString(blurredStyle.Render("  No matches.\n"))
	} else {
		for i, host := range l.filtered {
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
	if l.actionMode {
		b.WriteString(helpStyle.Render("  a: add • e: edit • d: delete • p: push • q: quit • esc: cancel"))
	} else {
		b.WriteString(helpStyle.Render("  type to filter • ;: actions • enter: connect • esc: clear"))
	}

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
