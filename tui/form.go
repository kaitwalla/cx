package tui

import (
	"fmt"
	"os/user"
	"strings"

	"cx/config"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// FormField represents a form field index
type FormField int

const (
	FieldAlias FormField = iota
	FieldHostName
	FieldUser
	FieldPort
	FieldIdentityFile
	FieldCount
)

// FormView handles the add/edit host form
type FormView struct {
	inputs      []textinput.Model
	focusIndex  int
	isEdit      bool
	originalAlias string
	err         string
	keyOptions  []string
	keyIndex    int
	showKeyMenu bool
}

// NewFormView creates a new form for adding a host
func NewFormView() FormView {
	inputs := make([]textinput.Model, FieldCount)

	// Alias
	inputs[FieldAlias] = textinput.New()
	inputs[FieldAlias].Placeholder = "myserver"
	inputs[FieldAlias].Focus()
	inputs[FieldAlias].CharLimit = 64
	inputs[FieldAlias].Width = 40

	// HostName
	inputs[FieldHostName] = textinput.New()
	inputs[FieldHostName].Placeholder = "192.168.1.100 or example.com"
	inputs[FieldHostName].CharLimit = 256
	inputs[FieldHostName].Width = 40

	// User
	currentUser, _ := user.Current()
	inputs[FieldUser] = textinput.New()
	inputs[FieldUser].Placeholder = currentUser.Username
	inputs[FieldUser].CharLimit = 64
	inputs[FieldUser].Width = 40

	// Port
	inputs[FieldPort] = textinput.New()
	inputs[FieldPort].Placeholder = "22"
	inputs[FieldPort].CharLimit = 5
	inputs[FieldPort].Width = 10

	// IdentityFile
	inputs[FieldIdentityFile] = textinput.New()
	inputs[FieldIdentityFile].Placeholder = "~/.ssh/id_ed25519 (or press Tab to select)"
	inputs[FieldIdentityFile].CharLimit = 256
	inputs[FieldIdentityFile].Width = 40

	// Get available keys
	keys, _ := config.ListKeyFiles()

	return FormView{
		inputs:      inputs,
		focusIndex:  0,
		keyOptions:  keys,
		keyIndex:    0,
		showKeyMenu: false,
	}
}

// NewEditFormView creates a form pre-filled with host data
func NewEditFormView(host config.Host) FormView {
	f := NewFormView()
	f.isEdit = true
	f.originalAlias = host.Alias

	f.inputs[FieldAlias].SetValue(host.Alias)
	f.inputs[FieldHostName].SetValue(host.HostName)
	f.inputs[FieldUser].SetValue(host.User)
	f.inputs[FieldPort].SetValue(host.Port)
	f.inputs[FieldIdentityFile].SetValue(host.IdentityFile)

	return f
}

// Update handles form input
func (f *FormView) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "shift+tab", "down", "up":
			if f.showKeyMenu {
				if msg.String() == "down" || msg.String() == "tab" {
					f.keyIndex = (f.keyIndex + 1) % len(f.keyOptions)
				} else {
					f.keyIndex = (f.keyIndex - 1 + len(f.keyOptions)) % len(f.keyOptions)
				}
				return nil
			}

			// Move focus
			if msg.String() == "tab" || msg.String() == "down" {
				f.focusIndex = (f.focusIndex + 1) % int(FieldCount)
			} else {
				f.focusIndex = (f.focusIndex - 1 + int(FieldCount)) % int(FieldCount)
			}

			// Update focus states
			for i := range f.inputs {
				if i == f.focusIndex {
					f.inputs[i].Focus()
				} else {
					f.inputs[i].Blur()
				}
			}
			return nil

		case "ctrl+k":
			// Toggle key selection menu if on identity file field
			if f.focusIndex == int(FieldIdentityFile) && len(f.keyOptions) > 0 {
				f.showKeyMenu = !f.showKeyMenu
			}
			return nil

		case "enter":
			if f.showKeyMenu && len(f.keyOptions) > 0 {
				f.inputs[FieldIdentityFile].SetValue(f.keyOptions[f.keyIndex])
				f.showKeyMenu = false
				return nil
			}
		}
	}

	// Handle text input
	var cmd tea.Cmd
	f.inputs[f.focusIndex], cmd = f.inputs[f.focusIndex].Update(msg)
	return cmd
}

// View renders the form
func (f *FormView) View() string {
	var b strings.Builder

	title := "Add New Host"
	if f.isEdit {
		title = "Edit Host"
	}
	b.WriteString(titleStyle.Render("📝 " + title))
	b.WriteString("\n\n")

	labels := []string{"Alias", "Hostname/IP", "User", "Port", "Identity File"}

	for i, input := range f.inputs {
		label := labels[i]
		if i == f.focusIndex {
			b.WriteString(focusedStyle.Render(fmt.Sprintf("  %s:", label)))
		} else {
			b.WriteString(blurredStyle.Render(fmt.Sprintf("  %s:", label)))
		}
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("    %s\n", input.View()))

		// Show key menu for identity file field
		if i == int(FieldIdentityFile) && f.showKeyMenu && len(f.keyOptions) > 0 {
			b.WriteString("\n")
			b.WriteString(helpStyle.Render("  Available keys:\n"))
			for j, key := range f.keyOptions {
				cursor := "  "
				if j == f.keyIndex {
					cursor = "▸ "
					b.WriteString(focusedStyle.Render(fmt.Sprintf("    %s%s\n", cursor, key)))
				} else {
					b.WriteString(blurredStyle.Render(fmt.Sprintf("    %s%s\n", cursor, key)))
				}
			}
		}

		b.WriteString("\n")
	}

	if f.err != "" {
		b.WriteString(errorStyle.Render("  ⚠ " + f.err))
		b.WriteString("\n\n")
	}

	// Help
	help := "tab: next field • ctrl+k: select key • enter: save • esc: cancel"
	b.WriteString(helpStyle.Render("  " + help))

	return b.String()
}

// Validate checks if the form is valid
func (f *FormView) Validate() error {
	alias := strings.TrimSpace(f.inputs[FieldAlias].Value())
	hostname := strings.TrimSpace(f.inputs[FieldHostName].Value())

	if alias == "" {
		return fmt.Errorf("alias is required")
	}
	if hostname == "" {
		return fmt.Errorf("hostname is required")
	}

	return nil
}

// ToHost converts form data to a Host struct
func (f *FormView) ToHost() config.Host {
	currentUser, _ := user.Current()

	port := strings.TrimSpace(f.inputs[FieldPort].Value())
	if port == "" {
		port = "22"
	}

	username := strings.TrimSpace(f.inputs[FieldUser].Value())
	if username == "" {
		username = currentUser.Username
	}

	return config.Host{
		Alias:        strings.TrimSpace(f.inputs[FieldAlias].Value()),
		HostName:     strings.TrimSpace(f.inputs[FieldHostName].Value()),
		User:         username,
		Port:         port,
		IdentityFile: strings.TrimSpace(f.inputs[FieldIdentityFile].Value()),
	}
}

// SetError sets an error message
func (f *FormView) SetError(err string) {
	f.err = err
}

// ClearError clears the error message
func (f *FormView) ClearError() {
	f.err = ""
}
