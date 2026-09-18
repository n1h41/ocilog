# Agent Notes

Compact guidance for working in this repo.

## Project

- `fw-oci` is a small Go TUI for browsing/searching Oracle Cloud Infrastructure logs.
- Module: `n1h41/fw-oci`, Go 1.26.6.
- Entry point: `cmd/fw-oci/main.go`.
- No tests, no CI, no Makefile/Taskfile, no lint config.

## Build & Run

```bash
# Build binary
go build -o fw-oci ./cmd/fw-oci

# Run directly
go run ./cmd/fw-oci
```

## Verification

```bash
go vet ./...
go build ./cmd/fw-oci
```

`go test ./...` succeeds trivially because there are no test files yet.

## Runtime Requirements

- Needs a valid OCI configuration (`~/.oci/config` or standard env vars).
- Optional env overrides specific to this app:
  - `OCI_REGION` – override the configured region.
  - `OCI_COMPARTMENT_OCID` – start scoped to a compartment instead of the tenancy.
- It is a terminal UI (Bubble Tea); it needs an interactive terminal.

## Architecture

- `internal/oci` wraps the OCI SDK clients (identity, logging management, logging search, logging ingestion). New OCI operations belong here.
- `internal/tui` contains Bubble Tea views. `model.go` is the root model; tabs are `compartments/`, `loggroups/`, `logs/`, `search/`.
- `internal/config` loads only the two env vars above and builds the OCI provider.
- `internal/history` persists recent searches to `~/.config/fw-oci/history.json`.

## Conventions

- Keep OCI SDK details out of the TUI views; views depend on `internal/oci.Client` and `internal/tui/state.Session`.
- The search API has a 14-day window limit; `internal/oci/search.go` already splits longer ranges automatically.
