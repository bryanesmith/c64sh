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

Both modes handle every line as if it were typed at a C64 (see HLD *Direct mode and program mode*): a line beginning with a line number is stored in the interpreter's program, and any other line runs immediately, in direct mode.

### Interactive mode

On startup the shell writes a C64-style banner and a ready prompt to **stderr**:

```

    **** C64SH BASIC V2 ****

READY.
```

It then reads lines. After each non-blank line has run (successfully or with an error), it writes `READY.` on its own line to stderr. Before it, the shell calls the interpreter's `FreshLine`, which writes a newline to stdout if program output left the line unfinished (for example after `PRINT "A";`), so `READY.` starts on a fresh line, as on a C64. A blank line (empty or only spaces and tabs) does nothing and prints no `READY.`, as on a C64. Neither does a numbered line that is stored: a C64 stores the line and waits for the next one without printing `READY.` (`$A52A` returns to the input loop directly). A line number above 63999 is a SYNTAX error, followed by `READY.` like any other error.

End of input (Ctrl-D at the start of a line) writes a newline to stderr, so the user's own shell prompt starts on a fresh line, and ends the session with exit status 0. Ctrl-C while a line is being typed discards that line (see *Line Editing*). Ctrl-C while a line is running stops it (see *Interrupts*): a running program prints `BREAK IN n`, and then `READY.`.

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
- **Ctrl-C** while a line runs stops the script (see *Interrupts*): `BREAK`, or `BREAK IN n` in a program, is written to stderr and the exit status is 130, which Unix shells use for a command ended by Ctrl-C (128 + SIGINT).
- **When input ends, a stored program that the script never ran is run.** If the program is not empty and no `RUN`, `GOTO`, or `GOSUB` has been executed during the script, the shell runs it, exactly as if a final `RUN` line followed; an error in it is reported and sets exit status 1, like any other. So a script of numbered lines runs as a program without a `RUN` line, while a script that runs its program itself, once or several times, is left as written.
- When input ends, nothing is added to the output. A script whose last output is `PRINT "A";` ends with `A` and no newline, as `printf "A"` does in a Unix shell; this lets scripts produce output without a trailing newline on purpose.

Stopping at the first error matches a C64 running a program, which halts at the failing line, and prevents later lines from running on the assumption that earlier ones succeeded.

## Program Files

The shell gives the interpreter storage for `LOAD`, `SAVE`, and `VERIFY` that reads and writes files in the **current directory** (names are relative paths). A file it creates gets permissions `0644`. `WriteFile` with `replace` false creates the file only if it does not exist, so another process cannot slip a file in between a check and the write.

In interactive mode the shell also gives the interpreter stderr for the C64's tape and disk messages (`SAVING HELLO`, `LOADING`), so they appear beside `READY.` and stay out of redirected stdout. In script mode there are no messages.

A `StorageError` from executing a line is reported as `c64sh: ` and the file, then: for a disk file that may not be replaced, `file exists (use REPLACE to replace it)`, with the error's `Replace` (such as `SAVE "@0:HELLO"`); otherwise the reason. When the session ends, the shell calls the interpreter's `CloseFiles`, so that data files a program left open are written, and reports any `StorageError` from it the same way. In interactive mode the shell then writes `READY.` and continues; in script mode it stops with exit status 1, like a BASIC error.

## Terminal Output

`Main` sets `Config.Terminal` when stdout is a terminal, and `Config.NoColor` when the environment variable `NO_COLOR` is set to a non-empty value (the common convention for turning colors off). `Run` passes them to the interpreter (`SetScreen(cfg.Terminal, !cfg.NoColor)`), which translates the C64's screen control codes in program output (see the interpreter design). Colors stay set until a program changes them, as on a C64, so when the session ends, if stdout is a terminal, the shell writes `ESC[0m` to stdout, so that the user's own shell does not inherit a program's color.

### Styling

