# High-Level Design: c64sh

## Problem

Commodore 64 BASIC V2 is a small language that many people learned first, and its direct mode — type a line, press RETURN, see the result — is a good interactive experience. Using it today means running a full machine emulator: a separate window, emulated keyboard and screen, and no way to pipe text in or out or write a script that behaves like any other command-line tool.

c64sh is a shell that speaks C64 BASIC V2 in an ordinary terminal. It can be run interactively like `bash` or `zsh`, or as a script interpreter (`#!/usr/bin/env c64sh`), with input from stdin and output to stdout/stderr.

## Approach

### Classic interpreter pipeline

Each input line passes through three independent stages:

1. **Lexer** — turns characters into tokens (`PRINT`, `REM` with its comment text, string literal, `;`, `,`, `+`, `:`, end of line).
2. **Parser** — turns tokens into a typed abstract syntax tree (AST).
3. **Interpreter** — walks the AST and performs its effects (writing output).

Each stage is its own Go package with its own unit tests beside the code, so each can be understood, tested, and extended on its own.

### The code defines the syntax

The syntax c64sh accepts is defined in one place: the hand-written lexer and parser. The parser is a recursive-descent parser with one function per grammar rule, and each function carries its rule, in Extended Backus-Naur Form (the notation of the Go language specification), as a comment directly above it. The lexer documents each token rule the same way. Read together, those comments are the grammar, kept beside the code that implements each rule.

The lexer and parser unit tests pin down exactly what is accepted. For users, the user guide describes the syntax in prose and `examples/` shows it in use.

### Direct mode and program mode

C64 BASIC has two ways of handling a line of input:

- **Direct mode** — a line that does not begin with a line number is executed immediately when RETURN is pressed. `PRINT "HELLO"` prints `HELLO` at once.
- **Program mode** — a line that begins with a line number (`10 PRINT "HELLO"`) is not executed. It is stored in the program, in line-number order, replacing any existing line with the same number. Stored lines execute when `RUN` is entered, and commands such as `LIST`, `NEW`, and `GOTO` operate on the stored program.

c64sh currently supports **direct mode only**. Every line, whether typed at the prompt or read from a script file or stdin, is executed immediately and independently. A line beginning with a line number is not yet accepted and fails with `?SYNTAX  ERROR`.

Program mode is a planned feature. It adds a stored program to the shell's state and a line-number prefix to the `Line` rule; the lexer → parser → interpreter pipeline is unchanged. How script files relate to program mode (for example, whether a file of numbered lines is loaded and run automatically) is decided when program mode is designed.

### Incremental language growth

The language grows one feature at a time. The language currently supports `PRINT` with string and number arguments and `REM` comments, in direct mode. Each new feature (numbers and math, functions such as `CHR$`, variables, program mode with line numbers, `LOAD`/`SAVE`) extends the lexer and parser, with their rule comments, then the interpreter, and gets its own tests at every layer. It also adds or extends a numbered example script in `examples/` that exercises the feature in many ways, with a snapshot test recording that script's exact output.

## Target Users

- **Retro-computing hobbyists** who know C64 BASIC and want to use it in a modern terminal without an emulator.
- **Learners** who want a small, readable language implementation to study: a parser whose functions document their own grammar rules, a conventional pipeline, and tests at every stage.
- **Scripters** who want to write short C64 BASIC scripts that run as ordinary Unix commands and compose with pipes and redirection.

## Goals

- `c64sh` started from a terminal gives an interactive prompt; each line entered is executed immediately (direct mode).
- `c64sh FILE` and executable files beginning with `#!/usr/bin/env c64sh` execute each line of the file in direct mode, as if typed. Input piped on stdin is executed the same way.
- `PRINT` with string literals behaves as it does on a C64: optional space after the keyword (`PRINT"X"`), `;` and `+` join strings, and `:` separates statements on one line. `,` moves to the next 10-column print zone, as on a C64.
- Numbers are written and printed as on a C64: literals such as `5`, `3.14`, `.5`, and `1E3`; printed with a leading space (or `-`) and a trailing space, rounded to 9 significant digits, with no leading zero before the decimal point (`.5`), and in scientific notation below 0.01 and from 1E9 up (`1E-03`, `1E+09`). Numbers and strings mix freely in `PRINT` (`? "5*9=";45`), and `+` adds numbers as well as joining strings.
- `REM` comments behave as they do on a C64: everything after `REM` to the end of the line is ignored, including colons, so comments can document scripts and follow other statements (`PRINT "A":REM SHOW A`).
- Input the shell does not accept produces the error a C64 would print for it (for example `?SYNTAX  ERROR`).
- The lexer, parser, and interpreter each have unit tests; functional tests run the whole shell on given input and assert on captured stdout and stderr.
- `examples/` holds numbered, executable BASIC scripts (`001-hello-world.bas`, …) that show each language feature in many forms, commented for readers. Snapshot tests run every example and compare its stdout, stderr, and exit status with a recorded snapshot, so any change in behavior appears as a reviewable diff.
- `make build`, `make run`, and `make install` build the binary, build and start the shell, and install `c64sh` into `~/bin`.
- `README.md` gives a short description, build and run instructions, and one example, and links to a user guide at `docs/user-guide.md`.

