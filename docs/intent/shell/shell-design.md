---
parent: high-level-design
prefix: SHELL
---

# Shell

## Context and Design Philosophy

The shell is the program users run. It decides how input arrives (an interactive terminal, a script file, or piped stdin), sends each line through lexer → parser → interpreter, and reports errors. It is the only component that knows about processes, terminals, exit statuses, and error display.

It follows the HLD's *C64 language, Unix I/O* tenet: the language behaves like a C64, but the process behaves like a Unix command. Program output goes to stdout, errors go to stderr, and failures set a non-zero exit status.

The shell takes its streams as parameters, so functional tests can run it in-process and capture everything it writes.

## Entry Point and API

`cmd/c64sh/main.go` is a thin wrapper:

```go
func main() {
    os.Exit(shell.Main(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
```

```go
package shell

// Main runs c64sh with the given command-line arguments and streams and
// returns the process exit status.
func Main(args []string, stdin io.Reader, stdout, stderr io.Writer) int
```

## Command Line

```
c64sh            interactive if stdin is a terminal, otherwise read lines from stdin
c64sh FILE       run FILE line by line
c64sh -h|--help  print usage to stdout and exit 0
```

- An argument beginning with `-` is an option. `-h` and `--help` are the only options; any other option is a usage error.
- More than one non-option argument is a usage error.
- A usage error prints `c64sh: ` followed by a short message and the usage text to stderr, and exits with status 2.
- If `FILE` cannot be opened or read (it does not exist, is a directory, lacks read permission, or a read fails partway through), the shell prints `c64sh: FILE: <reason>` to stderr (for example `c64sh: hello.bas: no such file or directory`) and exits with status 2. Lines already run before a mid-file read failure keep their output.
- If reading stdin fails with an I/O error, the shell prints `c64sh: stdin: <reason>` to stderr and exits with status 2.

When the kernel runs a script beginning with `#!/usr/bin/env c64sh`, it invokes `c64sh FILE`, so scripts are the `c64sh FILE` case.

## Modes

The shell runs in one of two modes, chosen at startup:

| Mode | Chosen when | Prompt and banner | On a BASIC error |
|---|---|---|---|
| **Interactive** | No `FILE` argument and stdin is a terminal | Yes | Report it and read the next line |
| **Script** | A `FILE` argument, or stdin is not a terminal | No | Report it and stop with exit status 1 |

