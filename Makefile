# @spec BUILD-002, BUILD-003, BUILD-004, BUILD-005, BUILD-006, BUILD-007, BUILD-008, BUILD-010

GO ?= go
BIN ?= bin/c64sh
INSTALL_DIR ?= $(HOME)/bin
ARGS ?=

.DEFAULT_GOAL := build
.PHONY: build run install test update-snapshots clean

# Compile c64sh. Go's build cache decides what needs rebuilding.
build:
	@$(GO) build -o $(BIN) ./cmd/c64sh

# Build, then start c64sh with ARGS (interactive when ARGS is empty).
run: build
	@$(BIN) $(ARGS)

# Build, then copy c64sh into INSTALL_DIR (default ~/bin).
install: build
	@mkdir -p "$(INSTALL_DIR)"
	@install -m 0755 $(BIN) "$(INSTALL_DIR)/c64sh"

test:
	@$(GO) test ./...

# Rewrite the snapshots of examples/ from current output; review the diff.
update-snapshots:
	@UPDATE_SNAPS=true $(GO) test ./test/snapshot

clean:
	@rm -rf bin
