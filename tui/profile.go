package tui

import (
	"fmt"
	"strings"

	"cx/config"
)

// ProfileView handles the tmux profile selection screen
type ProfileView struct {
	host         *config.Host
	profiles     []string
	cursor       int
	err          string
	status       string
}

// NewProfileView creates a new profile view for a host
func NewProfileView(host *config.Host) ProfileView {
	var errMsg string

	store, storeErr := config.LoadProfiles()
	profiles := []string{"none"} // First option is "none" (no profile)
	if storeErr != nil {
		errMsg = fmt.Sprintf("Failed to load profiles: %v", storeErr)
	} else if store != nil {
		profiles = append(profiles, store.ListProfiles()...)
	}

	// Find current profile assignment and set cursor
	cursor := 0
	hp, hpErr := config.LoadHostProfiles()
	if hpErr != nil && errMsg == "" {
		errMsg = fmt.Sprintf("Failed to load host profiles: %v", hpErr)
	}
	if hp != nil {
		currentProfile := hp.GetHostProfile(host.Alias)
		for i, p := range profiles {
			if p == currentProfile {
				cursor = i
				break
			}
		}
	}

	return ProfileView{
		host:     host,
		profiles: profiles,
		cursor:   cursor,
		err:      errMsg,
	}
}

// CursorLeft moves cursor left (wraps)
func (p *ProfileView) CursorLeft() {
	if p.cursor > 0 {
		p.cursor--
	} else {
		p.cursor = len(p.profiles) - 1
	}
}

// CursorRight moves cursor right (wraps)
func (p *ProfileView) CursorRight() {
	if p.cursor < len(p.profiles)-1 {
		p.cursor++
	} else {
		p.cursor = 0
	}
}

// SelectedProfile returns the currently selected profile name
// Returns empty string if "none" is selected
func (p *ProfileView) SelectedProfile() string {
	if p.cursor >= 0 && p.cursor < len(p.profiles) {
		profile := p.profiles[p.cursor]
		if profile == "none" {
			return ""
		}
		return profile
	}
	return ""
}

// SetError sets an error message
func (p *ProfileView) SetError(err string) {
	p.err = err
}

// SetStatus sets a status message
func (p *ProfileView) SetStatus(status string) {
	p.status = status
}

// View renders the profile selector view
func (p *ProfileView) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf("🎨 Tmux Profile for %s", p.host.Alias)))
	b.WriteString("\n\n")

	b.WriteString("  Select profile: ")

	// Render profile options horizontally with < > navigation
	b.WriteString("< ")
	for i, profile := range p.profiles {
		if i == p.cursor {
			b.WriteString(selectedItemStyle.Render(profile))
		} else {
			b.WriteString(blurredStyle.Render(profile))
		}
		if i < len(p.profiles)-1 {
			b.WriteString(" | ")
		}
	}
	b.WriteString(" >")
	b.WriteString("\n\n")

	// Show profile preview if not "none"
	if p.cursor > 0 && p.cursor < len(p.profiles) {
		store, _ := config.LoadProfiles()
		if store != nil {
			profile := store.GetProfile(p.profiles[p.cursor])
			if profile != nil {
				b.WriteString(helpStyle.Render("  Preview:\n"))
				b.WriteString(blurredStyle.Render(fmt.Sprintf("    Prefix: %s\n", profile.PrefixKey)))
				b.WriteString(blurredStyle.Render(fmt.Sprintf("    Colors: %s on %s\n", profile.StatusFG, profile.StatusBG)))
				b.WriteString(blurredStyle.Render(fmt.Sprintf("    Position: %s\n", profile.StatusPosition)))
			}
		}
	}

	b.WriteString("\n")

	if p.err != "" {
		b.WriteString(errorStyle.Render("  " + p.err))
		b.WriteString("\n\n")
	}

	if p.status != "" {
		b.WriteString(successStyle.Render("  " + p.status))
		b.WriteString("\n\n")
	}

	b.WriteString(helpStyle.Render("  left/right: select • enter: push profile • esc: cancel"))

	return b.String()
}
