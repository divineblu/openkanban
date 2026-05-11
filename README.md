<h1 align="center">
  <br>
  <img src="https://github.com/user-attachments/assets/14cde506-2091-4745-9349-2604d8ec5b32" alt="OpenKanban" width="600">
  <br>
</h1>

<h4 align="center">A TUI kanban board for orchestrating AI coding agents.</h4>

<p align="center">
  <a href="https://github.com/divineblu/openkanban/releases/latest">
    <img src="https://img.shields.io/github/v/release/divineblu/openkanban?style=flat-square&color=blue" alt="Release">
  </a>
  <a href="https://github.com/divineblu/openkanban/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/divineblu/openkanban?style=flat-square&color=green" alt="License">
  </a>
  <a href="https://github.com/divineblu/openkanban">
    <img src="https://img.shields.io/github/go-mod/go-version/divineblu/openkanban?style=flat-square" alt="Go Version">
  </a>
  <a href="https://github.com/divineblu/openkanban/actions">
    <img src="https://img.shields.io/github/actions/workflow/status/divineblu/openkanban/release.yaml?style=flat-square&label=build" alt="Build Status">
  </a>
</p>

<p align="center">
  <img src="./docs/assets/demo.gif" alt="OpenKanban Demo" width="800">
</p>

---

## Why?

AI coding agents are powerful, but managing multiple agents across projects gets messy fast. You end up with terminals everywhere, losing track of what's running where, and context-switching between tasks becomes a chore.

OpenKanban gives you a single view of all your work. Each ticket gets its own git worktree and embedded terminal. Spawn an agent, watch it work, jump between tasks. Everything stays organized.

## Features

- **Tickets as worktrees** - Each task gets an isolated git branch
- **Embedded terminals** - Agents run inside the TUI, not in random terminal tabs
- **Any agent** - OpenCode, Claude Code, Gemini, Codex, Aider, or whatever CLI tool you prefer
- **Multi-project** - Manage tickets across all your repositories from one board

## Install

### Homebrew (macOS/Linux)

```bash
brew install --cask divineblu/tap/openkanban
```

To update:

```bash
brew upgrade --cask openkanban
```

### Go

```bash
go install github.com/divineblu/openkanban@latest
```

## Quick Start

```bash
cd ~/projects/my-app
openkanban new "My App"
openkanban
```

## Keybindings

| Key | Action |
|-----|--------|
| `j/k` | Navigate tickets up/down |
| `h/l` | Navigate between columns |
| `space` | Move ticket to next column |
| `n` | New ticket |
| `s` | Spawn agent |
| `enter` | Attach to agent |
| `?` | Full help |

## Configuration

OpenKanban is highly configurable. Agents, keybindings, branch naming, cleanup behavior - all customizable in `~/.config/openkanban/config.json`.

See [Configuration Guide](./docs/CONFIGURATION.md) for details.

## License

[AGPL-3.0](LICENSE)

This repository is based on [TechDufus/openkanban](https://github.com/TechDufus/openkanban).
It was modified in May 2026 to use the `github.com/divineblu/openkanban`
repository and Go module path.