## Non-Goals

- **Emulating the C64 machine.** No screen memory, 40-column wrapping, colors, cursor control, PETSCII graphics, `PEEK`/`POKE`, or timing.
- **Supporting the whole language at once.** Arithmetic beyond `+` and other operators, variables, functions, program mode (see *Direct mode and program mode*), and device commands are future features, added one at a time.
- **Rich interactive line editing** (history, arrow keys, tab completion). The interactive prompt reads plain lines.
- **Extensions beyond BASIC V2.** No keywords from BASIC 3.5/7.0 or third-party extensions.
- **Real device I/O.** When `LOAD`/`SAVE` are added, the storage behind them will be replaceable, so tests never touch the real filesystem.

## Tenets

- **C64 language, Unix I/O.** What the language accepts and computes follows C64 BASIC V2, including its lenient syntax (an unterminated string literal is accepted). How the program behaves as a process follows Unix conventions: errors go to stderr, failures set a non-zero exit status, and C64 screen conventions that make no sense in a terminal are replaced by their terminal equivalents: the cursor-right moves a C64 uses to reach a print zone become spaces. The opposite choice would be to reproduce the C64 screen exactly.
- **Authentic errors over helpful ones.** When input is valid C64 BASIC that c64sh does not yet support, or is invalid, c64sh reports the error a real C64 would (`?SYNTAX  ERROR` and its siblings) rather than inventing clearer messages such as "not yet supported."

## System Design

```mermaid
flowchart LR
    subgraph shell ["shell (internal/shell)"]
        R[line source<br/>terminal / file / stdin]
    end
    R -->|line| L[lexer<br/>internal/lexer]
    L -->|tokens| P[parser<br/>internal/parser]
    P -->|AST| I[interpreter<br/>internal/interp]
    I -->|output| O[(stdout)]
    L -. error .-> E[(stderr)]
    P -. error .-> E
    I -. error .-> E
```

### Components

| Component | Package | Responsibility |
|---|---|---|
| Entry point | `cmd/c64sh` | Passes the command-line arguments and the real stdin/stdout/stderr to the shell, and exits with the status it returns. |
| Shell | `internal/shell` (+ `internal/basicerr`) | Parses command-line arguments, chooses interactive or script mode, reads lines, runs each through the pipeline, and reports errors. Takes its input and output streams as parameters so it can be run inside tests. Defines the BASIC error type. |
| Lexer | `internal/lexer` (+ `internal/token`) | Turns a line of characters into tokens. Handles C64 lexical rules such as keywords with no following space. |
| Parser | `internal/parser` (+ `internal/ast`) | Turns tokens into an AST using recursive descent, one function per grammar rule, each documented with its rule in EBNF. |
| Interpreter | `internal/interp` | Executes the AST by switching on node type. Writes program output to a given writer. |

### Errors

A BASIC error (such as `SYNTAX`) is an ordinary Go error value that carries the C64 error name. Any stage can return one; the shell formats it the way a C64 prints it and writes it to stderr. An error stops the rest of the line it occurs in, as on a C64.

### Tests

- **Unit tests** live beside the code in each package (`lexer_test.go`, `parser_test.go`, …).
- **Functional tests** in `test/functional/` run the shell in-process with given input, capture stdout and stderr, and assert on them. One test builds the real `c64sh` binary and runs a script through its `#!/usr/bin/env c64sh` line.
- **Snapshot tests** in `test/snapshot/` run each script in `examples/` through the shell, as `c64sh FILE` does, and compare the result with a recorded snapshot in `test/snapshot/testdata/`. `make update-snapshots` rewrites the snapshots from current behavior; the resulting diff is reviewed like code. The same tests check that the examples follow their conventions (numbered names, `#!` line, a leading `REM` comment).

### Build and installation

A `Makefile` at the repository root is the single entry point for building, running, testing, and installing:

| Target | Effect |
|---|---|
| `make build` | Compiles `cmd/c64sh` to `bin/c64sh` in the repository. `bin/` is build output and is not committed. |
| `make run` | Runs `make build`, then starts `bin/c64sh`. Arguments are passed with `ARGS`, e.g. `make run ARGS=hello.bas`. |
| `make install` | Runs `make build`, then copies the binary to `$(INSTALL_DIR)/c64sh`, creating the directory if needed. `INSTALL_DIR` defaults to `~/bin` and can be overridden (`make install INSTALL_DIR=/usr/local/bin`). |
| `make test` | Runs all unit, functional, and snapshot tests (`go test ./...`), failing if any example's output differs from its snapshot. |
| `make update-snapshots` | Reruns the snapshot tests, rewriting each snapshot from the current output. |
| `make clean` | Removes `bin/`. |