The shell colors its own text, so a session is easy to read at a glance: what was typed, what the program printed, and what went wrong. This is a c64sh extension; a C64 shows everything in one color. `Main` sets `Config.Styled` when stderr is a terminal and `Config.NoColor` is not set; without it, the shell writes no escape codes of its own.

| Text | Style |
|---|---|
| Typed lines, in the line editor and echoed by `INPUT` at a terminal | cyan, `ESC[36m` |
| The banner and `READY.` | green, `ESC[32m` |
| BASIC errors, `BREAK`, storage failures, and the end-of-input message | red, `ESC[31m` |

The styles are the terminal's own 16 theme colors, so they suit light and dark themes, and stay distinct from programs' colors, which are the C64's own as 24-bit colors. Program output (stdout) is never styled by the shell, so redirected output is unchanged.

Each styled span ends with `ESC[0m` followed by the interpreter's `ScreenState`: the color and reverse video a program has left set. So a color a program sets stays in effect for its later output, as on a C64, even when the shell's styled text comes between.

The line editor colors typing with its prompt: the prompt is the input style alone, which `term.Terminal` counts as zero columns wide, so every redraw of the line is in that color; after each line is read, the editor writes the end of the span. The `INPUT` echo writes each typed character as its own styled span.

## Clock

The interpreter's clock (for `RND(0)`) is the system clock, unless `Config.Clock` is set: tests set a fixed clock, so that programs using it give the same output every run.

## Keyboard Input

The shell gives the interpreter a console (`interp.Console`) for `INPUT` and `GET`. Both read **stdin**, whatever the mode:

- **When stdin is not a terminal** (a pipe or a file), the console reads it as plain text. If the script itself is stdin (no `FILE`), the console shares the shell's reader, so `INPUT` reads the line after the one running, which the shell then does not run: a piped script supplies its own answers, as if typed. `ReadLine` returns the next line without its `\n` and any `\r`, not echoed, so the interpreter writes it after the prompt and the output reads like a C64 screen. `ReadKey` returns the next character, a line end (`\n` or `\r\n`) being the C64's Return, `CHR$(13)`.
- **When stdin is a terminal**, then while a line executes, the shell switches the terminal to unbuffered input without echo, leaving signals on (Ctrl-C still sends SIGINT) and output processing unchanged, and restores the previous mode afterwards. Keys typed while a program runs wait in the terminal, like the C64's keyboard buffer, until `INPUT` or `GET` reads them, or until the program ends and the line editor reads them. `ReadKey` returns at once: a waiting key, or `""`. Return is `CHR$(13)`, Backspace (or Delete) the C64's DEL, `CHR$(20)`, and the arrow keys the C64's cursor keys: up `CHR$(145)`, down `CHR$(17)`, right `CHR$(29)`, left `CHR$(157)`; other escape sequences are ignored. `ReadLine` reads keys until Return, echoing each character to stderr and erasing the last one on Backspace; it checks `stop` while it waits, returning `ErrInterrupted` when Ctrl-C has been pressed. It reports the line as echoed, so the interpreter writes only the newline.

If the console finds the end of input (`interp.ErrEndOfInput`), there is no C64 equivalent: the shell writes `c64sh: stdin: end of input` to stderr, after `FreshLine`, and stops with exit status 1, in either mode.

## Interrupts

While the interpreter executes a line (`Exec`), the shell catches SIGINT, which the terminal sends when Ctrl-C is pressed, and calls the interpreter's `Interrupt`, which stops execution after the current statement with a `BREAK` error (see the interpreter design). Outside `Exec` the shell leaves SIGINT alone: while the line editor reads a line the terminal is in raw mode, so Ctrl-C arrives as a key and discards the line, and while a script waits for input, Ctrl-C ends c64sh with the default signal behavior, as it ends any Unix command.

The shell reports a `BREAK` like an error, but in the C64's form: `BREAK`, or `BREAK IN 20` in a running program (`$A381`), on stderr, after `FreshLine`. In interactive mode `READY.` follows and the session continues; in script mode the shell stops with exit status 130.

