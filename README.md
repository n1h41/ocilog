# ocilog

A keyboard-driven terminal UI for exploring and searching Oracle Cloud Infrastructure (OCI) logs.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), `ocilog` lets you browse compartments, log groups, and logs, run OCI Logging Search queries, ingest test events, and copy results to the clipboard without leaving the terminal.

## Features

- **Compartment browser** – navigate OCI compartments and scope the session.
- **Log groups & logs** – list log groups and their logs with status indicators.
- **Log search** – run Logging Search queries across custom time ranges; ranges longer than 14 days are automatically split into consecutive windows.
- **Search history** – recent searches (query + pipeline) are persisted to `~/.config/ocilog/history.json`; re-running the same query and pipeline refreshes the entry instead of duplicating it.
- **Ingestion** – send plain-text log lines to a specific log OCID.
- **Clipboard integration** – copy the focused pane (formatted output or original JSON) or the current query with `ctrl+y` / `ctrl+o`.
- **Fuzzy filtering** – filter long lists with `/` style search in each view.

## Requirements

- Go 1.26.6 or later
- Valid OCI configuration (see [Configuration](#configuration))

## Installation

```bash
git clone https://github.com/n1h41/ocilog.git
cd ocilog
go build -o ocilog ./cmd/ocilog
```

Optionally install to `$GOPATH/bin`:

```bash
go install ./cmd/ocilog
```

## Configuration

`ocilog` uses the standard OCI configuration provider, so it respects `~/.oci/config` and environment variables such as `OCI_CLI_AUTH`, `OCI_CONFIG_FILE`, `OCI_PROFILE`, etc.

Two optional environment variables are specific to this tool:

| Variable            | Description                                                      |
| ------------------- | ---------------------------------------------------------------- |
| `OCI_REGION`        | Override the region from your OCI config.                        |
| `OCI_COMPARTMENT_OCID` | Start the session scoped to this compartment instead of the tenancy. |

Example:

```bash
export OCI_REGION=us-ashburn-1
export OCI_COMPARTMENT_OCID=ocid1.compartment.oc1..example
./ocilog
```

## Usage

Launch the application:

```bash
./ocilog
```

### Keybindings

| Key                                | Action                                              |
| ---------------------------------- | --------------------------------------------------- |
| `1`                                | Open Compartments tab                               |
| `2`                                | Open Log Groups tab                                 |
| `3`                                | Open Logs tab                                       |
| `4` / `s`                          | Open Search tab                                     |
| `enter`                            | Open selected compartment / log group / log         |
| `enter` (in search with pipe)      | Apply pipeline action                               |
| `space`                            | Select / multi-select item                          |
| `tab`                              | Next tab or next input field                        |
| `r`                                | Refresh current tab                                 |
| `esc`                              | Go up one level                                     |
| `ctrl+r`                           | Open search history                                 |
| `ctrl+y`                           | Copy focused pane (formatted/original) to clipboard |
| `ctrl+o`                           | Copy current query to clipboard                     |
| `ctrl+t` / `ctrl+←` / `ctrl+→`     | Switch panes                                        |
| `ctrl+x`                           | Expand/collapse focused output pane                 |
| `pgup` / `pgdn`                    | Scroll page up / down                               |
| `home` / `end`                     | Jump to top / bottom                                |
| `ctrl+c`                     | Quit                                                |

## Project Structure

```
.
├── cmd/ocilog
│   └── main.go              # Application entry point
├── internal
│   ├── config
│   │   └── config.go        # OCI config / provider loading
│   ├── history
│   │   └── history.go       # Persistent search history
│   ├── oci
│   │   ├── client.go        # OCI SDK client wrapper
│   │   ├── compartments.go  # Compartment listing
│   │   ├── ingest.go        # Log ingestion
│   │   ├── logs.go          # Log group / log listing
│   │   └── search.go        # Logging Search with window splitting
│   └── tui
│       ├── model.go         # Root Bubble Tea model
│       ├── compartments/    # Compartment list view
│       ├── loggroups/       # Log group list view
│       ├── logs/            # Log list view
│       ├── search/          # Search input / results view
│       ├── state/           # Shared session state
│       └── theme/           # Lipgloss styles
├── go.mod
└── go.sum
```

## Development

Run the application during development:

```bash
go run ./cmd/ocilog
```

Run tests:

```bash
go test ./...
```

## License

[MIT](LICENSE)
