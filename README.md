# Argus 👁️

A lightweight, efficient, terminal-based log viewer for Linux. Built in Go for minimal memory footprint and maximum performance.

## Features

- **Unified View** — Monitor journalctl and text logs in a single TUI
- **Dynamic Sources** — Add/remove log sources at runtime
- **Syntax Highlighting** — Color-coded log levels and keywords
- **Efficient** — ~5-10MB memory footprint, single static binary
- **Security-First** — Read-only design, proper privilege separation

## Requirements

- Linux (primary target: Arch Linux)
- Go 1.21+ (for building)
- Systemd (for journalctl integration)

## Quick Start

```bash
# Clone the repository
git clone https://github.com/Expert21/argus.git
cd argus

# Build
make build

# Run (basic mode - may not read all logs)
./argus

# Run with privileges (for full log access)
sudo ./argus
```

## Installation (Production)

For production use with proper privilege separation:

```bash
# Build release binary
make release

# Install binary and wrapper (requires sudo)
sudo make install

# Create argus-users group
sudo groupadd argus-users
sudo usermod -aG argus-users $USER

# Add sudoers rule
echo '%argus-users ALL=(ALL) NOPASSWD: /usr/local/bin/argus' | sudo tee /etc/sudoers.d/argus

# Log out and back in, then run:
sudo argus
```

## Keybindings

| Key | Action |
|-----|--------|
| `q` / `Esc` | Quit |
| `Space` | Simulate log event (demo) |
| `/` | Search (coming soon) |
| `a` | Add source (coming soon) |

## Project Structure

```
argus/
├── cmd/argus/         # Entry point
├── internal/
│   ├── config/        # Configuration management
│   ├── ingest/        # Log source ingestors
│   ├── aggregate/     # Event aggregation
│   ├── filter/        # Log filtering
│   ├── format/        # Syntax highlighting
│   └── tui/           # TUI components
├── configs/           # Default configuration
├── scripts/           # Deployment scripts
└── Makefile
```

## Configuration

Copy the default config to your home directory:

```bash
mkdir -p ~/.config/argus
cp configs/default.yaml ~/.config/argus/config.yaml
```

See [configs/default.yaml](configs/default.yaml) for all options.

## Security Model

Argus uses a privilege separation model:

1. **Read-Only** — The application never writes to log files
2. **Secure Wrapper** — A root-owned wrapper script is the only sudoable entry point
3. **Group-Based Access** — Only members of `argus-users` can run with privileges
4. **Auditable** — All code paths are auditable; no shell escapes

## Development

```bash
# Format code
make fmt

# Run tests
make test

# Run with race detector
make test-race

# Clean build artifacts
make clean
```

## License

MIT

## Author

Built with 🔐 by Isaiah | [Expert21](https://github.com/Expert21)
