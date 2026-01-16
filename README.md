![image.png](image.png)
# doui - Docker Terminal UI

A beautiful and responsive Terminal User Interface (TUI) for managing Docker containers, images, and groups.

## Features

### ✅ Fully Implemented

#### Container Management
- **List Containers**: View all containers with status, image, ports, and network info
- **Start/Stop/Restart**: Full container lifecycle control
- **Delete Containers**: Remove containers with confirmation modal
- **Real-time Refresh**: Auto-updates every 2 seconds
- **Shell Access**: Interactive shell access with `docker exec -it`
- **Real-time Logs**: Stream container logs with follow mode and scroll
- **Stats Monitoring**: Live CPU, memory, network, and disk I/O monitoring

#### Image Management
- **List Images**: View all images with tags, size, and usage info
- **Remove Images**: Delete images with confirmation modal
- **Bulk Selection**: Select multiple images with space bar for batch operations
- **Bulk Delete**: Remove multiple selected images at once
- **Pull Images**: Pull new images with real-time progress display
- **Prune Images**: Remove all dangling (untagged) images
- **Smart Markers**: Visual indicators for `[dangling]` and `[unused]` images
- **Sorted List**: Tagged images first (alphabetically), then dangling (by date)
- **Usage Tracking**: See which containers use each image
- **Size Display**: Human-readable size formatting (MB/GB)

#### Container Groups
- **Create Groups**: Interactive form to create new groups
- **Manage Groups**: List, view, edit, and delete groups
- **Persistent Storage**: Groups saved to `~/.config/doui/config.json`
- **Batch Start/Stop**: Control all containers in a group simultaneously
- **Parallel Execution**: Group operations run concurrently for speed
- **Delete Groups**: Remove groups with confirmation modal

#### User Interface
- **Sidebar Navigation**: Beautiful left sidebar with tab-based navigation
- **Split-Pane Layout**: Sidebar + main content area (like the image.png)
- **Modal Dialogs**: Confirmation dialogs and input forms
- **Color-Coded States**: Running (green), stopped (gray), paused (yellow)
- **Multiple Views**: Containers, Images, Groups, Logs, Stats
- **Keyboard Navigation**: Intuitive keyboard shortcuts
- **Built-in Search**: Filter containers, images, and groups with `/`
- **Context-Aware Help**: Different help text for each view
- **Status Messages**: Real-time feedback for all operations

## Installation

```bash
go install github.com/rizface/doui@latest
```

## Usage

### Global Keybindings

**Tab Navigation (Multiple Ways):**
- `Tab` or `→` - Cycle to next tab (Containers → Images → Groups → Containers...)
- `Shift+Tab` or `←` - Cycle to previous tab (reverse direction)
- `1` - Jump directly to Containers view
- `2` - Jump directly to Images view
- `3` - Jump directly to Groups view

**Other Global Keys:**
- `Esc` - Return to Containers view from any other view
- `Ctrl+C` or `q` - Quit application

### Containers View
- `↑/↓` - Navigate list
- `s` - Start selected container
- `x` - Stop selected container
- `r` - Restart selected container
- `d` - **Delete container** (with confirmation)
- `e` - Enter container shell (interactive)
- `l` - View logs (streaming)
- `t` - View stats (real-time monitoring)
- `/` - Filter/search containers

### Images View
- `↑/↓` - Navigate list
- `Space` - Toggle selection for bulk operations
- `d` - **Remove image(s)** (with confirmation, works on selection or single)
- `p` - **Pull image** (opens form, shows real-time progress)
- `P` - **Prune dangling images** (removes all untagged images)
- `/` - Filter/search images

### Groups View
- `↑/↓` - Navigate list
- `n` - **Create new group** (opens form modal)
- `Enter` - View group details
- `s` - Start all containers in group
- `x` - Stop all containers in group
- `d` - **Delete group** (with confirmation)
- `/` - Filter/search groups

### Logs View
- `↑/↓` - Scroll through logs
- `f` - Toggle follow mode (auto-scroll)
- `g` - Go to top
- `G` - Go to bottom
- `Esc` - Return to Containers view

### Stats View
- `Esc` - Return to Containers view

