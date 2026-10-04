---
parent: high-level-design
prefix: BUILD
---

# Build and Installation

## Context and Design Philosophy

A `Makefile` at the repository root is the documented way to build, run, test, and install c64sh. Each target is a short, memorable command that bundles the steps a user would otherwise type by hand, and installs to a directory the user owns, so no `sudo` is needed.

## Module

The Go module path is `github.com/bryanesmith/c64sh`, declared in `go.mod` at the repository root, so packages import as `github.com/bryanesmith/c64sh/internal/lexer` and so on.

## Variables

| Variable | Default | Purpose |
|---|---|---|
| `GO` | `go` | Go command, overridable to use a specific toolchain. |
| `BIN` | `bin/c64sh` | Build output path, relative to the repository root. |
| `INSTALL_DIR` | `$(HOME)/bin` | Directory `make install` copies the binary into. |
| `ARGS` | empty | Arguments passed to `c64sh` by `make run`. |

All are set with `?=` so they can be overridden on the command line (`make install INSTALL_DIR=/usr/local/bin`) or from the environment.

## Targets

| Target | Depends on | Effect |
|---|---|---|
| `build` (default) | — | `$(GO) build -o $(BIN) ./cmd/c64sh`. |
| `run` | `build` | Runs `$(BIN) $(ARGS)` with the terminal's stdin, stdout, and stderr. |
| `install` | `build` | Creates `$(INSTALL_DIR)` if missing, then copies `$(BIN)` to `$(INSTALL_DIR)/c64sh` with mode `0755`, replacing any existing file. |
| `test` | — | `$(GO) test ./...`. This includes the snapshot tests, so it fails if an example's output differs from its snapshot. |
| `update-snapshots` | — | `UPDATE_SNAPS=true $(GO) test ./test/snapshot ./test/tutorial`: rewrites the snapshots of `examples/features/` and of the tutorial's programs from current output, creating missing ones and deleting those without an example (see the snapshot design). |
| `clean` | — | Removes the `bin/` directory. |

Recipes run silently (each command prefixed with `@`), so `make run` shows only c64sh's own output, not the command line `make` executed.

`build` always invokes `go build` and relies on Go's build cache for speed, rather than tracking source files as `make` prerequisites. All targets are declared `.PHONY`.

`make run` with no `ARGS` starts an interactive session, because its stdin is the terminal.

## Repository Hygiene

`bin/` and `.DS_Store` (macOS folder metadata) are listed in `.gitignore`.

## Testing

A functional test in `test/functional/` runs `make install INSTALL_DIR=<temporary directory>` from the repository root and checks that `<temporary directory>/c64sh` exists, is executable, and runs `PRINT "HELLO"` correctly from piped stdin. The test is skipped if `make` is not on `PATH`.

## Decisions & Alternatives

| Decision | Chosen | Alternatives Considered | Rationale |
|---|---|---|---|
| Tool | GNU/BSD `make` | Plain `go` commands; `just`; `task` | `make` is preinstalled on macOS and Linux. Named targets bundle steps (`run` builds first) and install to `~/bin`, which `go install` (`$GOPATH/bin`) does not. |
| Rebuild detection | Always run `go build`; Go's cache decides | Source files as `make` prerequisites | Go already tracks dependencies precisely, including imported packages and embedded files; duplicating that in `make` is error-prone. |
| Install method | Copy with mode `0755` into `INSTALL_DIR` | Symlink to `bin/c64sh`; `go install` with `GOBIN` | A copy keeps working if the repository moves or `make clean` runs. |
| Default install directory | `~/bin` | `/usr/local/bin`; `~/.local/bin` | User-owned (no `sudo`) and a long-standing convention on macOS and Linux. Overridable for users who prefer another location. |

## Open Questions & Future Decisions

### Deferred
1. Version stamping (`c64sh --version` with the Git revision via `-ldflags`) is added if and when releases are published.

## References

- HLD *Build and installation*