## Line Handling

Lines have no length limit. The shell reads with a reader that grows its buffer as needed, rather than one with a fixed maximum line size such as `bufio.Scanner`'s 64 KB default.

For every line, in both modes:

1. Remove the trailing `\n` and, if present, a trailing `\r` (so files with Windows line endings work). A final line without a line terminator is still run.
2. If the line is blank, skip it.
3. If `lexer.LineNumber(line)` finds a line number, store the line: `interp.Store(n, rest)`, which also deletes line `n` when `rest` is empty. Nothing else happens for that line, and no `READY.` follows it. A line number above 63999 is a SYNTAX error, displayed as in step 6.
4. Otherwise, `tokens := lexer.Lex(line)` and `ast, perr := parser.Parse(tokens)`.
5. `err := interp.Exec(ast)` — runs the statements the parser returned. A syntax error is part of that tree, as a `BadItem` or `BadStmt` at the point where parsing failed (see the parser design, *Errors*), so it is returned only if execution reaches it.
6. If `err` is a BASIC error, display it. The parser's own returned error is not displayed: a syntax error that execution does not reach, such as one after a false `IF`, is never reported, as on a C64.
7. If `err` is a write error (not a BASIC error), stop and exit with status 1 without further output.

When stdout is a pipe whose reader has exited (as in `c64sh script.bas | head -1`), the Go runtime's default SIGPIPE handling terminates the process silently on the next write. The shell keeps that default, which is how Unix commands behave in pipelines; step 7 therefore applies to other write failures, such as a full disk.

Steps 5–6 reproduce the C64 order of events: `PRINT "A":PRINT "B"@` prints `A` and `B`, and then reports `?SYNTAX  ERROR`, while `IF 0 THEN PRINT "A"@` prints and reports nothing.

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
    UndefdStatement           // UNDEF'D STATEMENT
    NextWithoutFor            // NEXT WITHOUT FOR
    OutOfMemory               // OUT OF MEMORY
    ReturnWithoutGosub        // RETURN WITHOUT GOSUB
    IllegalDirect             // ILLEGAL DIRECT
    UndefdFunction            // UNDEF'D FUNCTION
    FileNotFound              // FILE NOT FOUND
    DeviceNotPresent          // DEVICE NOT PRESENT
    IllegalDeviceNumber       // ILLEGAL DEVICE NUMBER
    MissingFileName           // MISSING FILE NAME
    Load                      // LOAD
    Verify                    // VERIFY
    FileOpen                  // FILE OPEN
    FileNotOpen               // FILE NOT OPEN
    NotInputFile              // NOT INPUT FILE
    NotOutputFile             // NOT OUTPUT FILE
    TooManyFiles              // TOO MANY FILES
    FileData                  // FILE DATA
    BadSubscript              // BAD SUBSCRIPT
    RedimdArray               // REDIM'D ARRAY
    OutOfData                 // OUT OF DATA
    Break                     // BREAK: execution stopped by Ctrl-C
)

type Error struct {
    Kind    Kind
    Line    int  // the program line where the error occurred, if HasLine
    HasLine bool // false for an error in direct mode
}

