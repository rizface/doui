# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Run Commands

```bash
# Build the application
go build -o doui .

# Run the application (requires Docker daemon running)
./doui

# Install dependencies
go mod download

# Tidy dependencies
go mod tidy

# Development with auto-refresh (using entr)
ls **/*.go | entr -r go run .
```

## Architecture Overview

doui is a Terminal User Interface (TUI) for managing Docker resources built with Go and the Charm libraries (Bubbletea, Bubbles, Lipgloss).

### Core Pattern: Bubbletea Model-View-Update

The application follows the Elm architecture via Bubbletea:
- **Model**: `internal/app/app.go` - Main `App` struct holds all state
- **Update**: Message-based state changes via `Update()` method
- **View**: Renders UI via `View()` method

### Package Structure

- **`internal/app/`**: Main application logic
  - `app.go`: Central Bubbletea model orchestrating all views and state
  - `messages.go`: All message types for the Bubbletea message loop

- **`internal/docker/`**: Docker SDK wrapper
  - `client.go`: Connection management with API version negotiation
  - `containers.go`, `images.go`, `volumes.go`, `networks.go`, `compose.go`: Resource operations
  - `logs.go`, `stats.go`: Streaming channels for real-time data

- **`internal/ui/`**: UI layer
  - `views/`: Full-screen views (containers, images, groups, volumes, compose, networks, logs, stats)
  - `components/`: Reusable components (sidebar, header, footer, modal)
  - `styles/`: Lipgloss styling definitions

- **`internal/models/`**: Data types
  - `view.go`: View type enum and app state
  - Resource models: container, image, group, volume, network, compose

- **`internal/config/`**: Persistent configuration
  - Group management stored at `~/.config/doui/config.json`

### Key Patterns

**View Navigation**: Uses `ViewType` enum with sidebar navigation. Number keys (1-6) jump to views, Tab cycles through them.

**Async Operations**: Docker operations return `tea.Cmd` functions that emit result messages (e.g., `ContainerStartedMsg`, `ImageRemovedMsg`).

**Modal System**: Confirmation and form modals in `components/modal.go` handle destructive operations. Modal state tracked via `pendingDelete` and `pendingDeleteType` fields.

**Auto-Refresh**: 2-second tick refreshes current view data unless filtering is active.

**Tabbed Views**: Groups and Networks views have internal tabs (List/Containers/Available) managed within their view components.

## Configuration

Groups persist to `~/.config/doui/config.json` (fallback: `~/.doui/config.json`, override: `DOUI_CONFIG_PATH` env var).
