# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Project Overview

GitGoblin is a lightweight, terminal-based Git dashboard built with Go and Bubble Tea. It provides a real-time visual overview of repository status with auto-refreshing every 2 seconds. The application is read-only and non-intrusive, designed to run in a split terminal pane while developers work.

## Common Commands

### Building
```bash
# Build the binary (outputs to ./goblin)
go build -o goblin

# Build and install to $GOPATH/bin (installs as 'gitgoblin')
go install

# Run from source
go run main.go
```

### Running
```bash
# Run the application in any Git repository
./goblin

# Or if installed via go install
gitgoblin
```

### Testing
The project currently has no test files. When adding tests, use standard Go testing:
```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./internal/git/

# Run with verbose output
go test -v ./...
```

## Architecture

### Elm Architecture Pattern
GitGoblin follows the Elm architecture pattern via Bubble Tea, consisting of:
- **Model**: Application state (`internal/ui/app.go`, `internal/ui/dashboard.go`)
- **Update**: State updates based on messages (`Update` methods)
- **View**: Rendering logic (`View` methods)
- **Commands**: Side effects that return messages (`Init`, `loadData`, `tickCmd`)

### Package Structure
```
internal/
├── git/        # Git operations - all git commands are executed here
│   ├── branch.go   # Branch operations (list, switch, create, delete, default detection)
│   ├── log.go      # Commit history and current branch retrieval
│   └── status.go   # Working tree status, staging, diff, line stats
├── models/     # Data structures
│   ├── branch.go   # Branch model with upstream tracking
│   ├── commit.go   # Commit metadata
│   └── file.go     # File change status (git porcelain format)
└── ui/         # Terminal UI components
    ├── app.go          # Root model with tick-based auto-refresh
    └── dashboard.go    # Main dashboard view with metrics
```

### Key Design Patterns

**Message Passing**: All state updates flow through typed messages:
- `tickMsg` triggers auto-refresh every 2 seconds
- `dashboardDataMsg` carries all git data from async load
- `tea.KeyMsg` for keyboard input (only Ctrl+C handled)
- `tea.WindowSizeMsg` for responsive layout

**Git Integration**: The `internal/git` package wraps `os/exec` to run git commands. All functions parse git output and return Go structs. Common patterns:
- Use `git status --porcelain` for parseable output
- Use `git branch -vv` to get upstream tracking info
- Use `git diff --numstat` for line statistics
- Branch comparison uses `git rev-list --left-right --count`

**Default Branch Detection**: Multi-strategy approach in `GetDefaultBranch()`:
1. Try `git symbolic-ref refs/remotes/origin/HEAD --short` (fastest)
2. Try `git remote show origin` and parse "HEAD branch"
3. Fallback to checking common names: main, master, dev, develop

**Styling with Lipgloss**: All UI rendering uses Lipgloss styles with consistent color scheme:
- Cyan (color "cyan"): Labels and branch names
- Green (color "34"): Additions, new files, commits ahead
- Red (color "196"): Deletions, deleted files
- Gray (color "240"): Zero values, borders, dividers
- Orange (color "214"): Behind warnings
- White: Modified files

## Development Guidelines

### Adding New Git Operations
1. Add function to appropriate file in `internal/git/`
2. Execute git command using `exec.Command("git", args...)`
3. Parse output into model structs from `internal/models/`
4. Handle errors gracefully - dashboard should degrade, not crash
5. Return Go types, not raw strings

### Adding New Dashboard Metrics
1. Add field to `DashboardView` struct in `dashboard.go`
2. Add field to `dashboardDataMsg` struct
3. Fetch data in `loadData()` command
4. Update field in `Update()` message handler
5. Render in `View()` or `renderStatusBox()`

### Modifying the Refresh Cycle
- Auto-refresh interval is controlled by `tickCmd()` in `app.go`
- Currently set to 2 seconds: `tea.Tick(time.Second*2, ...)`
- All git operations are loaded in `dashboard.loadData()` which runs on every tick

### Go Version Requirement
- The project requires Go 1.21+ according to README
- The `go.mod` specifies `go 1.25.4` (verify this is correct or update)

## Important Considerations

### Read-Only Design
GitGoblin is intentionally read-only for safety. It has no commands to:
- Make commits
- Switch branches
- Stage/unstage files (though functions exist in `status.go`, they're unused)

If adding interactive features, carefully consider the safety implications and user expectations.

### Terminal Compatibility
- Uses `tea.WithAltScreen()` for full-screen TUI
- Only handles Ctrl+C for quitting (minimal keybindings)
- Requires terminal with color and Unicode support (for emojis)

### Git Repository Check
The application checks for `.git` directory on startup via `isGitRepo()` in `cmd/root.go`. This is a simple check; it won't detect bare repositories or worktrees.

### Performance
- Auto-refresh every 2 seconds spawns multiple git subprocesses
- All git operations are synchronous within `loadData()`
- For large repositories, consider caching or throttling expensive operations
