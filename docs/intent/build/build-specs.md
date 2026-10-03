# Build Specs

Design: `build-design.md`

- [x] **BUILD-001**: The repository root shall contain a `go.mod` declaring the module path `github.com/bryanesmith/c64sh`.
- [x] **BUILD-002**: When `make` or `make build` is run at the repository root, it shall compile `./cmd/c64sh` to `bin/c64sh`.
- [x] **BUILD-003**: When `make run` is run, it shall build `bin/c64sh` and then run it with the arguments in `ARGS`, connected to the invoking terminal's stdin, stdout, and stderr.
- [x] **BUILD-004**: When `make install` is run, it shall build `bin/c64sh`, create `INSTALL_DIR` if it does not exist, and copy the binary to `INSTALL_DIR/c64sh` with mode `0755`, replacing any existing file.
- [x] **BUILD-005**: The Makefile shall default `INSTALL_DIR` to `$(HOME)/bin`, `BIN` to `bin/c64sh`, `GO` to `go`, and `ARGS` to empty, each overridable on the `make` command line or from the environment.
- [x] **BUILD-006**: When `make test` is run, it shall run `go test ./...`.
- [x] **BUILD-007**: When `make clean` is run, it shall remove the `bin/` directory.
- [x] **BUILD-008**: The Makefile's recipes shall not echo the commands they run, so `make run` shows only c64sh's own output.
- [x] **BUILD-009**: `.gitignore` shall exclude the `bin/` directory and `.DS_Store` files.
- [x] **BUILD-010**: When `make update-snapshots` is run, it shall run `go test ./test/snapshot ./test/tutorial` with the environment variable `UPDATE_SNAPS` set to `true`.