All views support:
- Arrow keys for navigation
- `/` for filtering (where applicable)
- `q` or `Ctrl+C` to quit

## Configuration

Container groups are stored in:
- Primary: `$HOME/.config/doui/config.json`
- Fallback: `$HOME/.doui/config.json`
- Override: Set `DOUI_CONFIG_PATH` environment variable

## Project Structure

```
doui/
├── main.go                           # Entry point
├── internal/
│   ├── app/                         # Application logic
│   │   ├── app.go                   # Main bubbletea model
│   │   └── messages.go              # Message types
│   ├── ui/                          # UI components
│   │   ├── styles/                  # Lipgloss styles
│   │   ├── components/              # Reusable components
│   │   └── views/                   # Full-screen views
│   ├── docker/                      # Docker SDK wrapper
│   │   ├── client.go                # Client initialization
│   │   ├── containers.go            # Container operations
│   │   ├── images.go                # Image operations
│   │   ├── logs.go                  # Log streaming
│   │   └── stats.go                 # Stats monitoring
│   ├── config/                      # Configuration management
│   │   ├── config.go                # Config utilities
│   │   ├── groups.go                # Group management
│   │   └── loader.go                # File I/O
│   └── models/                      # Data models
│       ├── container.go             # Container types
│       ├── image.go                 # Image types
│       ├── group.go                 # Group types
│       └── view.go                  # View state
└── pkg/utils/                       # Utilities
```

## Current Status

### ✅ Completed Features
**ALL** features from the spec are implemented and fully working:

#### Core Features
- ✅ **Container listing** with real-time auto-refresh
- ✅ **Container operations** (start, stop, restart, delete with confirmation)
- ✅ **Image listing** and removal (with confirmation modal)
- ✅ **Image bulk operations** (select, bulk delete)
- ✅ **Image pull with progress** (real-time progress display)
- ✅ **Image pruning** (remove dangling images)
- ✅ **Container groups** with persistent storage
- ✅ **Group operations** (start/stop all containers in parallel)
- ✅ **Group creation UI** with interactive form modal
- ✅ **Group deletion** with confirmation
- ✅ **Interactive shell access** (`docker exec -it`)
- ✅ **Real-time log streaming** with follow mode and scroll
- ✅ **Live stats monitoring** (CPU, memory, network, disk I/O)

#### UI/UX Features
- ✅ **Sidebar navigation** with tab-based layout (as per image.png)
- ✅ **Split-pane layout** (sidebar + main content)
- ✅ **Modal dialogs** (confirmations and forms)
- ✅ **Multi-view navigation** (5 different views)
- ✅ **Beautiful TUI** with color-coded states
- ✅ **Search/filter** in all list views
- ✅ **Context-aware help** in footer
- ✅ **Status messages** with auto-clear
- ✅ **Error handling** with user-friendly messages
- ✅ **Docker SDK integration** (not CLI wrapper)

### 🎯 Enhancement Summary (Completed)
1. ✅ **Sidebar Tab Layout** - Left sidebar with visual tabs
2. ✅ **Confirmation Modals** - All destructive operations require confirmation
3. ✅ **Group Creation UI** - Interactive form with name & description fields
4. ✅ **Container Deletion** - Remove containers with confirmation
5. ✅ **Image Deletion** - Remove images with confirmation
6. ✅ **Group Deletion** - Delete groups with confirmation
7. ✅ **Modal System** - Reusable modal component for all dialogs

### 🔧 Future Enhancements (Optional)
- Add containers to existing groups via UI
- Image building from Dockerfile
- Export/import group configurations
- Custom themes and color schemes
- Resource limit configuration
- Multi-host Docker support
- Container creation wizard

## Requirements

- Go 1.24.0 or higher
- Docker daemon running locally
- Terminal with color support

## Dependencies

- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling
- [Docker SDK](https://github.com/docker/docker) - Docker API client

## Development

```bash
# Install dependencies
go mod download

# Build
go build -o doui .

# Run
./doui

# Run with auto-refresh on code changes (using entr or similar)
ls **/*.go | entr -r go run .
```

## Contributing

This is a personal project, but contributions are welcome! Feel free to open issues or submit pull requests.

## License

MIT License - feel free to use this project as you wish.