`stdin` is a terminal when it is an `*os.File` for which `golang.org/x/term.IsTerminal` reports true. Any other reader (a pipe, a regular file, `/dev/null`, or a test's `strings.Reader`) is not. Functional tests reach interactive mode through `Run` (see *Test Seam*).

Both modes run every line in direct mode (see HLD *Direct mode and program mode*): each line is executed immediately and independently.

### Interactive mode

On startup the shell writes a C64-style banner and a ready prompt to **stderr**:

```

    **** C64SH BASIC V2 ****

READY.
```

It then reads lines. After each non-blank line has run (successfully or with an error), it writes `READY.` on its own line to stderr. Before it, the shell calls the interpreter's `FreshLine`, which writes a newline to stdout if program output left the line unfinished (for example after `PRINT "A";`), so `READY.` starts on a fresh line, as on a C64. A blank line (empty or only spaces and tabs) does nothing and prints no `READY.`, as on a C64.

End of input (Ctrl-D at the start of a line) writes a newline to stderr, so the user's own shell prompt starts on a fresh line, and ends the session with exit status 0. Ctrl-C while a line is being typed discards that line (see *Line Editing*); Ctrl-C while a line is running terminates the process with the default signal behavior.

The banner and `READY.` go to stderr, as `bash` writes its prompt, so `c64sh > out.txt` captures only program output.

### Line Editing

When interactive mode runs with **both stdin and stderr connected to a terminal**, lines are read with the line editor of `golang.org/x/term` (`term.Terminal`). Otherwise, including every test that passes a reader to `Run`, lines are read as described in *Line Handling*, byte for byte.

| Key | Effect |
|---|---|
| Up, Ctrl-P | Replace the line with the previous line from the session's history |
| Down, Ctrl-N | Replace it with the next, more recent line; past the most recent, the line being typed before navigating |
| Left, Right, Ctrl-B, Ctrl-F | Move the cursor within the line |
| Home, End, Ctrl-A, Ctrl-E | Move to the start or end of the line |
| Backspace, Delete | Delete the character before or under the cursor |
| Ctrl-U, Ctrl-K, Ctrl-W | Delete to the start of the line, to its end, or the previous word |
| Ctrl-L | Clear the screen |
| Return | Run the line |
| Ctrl-C | Discard the line: it is neither run nor added to history, and a new line is read |
| Ctrl-D | On an empty line, end of input (ends the session); otherwise, delete the character under the cursor |

**History** holds the most recent 100 lines entered in the session, oldest dropped first. A line is added when it is run; blank lines and discarded lines are not added. History is also saved to a file, so it carries over between sessions (see *History File*).

### History File

When the line editor is used and a history file is configured (`Config.HistoryFile`), the editor's history is kept in that file as well as in memory.

- **Location.** `Main` uses the path in the environment variable `C64SH_HISTORY` if it is set; an empty value turns the history file off. Otherwise it uses `.c64sh_history` in the user's home directory. If the home directory cannot be determined, there is no history file.
- **Format.** Plain text, one line of input per line, oldest first, each followed by `\n`. Lines never contain a line feed, since input is split at line feeds.
- **Loading.** At the start of the session the file is read, a trailing `\r` is removed from each line (for a file edited on Windows), blank lines are skipped, and the most recent 100 lines become the history, so the up arrow recalls lines from earlier sessions. A missing file is an empty history.
- **Saving.** Each time a line is added to history, the whole history (at most 100 lines) is written to a temporary file in the same directory, which is then renamed over the history file. The file is therefore never left half-written, and a session killed at any moment keeps every line added before that moment. The file is created with permissions `0600`, readable only by the user, since commands may contain private text.
- **Errors.** If the file exists but cannot be read, or cannot be written, the shell writes one warning, `c64sh: history: <reason>`, to stderr for the session and continues with the history in memory, retrying later saves silently.
- **Several sessions at once.** Each session keeps its own history in memory and rewrites the file with it, so the session that saves last determines the file's contents; lines from a concurrent session that saved earlier may be lost from the file, though not from that session's memory.
- **Only for the line editor.** Script mode, pipes, and any input read without the editor never read or write the history file.

**Raw mode.** Arrow keys and Ctrl-C reach the editor only when the terminal is in raw mode. The shell switches stdin to raw mode just before reading each line and restores the terminal's previous mode as soon as the line is read, before it runs. Program output, errors, and `READY.` are therefore written with the terminal in its normal mode, exactly as without line editing. The previous mode is restored on every path out of line reading, including errors.

**Echo** of the keystrokes goes to stderr, with the banner and `READY.`, so redirecting stdout still captures only program output. The editor is told the terminal's size before each line, so long lines wrap correctly; a size that cannot be read, or is reported as 0 (as some pseudo-terminals do), leaves the editor's default of 80 by 24.

**Ctrl-C.** `term.Terminal.ReadLine` reports Ctrl-C as end of input, the same as Ctrl-D on an empty line. To discard the line instead, the shell reads the terminal through a small filter: when a Ctrl-C byte arrives, the filter replaces it with Return, holds back any bytes after it until the next read (so they begin the next line), and records that the line was cancelled. The shell then discards the line that `ReadLine` returns and reads another; the history ignores lines added while the cancelled flag is set.

**Pasted text** containing several lines is run one line at a time, as if typed. The editor reports pasted lines with `term.ErrPasteIndicator`, which the shell treats as an ordinary line.

### Script mode

Lines are read from `FILE`, or from stdin when there is no `FILE`.

- If the **first** line starts with `#!`, it is skipped. A `#!` line anywhere else is ordinary input (and a syntax error).
- Blank lines are skipped.
- A BASIC error is reported (see *Error display*) and the shell stops with exit status 1. Lines after it do not run.
- If all lines run without error, the exit status is 0.
- When input ends, nothing is added to the output. A script whose last output is `PRINT "A";` ends with `A` and no newline, as `printf "A"` does in a Unix shell; this lets scripts produce output without a trailing newline on purpose.

Stopping at the first error matches a C64 running a program, which halts at the failing line, and prevents later lines from running on the assumption that earlier ones succeeded.

## Line Handling

Lines have no length limit. The shell reads with a reader that grows its buffer as needed, rather than one with a fixed maximum line size such as `bufio.Scanner`'s 64 KB default.

For every line, in both modes:

1. Remove the trailing `\n` and, if present, a trailing `\r` (so files with Windows line endings work). A final line without a line terminator is still run.
2. If the line is blank, skip it.
3. `tokens := lexer.Lex(line)`
4. `ast, perr := parser.Parse(tokens)`
5. `err := interp.Exec(ast)` — runs the statements the parser returned, including a partially parsed `PRINT` that ends at the syntax error (see the parser design, *Errors inside PRINT*).
6. If `err` is a BASIC error, display it. Otherwise, if `perr` is non-nil, display it.
7. If `err` is a write error (not a BASIC error), stop and exit with status 1 without further output.

When stdout is a pipe whose reader has exited (as in `c64sh script.bas | head -1`), the Go runtime's default SIGPIPE handling terminates the process silently on the next write. The shell keeps that default, which is how Unix commands behave in pipelines; step 7 therefore applies to other write failures, such as a full disk.

Steps 5–6 reproduce the C64 order of events: `PRINT "A":PRINT "B"@` prints `A` and `B`, and then reports `?SYNTAX  ERROR`. Because the parser's error also travels inside the `BadItem`, `Exec` usually returns it itself; step 6 reports it once either way.

## Errors

### Error type

BASIC errors are defined in `internal/basicerr`:

```go
package basicerr

type Kind int

const (
    Syntax        Kind = iota // SYNTAX
    StringTooLong             // STRING TOO LONG
    TypeMismatch              // TYPE MISMATCH
    Overflow                  // OVERFLOW
    DivisionByZero            // DIVISION BY ZERO
    IllegalQuantity           // ILLEGAL QUANTITY
)

type Error struct{ Kind Kind }

func (e *Error) Error() string // the C64 name, e.g. "SYNTAX"
```

Each kind maps to the name the C64 uses in its message. New kinds are added as the language grows, always with the C64's own name (tenet *Authentic errors over helpful ones*).

### Error display

A BASIC error is written to stderr as:

```
?<NAME>  ERROR
```

with **two spaces** between the name and `ERROR`, followed by a newline, exactly as a C64 prints it in direct mode — for example `?SYNTAX  ERROR` and `?STRING TOO LONG  ERROR`.

A C64 moves to a new line before printing an error. The shell does the same: before writing the error to stderr, it calls the interpreter's `FreshLine`, which writes a newline to **stdout** if program output left the line unfinished (for example after `PRINT "A";`, on this line or an earlier one). This keeps the error on its own line in a terminal, and keeps redirected stdout ending in a complete line.

### Exit statuses

| Status | Meaning |
|---|---|
| 0 | Success; or interactive session ended by end of input |
| 1 | Script mode stopped by a BASIC error, or program output could not be written (other than to a closed pipe, which ends the process through SIGPIPE) |
| 2 | Usage error; `FILE` could not be opened or read; stdin could not be read |

## Output Tracking

The shell does not track program output itself. The interpreter writes all program output, keeps the cursor column, and provides `FreshLine`, which the shell calls before errors and `READY.` (see the interpreter design). Every newline on stdout is therefore written by the interpreter, so its column is always accurate. A write error from `FreshLine` is ignored: the next program output hits the same failure and stops the shell.

## Test Seam

`Main` decides interactive mode from its real `stdin`. Functional tests need to exercise interactive behavior with a string as input, so the package also exposes:

```go
// Run is Main with the mode chosen by the caller instead of detected.
func Run(cfg Config, stdin io.Reader, stdout, stderr io.Writer) int

type Config struct {
    Interactive bool
    File        string // empty: read from stdin
    HistoryFile string // line-editor history file; empty: none
}
```

The two fields are independent: `Interactive` selects the behavior (banner, `READY.`, continuing after errors) and `File` selects the input (the named file, or stdin when empty). `Main` never sets both, but `Run` needs no special case when a test does.

`Main` parses arguments, detects the terminal, and calls `Run`. Tests call `Run` directly for interactive-mode cases and `Main` for argument handling and script mode.

## Functional Tests

Functional tests live in `test/functional/` (package `functional_test`). Each test gives the shell input and asserts on stdout, stderr, and exit status. Most tests call `shell.Main` or `shell.Run` in-process. One test builds the real binary with `go build`, writes a temporary script beginning with `#!/usr/bin/env c64sh`, marks it executable, puts the binary's directory first on `PATH`, runs the script, and checks its output.

## Decisions & Alternatives

| Decision | Chosen | Alternatives Considered | Rationale |
|---|---|---|---|
| Mode detection | Terminal on stdin and no `FILE` → interactive | Explicit `-i` flag; always interactive without `FILE` | Same rule as `bash`, `python`, and `sh`, so `echo 'PRINT "X"' \| c64sh` behaves as a script without flags. |
| Terminal detection | `golang.org/x/term.IsTerminal` | `os.File.Stat()` character-device check | The character-device check also matches `/dev/null` and other devices, so `c64sh < /dev/null` would wrongly start an interactive session. `IsTerminal` asks the operating system whether the descriptor is a terminal. |
| Prompt and banner stream | stderr | stdout | Tenet *C64 language, Unix I/O*: redirected stdout contains only program output. Matches `bash`. |
| Prompt style | C64 `READY.` after each command, banner at start | A per-line prompt such as `> `; no prompt | Gives the interactive session C64 character while staying off stdout. A C64 has no per-line prompt, only `READY.` after each command. |
| Banner text | `**** C64SH BASIC V2 ****` | The C64's own `**** COMMODORE 64 BASIC V2 ****` and `64K RAM SYSTEM  38911 BASIC BYTES FREE` | Evokes the C64 start-up screen without presenting c64sh as a Commodore product, and without claiming a memory size that does not apply. |
| Line reading | Unbounded line length | `bufio.Scanner` with its 64 KB default limit | A BASIC line in a script has no reason to fail because of an implementation buffer size. |
| Script errors | Stop at first error, exit 1 | Continue and exit non-zero at the end; continue and exit 0 | A C64 program halts at the failing line. Continuing risks later lines acting on the assumption that earlier ones worked. |
| Error format | `?NAME  ERROR` with two spaces, to stderr | Single space; to stdout; include a line number or column | Tenet *Authentic errors over helpful ones*: this is the exact direct-mode C64 message. stderr per tenet *C64 language, Unix I/O*. |
| Newline before error | Written to stdout when program output is mid-line | Write nothing; write the newline to stderr | A C64 starts error messages on a new line. Writing it to stdout keeps redirected output ending with a complete line. |
| Exit statuses | 0 / 1 BASIC or output error / 2 usage | Single non-zero status | Distinguishes "the script failed" from "c64sh was invoked wrongly", as `grep` and `diff` do. |
| Shebang handling | Skip only a first line beginning with `#!` | Treat `#` as a comment everywhere | `#` is not a comment in BASIC V2; only the shebang needs special handling. |
| Error type location | `internal/basicerr` | Inside `parser` or `interp` | Both parser and interpreter produce BASIC errors and the shell displays them; a neutral package avoids an import cycle and a false owner. |
| End of script output | Left as written, even mid-line | Add a newline if output ends mid-line | Trailing `;` is how BASIC suppresses a newline; honoring it at the end of a script lets scripts write partial lines on purpose, like `printf`. |
| Closed-pipe writes | Go's default SIGPIPE termination | Ignore SIGPIPE and exit with status 1 and a message | Silent termination is what Unix commands do when a downstream reader such as `head` exits; an error message there would be noise. |
| Line editor | `golang.org/x/term`'s `Terminal` | `peterh/liner`, `chzyer/readline`; a hand-written editor | See HLD *Interactive line editing*. `Terminal` runs on any reader and writer, so the editor is tested by feeding it key sequences. |
| Raw mode scope | Only while a line is read | The whole session | Program output and errors keep the terminal's normal newline handling, and a crash while a BASIC line runs cannot leave the terminal in raw mode. |
| Ctrl-C while typing | Discard the line, as `bash` does | End the session (`term.Terminal`'s own behavior) | A cancelled line should not cost the whole session and its history. The input filter needed is a few lines. |
| Line editing condition | stdin and stderr both terminals | stdin a terminal | The editor echoes to stderr; echoing into a redirected stderr file would put keystrokes and cursor codes in it. |
| History persistence | Saved to `~/.c64sh_history`, rewritten after each line | In memory only; saved on exit; appended line by line | See HLD *History between sessions*. |
| History file location | `$C64SH_HISTORY`, else `~/.c64sh_history`; empty `C64SH_HISTORY` disables | A fixed path; an XDG state directory | Mirrors `bash`'s `HISTFILE`: a user can move or turn off the file, and tests can point it at a temporary directory. A single dotfile in the home directory is the long-standing shell convention. |
| History file errors | One warning per session, then continue in memory | Silent; fail the session | History is a convenience; a read-only home directory should not stop anyone from using c64sh, but a silent failure would leave users wondering why history is missing. |
| Concurrent sessions | Last session to save wins | File locking and merging | Simple and predictable; losing some history from overlapping sessions is a minor cost for an interactive convenience. |
| Test seam | `Run(Config, …)` beside `Main` | Inject a terminal-detection function; a pseudo-terminal in tests | Keeps `Main` simple and makes interactive tests plain string-in, string-out. |

## Open Questions & Future Decisions

### Deferred
1. When program mode is added, the shell holds the stored program, and error messages from running programs gain the C64's ` IN <line>` suffix (`?SYNTAX  ERROR IN 10`). How script files interact with program mode is decided then.
2. Interactive line editing (history, arrow keys) is out of scope for now (HLD *Non-Goals*).
3. How user-guide examples are kept in sync with functional tests (hand-copied cases or extraction from the guide) is decided when the user guide is written.

## References

- `docs/intent/lexer/lexer-design.md`, `docs/intent/parser/parser-design.md`, `docs/intent/interp/interp-design.md`
- *Commodore 64 Programmer's Reference Guide*, appendix on error messages
