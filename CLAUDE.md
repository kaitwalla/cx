# cx - SSH Host Manager

A terminal UI application for managing SSH hosts and connections with automatic tmux session handling.

## Overview

cx provides an interactive terminal interface for:
- Managing SSH host configurations (stored in `~/.ssh/config`)
- Connecting to hosts with automatic tmux session creation
- Deploying SSH keys and configs to remote hosts
- Self-updating from GitHub releases

## Architecture

```
main.go          Entry point, CLI argument handling
├── tui/         Terminal UI (Bubble Tea framework)
│   ├── app.go       Main application state machine
│   ├── list.go      Host list view
│   ├── form.go      Add/edit host form
│   ├── push.go      Push options view
│   └── styles.go    Lip Gloss styling
├── config/      SSH config management
│   ├── parser.go    Parse ~/.ssh/config
│   ├── writer.go    Write host entries
│   └── history.go   Usage tracking for sorting
├── ssh/         SSH operations
│   ├── connect.go   SSH connection handling
│   ├── copyid.go    ssh-copy-id functionality
│   ├── keygen.go    Key generation
│   └── push.go      Push configs/keys to remote
├── tmux/        Tmux integration
│   ├── detect.go    OS/distro detection
│   ├── install.go   Remote tmux installation
│   └── session.go   Session management
└── update/      Self-update from GitHub
    └── update.go    Download and verify releases
```

## CLI Commands

```
cx              Launch interactive TUI
cx update       Self-update to latest release
cx version      Show version
cx help         Show help
```

## TUI Key Bindings

### List View (main screen)
- `j`/`k` or `up`/`down` - Navigate hosts
- `Enter` - Connect to selected host (with tmux)
- `a` - Add new host
- `e` - Edit selected host
- `d` - Delete selected host
- `p` - Push menu (deploy keys/config to remote)
- `q`/`Esc` - Quit

### Form View (add/edit)
- `Tab`/`Shift+Tab` - Navigate fields
- `Enter` - Save host
- `Esc` - Cancel

### Delete Confirmation
- `y` - Confirm delete
- `n`/`Esc` - Cancel

### Push View
- `j`/`k` - Navigate options
- `Space` - Toggle option
- `Enter` - Execute selected pushes
- `Esc` - Cancel

## Push Operations

1. **Deploy Key** - Copies public key to remote `~/.ssh/authorized_keys`
2. **Push SSH Config** - Copies local `~/.ssh/config` to remote
3. **Push SSH Keys** - Copies all key pairs to remote `~/.ssh/`

## Connection Flow

1. User selects host and presses Enter
2. Usage recorded for sorting (most recently used first)
3. SSH connection initiated with tmux command:
   - If tmux exists on remote: attaches to or creates session named after host alias
   - If tmux missing: auto-installs (Debian/RHEL/Arch/macOS) then creates session

## Config Storage

- Hosts stored in `~/.ssh/config` (standard SSH config format)
- Usage history stored in `~/.config/cx/history.json`

## Building

```bash
# Development build
go build -o cx .

# Release build with version
go build -ldflags "-X main.version=1.0.0" -o cx .

# Cross-compile (CI builds these)
GOOS=darwin GOARCH=arm64 go build -o cx-darwin-arm64 .
GOOS=darwin GOARCH=amd64 go build -o cx-darwin-amd64 .
GOOS=linux GOARCH=amd64 go build -o cx-linux-amd64 .
GOOS=linux GOARCH=arm64 go build -o cx-linux-arm64 .
```

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Styling

## Update Mechanism

`cx update` downloads from GitHub releases at `kaitwalla/cx`:
1. Fetches release metadata from GitHub API
2. Downloads platform-specific binary (`cx-{os}-{arch}`)
3. Downloads and verifies SHA256 checksum
4. Replaces current executable

## Key Files for Common Tasks

| Task | Files |
|------|-------|
| Add TUI feature | `tui/app.go`, add ViewMode, update handlers |
| New SSH operation | `ssh/` package |
| Change host storage | `config/parser.go`, `config/writer.go` |
| Modify connection behavior | `tui/app.go:buildConnectCommand()` |
| Update tmux handling | `tmux/session.go`, `tmux/install.go` |