func (e *Error) Error() string // the C64 name, e.g. "SYNTAX"
```

Each kind maps to the name the C64 uses in its message. New kinds are added as the language grows, always with the C64's own name (tenet *Authentic errors over helpful ones*).

### Error display

A BASIC error is written to stderr as:

```
?<NAME>  ERROR
```

with **two spaces** between the name and `ERROR`, followed by a newline, exactly as a C64 prints it in direct mode — for example `?SYNTAX  ERROR` and `?STRING TOO LONG  ERROR`. An error in a running program (one with `HasLine` set) adds ` IN ` and the line number before the newline, as the C64 ROM does (`$A469`, `$BDC2`): `?SYNTAX  ERROR IN 20`. A `Break` is written without `?` and `  ERROR`, as the C64 prints it: `BREAK` or `BREAK IN 20`.

A C64 moves to a new line before printing an error. The shell does the same: before writing the error to stderr, it calls the interpreter's `FreshLine`, which writes a newline to **stdout** if program output left the line unfinished (for example after `PRINT "A";`, on this line or an earlier one). This keeps the error on its own line in a terminal, and keeps redirected stdout ending in a complete line.

### Exit statuses

| Status | Meaning |
|---|---|
| 0 | Success; or interactive session ended by end of input |
| 1 | Script mode stopped by a BASIC error, or program output could not be written (other than to a closed pipe, which ends the process through SIGPIPE) |
| 2 | Usage error; `FILE` could not be opened or read; stdin could not be read |
| 130 | Script mode stopped by Ctrl-C (`BREAK`) |

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
    Clock       func() time.Time // the interpreter's clock; nil: the system clock
    Terminal    bool             // stdout is a terminal: screen codes become escape codes
    NoColor     bool             // NO_COLOR is set: no colors
  Styled      bool             // stderr is a terminal and colors are on: the shell styles its own text
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
| Error format | `?NAME  ERROR` with two spaces, plus ` IN n` in a running program, to stderr | Single space; to stdout; include a column, or a line number in direct mode | Tenet *Authentic errors over helpful ones*: this is the exact C64 message. stderr per tenet *C64 language, Unix I/O*. |
| Numbered lines | Read by `lexer.LineNumber`, stored with `interp.Store`; no `READY.` after them | Parse every line and let the parser report a line number | A stored line is checked only when it runs, so the shell needs its text, not a parse; this is also how the C64's input loop decides (`$A494`). |
| End of input during `INPUT` or `GET` | Stop with `c64sh: stdin: end of input` and status 1 | Treat it as an empty line; exit quietly | The program cannot get the input it asked for, and treating it as empty can loop forever (`IF A$="" THEN 10`). A message in the style of other stdin problems says what happened. |
| Program never run by a script | Run when input ends | Leave it unrun; require `RUN` | See HLD *Scripts and program mode*. |
| Catching Ctrl-C | Only while `Exec` runs, by `signal.Notify` around each call | For the whole session; never (the default kills c64sh) | Stopping a running program with `BREAK` is the C64 behavior, and keeps an interactive session (and its variables and program) alive. Catching it only during execution leaves Ctrl-C's usual meaning everywhere else: discarding a typed line, or ending a script that is waiting for input. |
| Exit status after `BREAK` | 130 | 1, like other errors | 130 is what a Unix shell reports for a command ended by Ctrl-C, so callers can tell an interrupted script from a failed one. |
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
| Shell styling colors | The terminal's 16 theme colors: cyan typing, green `READY.`, red errors | The C64's own colors; bold and faint instead of colors | Theme colors are readable on light and dark terminals, and set the shell's text apart from programs' C64 colors. Red for errors and green for ready are near-universal terminal conventions. |
| Styling condition | stderr a terminal and `NO_COLOR` unset | stdout a terminal; always in interactive mode | The styled text goes to stderr, so it is stderr that must be a terminal; a redirected stderr gets plain text. |
| Typed-input color | The line editor's prompt holds the style | Fork or wrap `term.Terminal` to color its echo | `term.Terminal` skips escape codes when measuring the prompt, so a zero-width prompt colors the line with no change to the editor. |
| Test seam | `Run(Config, …)` beside `Main` | Inject a terminal-detection function; a pseudo-terminal in tests | Keeps `Main` simple and makes interactive tests plain string-in, string-out. |

## Open Questions & Future Decisions

### Deferred
1. Interactive line editing (history, arrow keys) is out of scope for now (HLD *Non-Goals*).
2. How user-guide examples are kept in sync with functional tests (hand-copied cases or extraction from the guide) is decided when the user guide is written.

## References

- `docs/intent/lexer/lexer-design.md`, `docs/intent/parser/parser-design.md`, `docs/intent/interp/interp-design.md`
- *Commodore 64 Programmer's Reference Guide*, appendix on error messages
