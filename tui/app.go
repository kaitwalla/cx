package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	ModeTmuxProfile
)

// App is the main application model
type App struct {
	mode       ViewMode
	list       ListView
	form       FormView
	push       PushView
	profile    ProfileView
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
type pushChainMsg struct {
	options []PushOption
	nextIdx int
}
type profilePushDoneMsg struct{ err error }

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
	// Sort by last used (most recent first)
	hosts = config.SortByLastUsed(hosts)
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
		case ModeTmuxProfile:
			return a.updateTmuxProfile(msg)
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

	case pushChainMsg:
		// Continue with the next push operation in the chain
		return a, a.runPushOperation(msg.options, msg.nextIdx)

	case profilePushDoneMsg:
		if msg.err != nil {
			a.err = msg.err.Error()
		} else {
			a.status = "Tmux profile pushed successfully"
		}
		a.mode = ModeList
		return a, nil
	}

	return a, nil
}

// updateList handles list view input
func (a App) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle action mode
	if a.list.InActionMode() {
		a.list.SetActionMode(false)
		switch key {
		case "q":
			return a, tea.Quit
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
		case "t":
			if host := a.list.SelectedHost(); host != nil {
				a.mode = ModeTmuxProfile
				a.profile = NewProfileView(host)
				a.err = ""
				a.status = ""
			}
		}
		// Any other key just exits action mode
		return a, nil
	}

	// Normal mode
	switch key {
	case "esc":
		// Clear filter, or quit if no filter
		if a.list.Filter() != "" {
			a.list.ClearFilter()
		} else {
			return a, tea.Quit
		}

	case "up", "k":
		a.list.CursorUp()

	case "down", "j":
		a.list.CursorDown()

	case ";":
		a.list.SetActionMode(true)

	case "backspace":
		a.list.BackspaceFilter()

	case "enter":
		if host := a.list.SelectedHost(); host != nil {
			a.list.ClearFilter()
			return a, a.connect(host)
		}

	default:
		// Type to filter - only single printable characters
		if len(key) == 1 {
			r := rune(key[0])
			if r >= 32 && r < 127 {
				a.list.AppendFilter(r)
			}
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

		// Save the profile assignment
		profileName := a.form.SelectedProfile()
		hp, _ := config.LoadHostProfiles()
		if hp == nil {
			hp = &config.HostProfiles{Assignments: make(map[string]string)}
		}
		hp.SetHostProfile(host.Alias, profileName)

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

// updateTmuxProfile handles tmux profile view input
func (a App) updateTmuxProfile(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.mode = ModeList
		return a, nil

	case "left", "h":
		a.profile.CursorLeft()

	case "right", "l":
		a.profile.CursorRight()

	case "enter":
		return a, a.executeTmuxProfilePush()
	}

	return a, nil
}

// executePush runs the selected push operations
// Uses tea.ExecProcess to properly release the terminal for interactive password prompts
func (a *App) executePush() tea.Cmd {
	options := a.push.GetSelectedOptions()

	if len(options) == 0 {
		return func() tea.Msg { return pushDoneMsg{nil} }
	}

	// Run operations sequentially, starting with the first
	return a.runPushOperation(options, 0)
}

// runPushOperation executes a single push operation and chains to the next
func (a *App) runPushOperation(options []PushOption, idx int) tea.Cmd {
	if idx >= len(options) {
		return func() tea.Msg { return pushDoneMsg{nil} }
	}

	host := a.push.host
	var cmd *exec.Cmd

	switch options[idx] {
	case PushDeployKey:
		keyPath := host.IdentityFile
		if keyPath == "" {
			home, _ := os.UserHomeDir()
			keyPath = filepath.Join(home, ".ssh", "id_ed25519")
		}
		// Use exec.Command with separate args to avoid shell injection
		cmd = exec.Command("ssh-copy-id", "-i", keyPath, host.Alias)

	case PushSSHConfig:
		home, _ := os.UserHomeDir()
		configPath := filepath.Join(home, ".ssh", "config")
		// scp with explicit arguments, no shell interpolation
		cmd = exec.Command("scp", configPath, host.Alias+":~/.ssh/config")

	case PushSSHKeys:
		// Use the existing ssh package function which handles file enumeration
		// safely without shell glob expansion. This uses key auth (no interactive prompt).
		if err := ssh.PushSSHKeys(host.Alias, nil); err != nil {
			return func() tea.Msg { return pushDoneMsg{err} }
		}
		// Chain to next operation
		return a.runPushOperation(options, idx+1)
	}

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return pushDoneMsg{err}
		}
		// Chain to next operation
		return pushChainMsg{options: options, nextIdx: idx + 1}
	})
}

// executeTmuxProfilePush pushes the selected tmux profile to the host
func (a *App) executeTmuxProfilePush() tea.Cmd {
	host := a.profile.host
	profileName := a.profile.SelectedProfile()

	// Save the host-profile association
	hp, _ := config.LoadHostProfiles()
	if hp == nil {
		hp = &config.HostProfiles{Assignments: make(map[string]string)}
	}
	hp.SetHostProfile(host.Alias, profileName)

	// If "none" selected, just save the assignment without pushing
	if profileName == "" {
		return func() tea.Msg {
			return profilePushDoneMsg{nil}
		}
	}

	// Get the profile and generate config
	store, err := config.LoadProfiles()
	if err != nil {
		return func() tea.Msg {
			return profilePushDoneMsg{err}
		}
	}

	profile := store.GetProfile(profileName)
	if profile == nil {
		return func() tea.Msg {
			return profilePushDoneMsg{fmt.Errorf("profile %q not found", profileName)}
		}
	}

	configContent := profile.GenerateConfig()

	// Push to remote
	return func() tea.Msg {
		err := ssh.PushTmuxProfile(host.Alias, configContent)
		return profilePushDoneMsg{err}
	}
}

// connect initiates SSH connection
func (a *App) connect(host *config.Host) tea.Cmd {
	// Record usage for sorting
	config.RecordUsage(host.Alias)

	return tea.ExecProcess(
		exec.Command("bash", "-c", a.buildConnectCommand(host)),
		func(err error) tea.Msg {
			return connectDoneMsg{err}
		},
	)
}

// buildConnectCommand builds the SSH command with tmux
func (a *App) buildConnectCommand(host *config.Host) string {
	// Escape single quotes for safe shell interpolation
	escapeShell := func(s string) string {
		return strings.ReplaceAll(s, "'", "'\\''")
	}

	// Check if we're in iTerm2 for control mode
	useControlMode := tmux.IsITerm()

	// Build the remote tmux command using host alias as session name
	tmuxCmd := tmux.BuildTmuxCommandWithOptions(host.Alias, useControlMode)

	// Wrap with ensure-tmux logic (checks for tmux, installs if missing)
	ensureCmd := tmux.BuildEnsureTmuxCommand(tmuxCmd)

	// Escape for shell execution
	escapedHost := escapeShell(host.Alias)
	escapedCmd := escapeShell(ensureCmd)

	// Clear screen first, then run SSH with tmux command
	return fmt.Sprintf("clear && ssh '%s' -t '%s'", escapedHost, escapedCmd)
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

	case ModeTmuxProfile:
		b.WriteString(a.profile.View())
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