`make build` is the default target. For `#!/usr/bin/env c64sh` scripts and a bare `c64sh` command to work, the install directory must be on the user's `PATH`; the user guide explains this. `go install ./cmd/c64sh` also works for users who prefer Go's own tooling, but the Makefile is the documented path.

### Repository layout

```
Makefile             build, run, test, install
bin/                 build output (not committed)
cmd/c64sh/           entry point
internal/basicerr/   BASIC error type (SYNTAX, …)
internal/token/      token types
internal/lexer/
internal/ast/        AST node types
internal/parser/
internal/interp/
internal/shell/
test/functional/     end-to-end tests
test/snapshot/       snapshot tests of examples/ (snapshots in testdata/)
examples/            numbered example scripts, one or more per feature
docs/                design docs (HLD, docs/intent/) and user-guide.md
README.md
```

## Key Design Decisions

| Decision | Chosen | Alternatives considered | Rationale |
|---|---|---|---|
| Implementation language | Go | — | Single static binary that installs with `go install`; a standard library strong enough for the whole project. |
| Pipeline | Lexer → parser → interpreter over an AST | Tokenize and interpret directly from the token stream, as the C64 ROM does | Separate stages can be tested separately, and an AST gives future features (program mode, expressions) a clean structure to extend. |
| Where the syntax is defined | The hand-written lexer and recursive-descent parser, each rule documented in EBNF beside its implementation | (a) A separate EBNF grammar file kept aligned with the parser by conformance tests; (b) a parser that interprets an EBNF file at runtime; (c) a parser generated from an EBNF file at build time | A separate grammar file is a second description of the same syntax: conformance tests can check that its rule names match the parser, but not that the rules mean the same thing, so it invites drift and adds code. A runtime grammar interpreter still needs hand-written code per rule to build typed AST nodes, and cannot express C64 behaviors such as `REM` consuming the rest of the line or printing the items before a syntax error. A generator is a separate project, too large for BASIC V2's small grammar. |
| AST evaluation | Type switch over sealed node interfaces (unexported marker methods), with a `default` case that panics | Visitor pattern | Go's type switch does the job the Visitor pattern exists for, without an `Accept`/`Visit` method pair per node type. There is a single pass over the tree (execution), so the Visitor's support for many passes buys nothing. The panicking `default` plus unit tests catch unhandled node types. |
| Build tooling | `Makefile` with `build`, `run`, `install`, `test`, `clean` | Plain `go build`/`go install` commands; a task runner such as `just` or `task` | `make` is already present on macOS and Linux, so there is nothing extra to install. Named targets give short, memorable commands that bundle steps (`run` builds first) and a user-level install location (`~/bin`, no `sudo`) that `go install`'s `$GOPATH/bin` does not match. |
| Number representation | Go `float64`, formatted as the C64 formats numbers | Emulating the C64's 5-byte floating-point format and its arithmetic routines | Rounded to the C64's 9 significant digits, nearly every result prints identically, at a fraction of the effort. The rare last-digit differences, and C64 imprecisions such as slightly-off powers, are not reproduced. Numbers are confined to the interpreter's value type, so an exact emulation could replace `float64` later without touching other components. |
| Stream handling | Shell takes `io.Reader`/`io.Writer` parameters | Shell uses `os.Stdin`/`os.Stdout` directly | Functional tests run the shell in-process and capture output, and the same seam will let future device commands use replaceable storage. |

## Success Metrics

- Every example in the user guide produces exactly the documented output when run through `c64sh`. Falsified by any documented example that does not.
- Every language feature is shown in at least one script in `examples/`, and every script's output matches its snapshot. Falsified by a feature with no example, or a failing snapshot test.
- Every `PRINT` and `REM` form listed under Goals produces the same text a C64 would, with spaces in place of the C64's cursor-right moves. Falsified by any difference in functional tests.
- Adding a new statement touches only the lexer, parser, interpreter, their tests, and `examples/`, not the shell. Falsified if a language feature requires shell changes.

## References

- [The Go Programming Language Specification — Notation](https://go.dev/ref/spec#Notation): the EBNF notation used in the lexer and parser rule comments.
- *Commodore 64 Programmer's Reference Guide* (Commodore, 1982): BASIC V2 keywords, syntax, and error messages.
- [C64 BASIC V2 on C64-Wiki](https://www.c64-wiki.com/wiki/BASIC): keyword reference and behavior notes.
