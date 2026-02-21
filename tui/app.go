package tui

import (
	"fmt"
	"os/exec"
	"strings"

	"cx/config"
	"cx/ssh"
	"cx/tmux"

	tea "github.com/charmbracelet/bubbletea"
)

// ViewMode represents the current view
type ViewMode int

const (
	ModeList ViewMode = iota
	ModeAdd
	ModeEdit
	ModeDelete
	ModeConnect
	ModeKeyDeploy
	ModePush
)

// App is the main application model
type App struct {
	mode       ViewMode
	list       ListView
	form       FormView
	push       PushView
	hosts      []config.Host
	err        string
	status     string
	connecting bool
	deleteHost *config.Host
}

// Messages
type hostsLoadedMsg struct{ hosts []config.Host }
type errMsg struct{ err error }
type statusMsg struct{ msg string }
type connectDoneMsg struct{ err error }
type pushDoneMsg struct{ err error }

// NewApp creates a new application
func NewApp() App {
	return App{
		mode: ModeList,
		list: NewListView(nil),
	}
}

// Init initializes the app
func (a App) Init() tea.Cmd {
	return loadHosts
}

// loadHosts loads hosts from config
func loadHosts() tea.Msg {
	hosts, err := config.ParseConfig()
	if err != nil {
		return errMsg{err}
	}
	return hostsLoadedMsg{hosts}
}

// Update handles messages
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global keys
		switch msg.String() {
		case "ctrl+c":
			return a, tea.Quit
		}

		// Mode-specific handling
		switch a.mode {
		case ModeList:
			return a.updateList(msg)
		case ModeAdd, ModeEdit:
			return a.updateForm(msg)
		case ModeDelete:
			return a.updateDelete(msg)
		case ModePush:
			return a.updatePush(msg)
		}

	case hostsLoadedMsg:
		a.hosts = msg.hosts
		a.list.SetHosts(msg.hosts)
		return a, nil

	case errMsg:
		a.err = msg.err.Error()
		return a, nil

	case statusMsg:
		a.status = msg.msg
		return a, nil

	case connectDoneMsg:
		a.connecting = false
		if msg.err != nil {
			a.err = msg.err.Error()
		}
		return a, nil

	case pushDoneMsg:
		if msg.err != nil {
			a.err = msg.err.Error()
		} else {
			a.status = "Push completed successfully"
		}
		a.mode = ModeList
		return a, nil
	}

	return a, nil
}

// updateList handles list view input
func (a App) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		return a, tea.Quit

	case "up", "k":
		a.list.CursorUp()

	case "down", "j":
		a.list.CursorDown()

	case "a":
		a.mode = ModeAdd
		a.form = NewFormView()
		a.err = ""
		a.status = ""

	case "e":
		if host := a.list.SelectedHost(); host != nil {
			a.mode = ModeEdit
			a.form = NewEditFormView(*host)
			a.err = ""
			a.status = ""
		}

	case "d":
		if host := a.list.SelectedHost(); host != nil {
			a.mode = ModeDelete
			a.deleteHost = host
			a.err = ""
			a.status = ""
		}

	case "p":
		if host := a.list.SelectedHost(); host != nil {
			a.mode = ModePush
			a.push = NewPushView(host)
			a.err = ""
			a.status = ""
		}

	case "enter":
		if host := a.list.SelectedHost(); host != nil {
			return a, a.connect(host)
		}
	}

	return a, nil
}

// updateForm handles form input
func (a App) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.mode = ModeList
		a.err = ""
		return a, nil

	case "enter":
		// Check if in key menu
		if a.form.showKeyMenu {
			cmd := a.form.Update(msg)
			return a, cmd
		}

		// Validate and save
		if err := a.form.Validate(); err != nil {
			a.form.SetError(err.Error())
			return a, nil
		}

		host := a.form.ToHost()

		var err error
		if a.mode == ModeAdd {
			err = config.AddHost(host)
		} else {
			err = config.UpdateHost(a.form.originalAlias, host)
		}

		if err != nil {
			a.form.SetError(err.Error())
			return a, nil
		}

		a.mode = ModeList
		a.status = fmt.Sprintf("Host %q saved", host.Alias)
		return a, loadHosts

	default:
		cmd := a.form.Update(msg)
		return a, cmd
	}
}

