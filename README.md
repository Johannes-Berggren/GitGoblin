# GitGoblin 🧙

A beautiful, lightweight terminal-based Git dashboard built with Go and Bubble Tea. GitGoblin provides a real-time visual overview of your repository status, helping you stay on top of your development workflow without leaving the terminal.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)

## ✨ Features

- 📊 **Real-time Dashboard** - Auto-refreshing view of your repository status (every 2 seconds)
- 📁 **File Change Tracking** - See all uncommitted changes with color-coded status indicators
- 🎯 **Default Branch Comparison** - Know exactly how many commits you're ahead/behind the default branch (main/dev)
- 📈 **Development Metrics** - Track files changed, time since last commit, commits ahead, and line changes
- 🌿 **Branch Visualization** - See your current branch with visual indicators
- 🎨 **Color-Coded File Status** - Green for new files, red for deleted, white for modified
- 📊 **Per-File Line Stats** - See +additions/-deletions for each file
- ⚡ **Fast & Lightweight** - Built in Go, minimal resource usage
- 🚀 **Read-Only & Safe** - Non-intrusive monitoring, no accidental changes

## 📸 What It Looks Like

GitGoblin displays a clean dashboard with:
- Branch name in a bordered box
- Warning if behind origin
- Status box showing:
  - Files changed count
  - Time since last commit
  - Commits ahead of upstream
  - Total lines added/deleted
  - Comparison to default branch (e.g., "vs main: ↑5 ↓2")
- List of all uncommitted files with per-file line statistics

## 🚀 Installation

### Homebrew (macOS/Linux)

```bash
brew install Johannes-Berggren/tap/gitgoblin
```

### Download Binary

Download the latest release from [GitHub Releases](https://github.com/johannesberggren/gitgoblin/releases).

### Using `go install`

```bash
go install github.com/johannesberggren/gitgoblin@latest
```

The binary will be installed as `gitgoblin` in your `$GOPATH/bin` directory.

### From Source

```bash
git clone https://github.com/johannesberggren/gitgoblin.git
cd gitgoblin
go build -o goblin
```

Then move the binary to your PATH or run it directly with `./goblin`.

## 🎯 Usage

Navigate to any Git repository and run:

```bash
goblin
```

The dashboard will appear and automatically refresh every 2 seconds, showing:
- Current branch and its status
- All uncommitted file changes
- Development metrics
- Comparison to the default branch

### Keyboard Shortcuts

- `Ctrl+C` - Quit GitGoblin

That's it! GitGoblin is designed to be a passive, glanceable dashboard that runs in a split terminal pane while you code.

## 📋 Requirements

- **Go 1.21 or higher** (for building from source)
- **Git** installed and accessible in your PATH
- A terminal that supports color and Unicode characters

## 🛠️ Building

```bash
# Clone the repository
git clone https://github.com/johannesberggren/gitgoblin.git
cd gitgoblin

# Build the binary
go build -o goblin

# Optional: Install to $GOPATH/bin
go install
```

## 💡 Use Cases

- **Development Dashboard** - Run in a tmux/screen pane while coding
- **PR Preparation** - Quickly see if you need to rebase before creating a pull request
- **Repository Monitoring** - Glance at current state without running multiple git commands
- **Clean Workspace Verification** - Instantly see if you have uncommitted changes

## 🎨 Color Coding

GitGoblin uses intuitive color coding:

- **Green**: New/added files, lines added, commits ahead
- **Red**: Deleted files, lines deleted
- **Orange/Yellow**: Behind origin warnings
- **White**: Modified files
- **Gray**: Zero values (e.g., +0/-0) to make actual changes stand out
- **Cyan**: Labels and branch names

## 🏗️ Architecture

GitGoblin is built with:

- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)** - Terminal UI framework (Elm architecture)
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)** - Styling and layout
- **[Cobra](https://github.com/spf13/cobra)** - CLI framework

The codebase is organized into:
- `internal/git/` - Git operations (status, branches, log, etc.)
- `internal/ui/` - TUI components and views
- `internal/models/` - Data structures
- `cmd/` - CLI entry point

## 🤝 Contributing

Contributions are welcome! Whether it's:

- 🐛 Bug reports
- 💡 Feature requests
- 📖 Documentation improvements
- 🔧 Code contributions

Please feel free to open an issue or submit a pull request.

### Development Setup

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/gitgoblin.git`
3. Create a feature branch: `git checkout -b feature/amazing-feature`
4. Make your changes and commit: `git commit -m "Add amazing feature"`
5. Push to your fork: `git push origin feature/amazing-feature`
6. Open a pull request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

Built with these amazing open source libraries:

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) by Charm - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) by Charm - Terminal styling
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components

Special thanks to the Charm community for building incredible terminal tools!

## 🔗 Links

- [Issues](https://github.com/johannesberggren/gitgoblin/issues)
- [Pull Requests](https://github.com/johannesberggren/gitgoblin/pulls)

---

Made with ❤️ and Go
