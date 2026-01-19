# Argus 👁️

A lightweight, efficient, terminal-based log viewer for Linux. Built in Go for minimal memory footprint and maximum performance.

## Features

- **Unified View** — Monitor journalctl and text logs in a single TUI
- **Live Streaming** — Real-time log updates with auto-scroll
- **Source Filtering** — Filter by log source (journal, files)
- **Syntax Highlighting** — Color-coded log levels and keywords
- **Efficient** — ~3.5MB binary, minimal memory footprint
- **Security-First** — Read-only design, proper privilege separation

## Quick Start

```bash
# Clone and build
git clone https://github.com/Expert21/argus.git
cd argus
make build

# Run (may need sudo for full log access)
./argus
sudo ./argus
```

## Installation

### Quick Install (Personal Use)
```bash
make install-quick
```

### Full Install (Production, with Security Features)
```bash
sudo make install
sudo usermod -aG argus-users $USER
# Log out and back in
sudo argus  # No password needed!
```

### Uninstall
```bash
sudo make uninstall
```

## Keybindings

| Key | Action |
|-----|--------|
| `q` / `Ctrl+C` | Quit |
| `Tab` | Switch focus (sidebar ↔ logs) |
| `j/k` or `↑/↓` | Navigate / Scroll |
| `Enter` | Select source filter |
| `p` / `Space` | Pause / Resume |
| `c` | Clear log view |
| `g` / `G` | Go to top / bottom |
| `r` | Reload config |
| `?` | Show help |

## Configuration

Config file location: `~/.config/argus/config.yaml`

```bash
# Install user config
make install-config
```

Example config:
```yaml
general:
  max_buffer: 10000
  timestamp_format: "2006-01-02 15:04:05"

sources:
  - name: "System Journal"
    type: journald
    enabled: true
    
  - name: "Auth Log"
    type: file
    path: "/var/log/auth.log"
    enabled: true
```

## Project Structure

```
argus/
├── cmd/argus/         # Entry point
├── internal/
│   ├── aggregate/     # Event aggregation, ring buffer
│   ├── config/        # Configuration loading
│   ├── ingest/        # Log source ingestors
│   └── tui/           # TUI components (Bubbletea/Lipgloss)
├── configs/           # Default configuration
├── scripts/           # Installation scripts
└── Makefile
```

## Security Model

Argus uses a privilege separation model for security:

1. **Read-Only** — Never writes to log files or executes commands
2. **Secure Wrapper** — Root-owned script validates binary integrity
3. **Group-Based Access** — `argus-users` group for passwordless sudo
4. **Minimal Privileges** — Requests only read access to logs

### Security Files
- `/usr/local/bin/argus` — Wrapper script (root-owned)
- `/usr/local/bin/argus-bin` — Actual binary (root-owned)
- `/etc/sudoers.d/argus` — Sudoers rule

## Development

```bash
make build       # Development build
make release     # Optimized release build
make test        # Run tests
make fmt         # Format code
make clean       # Clean artifacts
make help        # Show all targets
```

## License

MIT

## Author

Built with 🔐 by Isaiah | [Expert21](https://github.com/Expert21)