// updateDelete handles delete confirmation
func (a App) updateDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if a.deleteHost != nil {
			if err := config.DeleteHost(a.deleteHost.Alias); err != nil {
				a.err = err.Error()
			} else {
				a.status = fmt.Sprintf("Host %q deleted", a.deleteHost.Alias)
			}
			a.deleteHost = nil
		}
		a.mode = ModeList
		return a, loadHosts

	case "n", "N", "esc":
		a.mode = ModeList
		a.deleteHost = nil
		return a, nil
	}

	return a, nil
}

// updatePush handles push view input
func (a App) updatePush(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.mode = ModeList
		return a, nil

	case "up", "k":
		a.push.CursorUp()

	case "down", "j":
		a.push.CursorDown()

	case " ":
		a.push.Toggle()

	case "enter":
		if !a.push.HasSelections() {
			a.push.SetError("Select at least one option")
			return a, nil
		}
		return a, a.executePush()
	}

	return a, nil
}

// executePush runs the selected push operations
func (a *App) executePush() tea.Cmd {
	return func() tea.Msg {
		host := a.push.host
		options := a.push.GetSelectedOptions()

		for _, opt := range options {
			var err error
			switch opt {
			case PushDeployKey:
				err = ssh.PushPublicKey(host.Alias, host.IdentityFile)
			case PushSSHConfig:
				err = ssh.PushSSHConfig(host.Alias)
			case PushSSHKeys:
				err = ssh.PushSSHKeys(host.Alias, nil)
			}
			if err != nil {
				return pushDoneMsg{err}
			}
		}

		return pushDoneMsg{nil}
	}
}

// connect initiates SSH connection
func (a *App) connect(host *config.Host) tea.Cmd {
	return tea.ExecProcess(
		exec.Command("bash", "-c", a.buildConnectCommand(host)),
		func(err error) tea.Msg {
			return connectDoneMsg{err}
		},
	)
}

// buildConnectCommand builds the SSH command with tmux
func (a *App) buildConnectCommand(host *config.Host) string {
	var parts []string

	// Build SSH command
	sshCmd := fmt.Sprintf("ssh %s", host.Alias)

	// Build the remote tmux command using host alias as session name
	tmuxCmd := tmux.BuildTmuxCommand(host.Alias)

	// Combine: SSH into host and run tmux
	parts = append(parts, sshCmd)
	parts = append(parts, "-t")  // Force TTY allocation
	parts = append(parts, fmt.Sprintf("'%s'", tmuxCmd))

	return strings.Join(parts, " ")
}

// View renders the app
func (a App) View() string {
	var b strings.Builder

	switch a.mode {
	case ModeList:
		b.WriteString(a.list.View())

	case ModeAdd, ModeEdit:
		b.WriteString(a.form.View())

	case ModeDelete:
		b.WriteString(a.viewDelete())

	case ModePush:
		b.WriteString(a.push.View())
	}

	// Show status/error at bottom
	if a.err != "" {
		b.WriteString("\n\n")
		b.WriteString(errorStyle.Render("  ✗ " + a.err))
	} else if a.status != "" {
		b.WriteString("\n\n")
		b.WriteString(successStyle.Render("  ✓ " + a.status))
	}

	return containerStyle.Render(b.String())
}

// viewDelete renders the delete confirmation
func (a App) viewDelete() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🗑  Delete Host"))
	b.WriteString("\n\n")

	if a.deleteHost != nil {
		b.WriteString(fmt.Sprintf("  Are you sure you want to delete %q?\n\n",
			a.deleteHost.Alias))
		b.WriteString(warningStyle.Render("  This will remove the host from ~/.ssh/config\n"))
		b.WriteString(warningStyle.Render("  (Keys and other files will not be affected)\n"))
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  y: yes, delete • n: no, cancel"))

	return b.String()
}

// Helper functions for external use
func ConnectToHost(host *config.Host) error {
	return ssh.Connect(host.Alias, host.HostName, host.User, host.Port, host.IdentityFile)
}
