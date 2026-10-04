# Shell Specs

Design: `shell-design.md`

## Command line

- [x] **SHELL-CLI-001**: `cmd/c64sh` shall call `shell.Main` with the command-line arguments after the program name, `os.Stdin`, `os.Stdout`, and `os.Stderr`, and exit with the status it returns.
- [x] **SHELL-CLI-002**: When c64sh is invoked with `-h` or `--help`, it shall write usage text to stdout and exit with status 0.
- [x] **SHELL-CLI-003**: If c64sh is invoked with an argument beginning with `-` other than `-h` or `--help`, then it shall write `c64sh: `, a message naming the argument, and the usage text to stderr, and exit with status 2.
- [x] **SHELL-CLI-004**: If c64sh is invoked with more than one non-option argument, then it shall write `c64sh: `, a message, and the usage text to stderr, and exit with status 2.
- [x] **SHELL-CLI-005**: If the `FILE` argument cannot be opened or read (it does not exist, is a directory, is not readable, or a read fails partway), then c64sh shall write `c64sh: FILE: <reason>` to stderr and exit with status 2, keeping any output already produced by earlier lines.
- [x] **SHELL-CLI-006**: If reading stdin fails with an I/O error, then c64sh shall write `c64sh: stdin: <reason>` to stderr and exit with status 2.

## Mode selection

- [x] **SHELL-MODE-001**: When c64sh is invoked with no `FILE` argument and stdin is a terminal (an `*os.File` for which `golang.org/x/term.IsTerminal` reports true; `/dev/null`, pipes, and regular files are not terminals), it shall run in interactive mode.
- [x] **SHELL-MODE-002**: When c64sh is invoked with a `FILE` argument, or stdin is not a terminal, it shall run in script mode.
- [x] **SHELL-MODE-003**: When `shell.Run` is called, it shall use interactive-mode behavior if `Config.Interactive` is true and script-mode behavior otherwise, without detecting whether stdin is a terminal, and shall read input from `Config.File` if it is non-empty or else from stdin, independently of `Config.Interactive`.
- [x] **SHELL-MODE-004**: In both interactive and script mode, the shell shall store each non-blank line that begins with a line number in the program (SHELL-PROG-001), and execute every other non-blank line in direct mode, immediately.

## Interactive mode

- [x] **SHELL-INT-001**: When interactive mode starts, the shell shall write to stderr a blank line, `    **** C64SH BASIC V2 ****`, a blank line, and `READY.`, each followed by a newline.
- [x] **SHELL-INT-002**: In interactive mode, after each non-blank line has run, whether it succeeded or reported a BASIC error, the shell shall write `READY.` followed by a newline to stderr, except after a line stored in the program.
- [x] **SHELL-INT-003**: In interactive mode, before writing `READY.`, the shell shall call the interpreter's `FreshLine`, so that stdout gains a newline when program output left the line unfinished.
- [x] **SHELL-INT-004**: In interactive mode, when a blank line is entered, the shell shall execute nothing and write nothing.
- [x] **SHELL-INT-005**: In interactive mode, when a line reports a BASIC error, the shell shall continue reading the next line.
- [x] **SHELL-INT-006**: In interactive mode, when the end of input is reached, the shell shall write a newline to stderr and exit with status 0.

## Line editing

- [x] **SHELL-EDIT-001**: When interactive mode runs with both stdin and stderr connected to terminals, the shell shall read each line with the `golang.org/x/term` line editor, echoing keystrokes to stderr.
- [x] **SHELL-EDIT-002**: If stdin or stderr is not a terminal, or the shell is in script mode, then the shell shall read lines without the line editor, as SHELL-LINE-001 to SHELL-LINE-003 specify.
- [x] **SHELL-EDIT-003**: While a line is read with the line editor, the up arrow (or Ctrl-P) shall replace the line with the previous entry of the session's history, and the down arrow (or Ctrl-N) with the next, more recent entry, returning to the line being typed after the most recent entry.
- [x] **SHELL-EDIT-004**: The shell shall add each line read with the line editor to the session's history, except blank lines and lines discarded with Ctrl-C, keeping the most recent entries up to the history size (SHELL-SET-002), including any loaded from the history file (SHELL-HIST-002); with a history size of 0, it shall keep none and neither read nor write the history file.
- [x] **SHELL-EDIT-005**: When Ctrl-C is pressed while a line is read with the line editor, the shell shall discard that line without running it or adding it to history, and read a new line.
- [x] **SHELL-EDIT-006**: When Ctrl-D is pressed on an empty line read with the line editor, the shell shall treat it as the end of input (SHELL-INT-006).
- [x] **SHELL-EDIT-007**: The shell shall switch stdin to raw mode immediately before reading each line with the line editor and restore stdin's previous mode after the line is read, whether or not reading succeeded, so that each line runs with the terminal in its previous mode.
- [x] **SHELL-EDIT-008**: When the line editor reports a line as pasted (`term.ErrPasteIndicator`), the shell shall run it as an ordinary line.
- [x] **SHELL-EDIT-009**: While a line is read with the line editor, the left and right arrows, Home, End, Backspace, and Delete shall move the cursor and edit the line, so that the line returned is the edited text.

## History file

- [x] **SHELL-HIST-001**: When the shell reads its settings, it shall take the history file from the setting `C64SH_HISTORY` if that variable is set (so an empty value means no history file), and otherwise `.c64sh_history` in `Config.Home`, or no history file if `Config.Home` is empty.
- [x] **SHELL-HIST-002**: When the line editor is used and `Config.HistoryFile` is not empty, the shell shall start the session's history with the most recent non-blank lines of that file, up to the history size (SHELL-SET-002) (oldest first, one line per line of the file, with any trailing `\r` removed), so that the up arrow recalls lines from earlier sessions; a missing file shall give an empty history.
- [x] **SHELL-HIST-003**: When a line is added to the history of a line editor with a history file, the shell shall write the whole history, oldest first, each line followed by `\n`, to a new temporary file in the history file's directory with permissions `0600`, and rename it over the history file.
- [x] **SHELL-HIST-004**: If the history file exists but cannot be read, or cannot be written, then the shell shall write `c64sh: history: ` followed by the reason to stderr, at most once per session, and continue with the history in memory.
- [x] **SHELL-HIST-005**: When lines are read without the line editor (script mode, pipes, or input that is not a terminal), the shell shall neither read nor write the history file.

## Script mode

- [x] **SHELL-SCRIPT-001**: In script mode, the shell shall write no banner and no `READY.`.
- [x] **SHELL-SCRIPT-002**: In script mode, if the first line of input begins with `#!`, the shell shall skip that line.
- [x] **SHELL-SCRIPT-003**: In script mode, the shell shall process a line beginning with `#!` that is not the first line as ordinary input.
- [x] **SHELL-SCRIPT-004**: In script mode, the shell shall skip blank lines.
- [x] **SHELL-SCRIPT-005**: In script mode, if a line reports a BASIC error, the shell shall run no later lines and exit with status 1, or with status 130 if the error is a BREAK.
- [x] **SHELL-SCRIPT-006**: In script mode, when every line has run without a BASIC error, the shell shall exit with status 0.
- [x] **SHELL-SCRIPT-007**: When an executable file whose first line is `#!/usr/bin/env c64sh` is run and `c64sh` is on `PATH`, the file's remaining lines shall run in script mode.
- [x] **SHELL-SCRIPT-008**: In script mode, when input ends and any program run under SHELL-SCRIPT-009 has finished, the shell shall write nothing further, even if the output written to stdout does not end with a newline.
- [x] **SHELL-SCRIPT-009**: In script mode, when input ends without a BASIC error and the interpreter's `NeverRun` reports true, the shell shall run the line `RUN` as if it followed the last line, reporting any BASIC error from it and exiting as SHELL-SCRIPT-005 specifies if there is one.

## Program lines

- [x] **SHELL-PROG-001**: When `lexer.LineNumber` finds a line number at the start of a non-blank line, the shell shall call the interpreter's `Store` with that number and the rest of the line, and shall not otherwise execute the line.
- [x] **SHELL-PROG-002**: If `lexer.LineNumber` returns an error for a line, then the shell shall report it as a BASIC error, store nothing, and continue as for any other line reporting an error (SHELL-INT-005, SHELL-SCRIPT-005).

## Interrupts

- [x] **SHELL-BREAK-001**: While the interpreter executes a line, when the process receives SIGINT (Ctrl-C), the shell shall call the interpreter's `Interrupt`.
- [x] **SHELL-BREAK-002**: While the shell is not executing a line, it shall leave SIGINT's handling as it was when the shell started (by default, terminating the process).

## Line handling

- [x] **SHELL-LINE-001**: For both interactive and script input, the shell shall remove a line's trailing `\n` and, if then present, a trailing `\r` before processing it.
- [x] **SHELL-LINE-002**: For both interactive and script input, the shell shall process a final line that has no line terminator.
- [x] **SHELL-LINE-003**: The shell shall read input lines of any length without error.
- [x] **SHELL-LINE-004**: The shell shall treat a line that is empty or consists only of spaces and tabs as blank.
- [x] **SHELL-LINE-005**: When a line contains a syntax error, the shell shall execute the statements the parser returned, in which the error is a `BadItem` or `BadStmt` at the point where parsing failed, and report the error only if execution reaches it (so `PRINT "A":PRINT "B"@` prints `A` and `B` and reports `?SYNTAX  ERROR`, while `IF 0 THEN PRINT "A"@` reports nothing).
- [x] **SHELL-LINE-006**: If executing a line's statements returns a BASIC error, the shell shall report that error and not report a syntax error from the same line.
- [x] **SHELL-LINE-007**: If writing program output to stdout fails for a reason other than a closed pipe (such as a full disk), then the shell shall stop, write nothing further, and exit with status 1.
- [x] **SHELL-LINE-008**: If stdout is a pipe whose reader has exited, then c64sh shall be terminated silently by Go's default SIGPIPE handling on its next write to stdout, writing no error message.

## Error display

- [x] **SHELL-ERR-001**: When the shell reports a BASIC error other than BREAK, it shall write to stderr `?`, the error's C64 name, two spaces, `ERROR`, then, if the error has `HasLine` set, ` IN ` and its line number, and a newline (such as `?SYNTAX  ERROR`, `?STRING TOO LONG  ERROR`, and `?SYNTAX  ERROR IN 20`).
- [x] **SHELL-ERR-002**: When the shell reports a BASIC error, it shall first call the interpreter's `FreshLine`, so that stdout gains a newline when program output left the line unfinished.
- [x] **SHELL-ERR-003**: `basicerr.Error`'s `Error` method shall return the error kind's C64 name: `SYNTAX` for `Syntax`, `STRING TOO LONG` for `StringTooLong`, `TYPE MISMATCH` for `TypeMismatch`, `OVERFLOW` for `Overflow`, `DIVISION BY ZERO` for `DivisionByZero`, `ILLEGAL QUANTITY` for `IllegalQuantity`, `UNDEF'D STATEMENT` for `UndefdStatement`, `BREAK` for `Break`, `NEXT WITHOUT FOR` for `NextWithoutFor`, `OUT OF MEMORY` for `OutOfMemory`, `RETURN WITHOUT GOSUB` for `ReturnWithoutGosub`, `ILLEGAL DIRECT` for `IllegalDirect`, `UNDEF'D FUNCTION` for `UndefdFunction`, `FILE NOT FOUND` for `FileNotFound`, `DEVICE NOT PRESENT` for `DeviceNotPresent`, `ILLEGAL DEVICE NUMBER` for `IllegalDeviceNumber`, `MISSING FILE NAME` for `MissingFileName`, `LOAD` for `Load`, `VERIFY` for `Verify`, `FILE OPEN` for `FileOpen`, `FILE NOT OPEN` for `FileNotOpen`, `NOT INPUT FILE` for `NotInputFile`, `NOT OUTPUT FILE` for `NotOutputFile`, `TOO MANY FILES` for `TooManyFiles`, `FILE DATA` for `FileData`, `BAD SUBSCRIPT` for `BadSubscript`, `REDIM'D ARRAY` for `RedimdArray`, and `OUT OF DATA` for `OutOfData`.
- [x] **SHELL-ERR-004**: When the shell reports a BREAK error, it shall first call the interpreter's `FreshLine`, then write to stderr `BREAK`, then, if the error has `HasLine` set, ` IN ` and its line number, and a newline (such as `BREAK IN 20`).

## Keyboard input

- [x] **SHELL-KEY-001**: The shell shall set a console on the interpreter that reads stdin, sharing the reader the shell reads lines from when the script is stdin, so that `INPUT` consumes the input lines after the line running.
- [x] **SHELL-KEY-002**: When stdin is not a terminal, the console's `ReadLine` shall return the next line without its `\n` and any `\r`, reported as not echoed, and its `ReadKey` the next character, with `\n` or `\r\n` returned as `CHR$(13)`; both shall return `io.EOF` at the end of input.
- [x] **SHELL-KEY-003**: When stdin is a terminal, the shell shall, while executing each line, switch it to input without line buffering and without echo, leaving signal keys and output processing on, and restore its previous mode afterwards.
- [x] **SHELL-KEY-004**: When stdin is a terminal, the console's `ReadKey` shall return at once, with `""` if no key is waiting; it shall return Return as `CHR$(13)`, Backspace and Delete as `CHR$(20)`, the up, down, right, and left arrows as `CHR$(145)`, `CHR$(17)`, `CHR$(29)`, and `CHR$(157)`, ignore other escape sequences, and return any other character as it is.
- [x] **SHELL-KEY-005**: When stdin is a terminal, the console's `ReadLine` shall read keys until Return, echoing each character to stderr and removing the last one, with `\b \b` on stderr, on Backspace or Delete, report the line as echoed, and return `ErrInterrupted` if `stop` becomes true while it waits.
- [x] **SHELL-KEY-006**: If executing a line returns `interp.ErrEndOfInput`, then the shell shall call `FreshLine`, write `c64sh: stdin: end of input` and a newline to stderr, and exit with status 1.

## Program files

- [x] **SHELL-FILE-001**: The shell shall set storage on the interpreter that reads and writes files by relative name in the current directory, creating files with permissions `0644`, and, when replacing is not allowed, creating the file only if it does not exist.
- [x] **SHELL-FILE-002**: In interactive mode the shell shall set stderr as the interpreter's messages writer; in script mode it shall set none.
- [x] **SHELL-FILE-003**: If executing a line returns an `*interp.StorageError`, then the shell shall write to stderr `c64sh: `, the file, `: `, and, for `fs.ErrExist`, `file exists (use REPLACE to replace it)` with the error's `Replace`, or else the error, then a newline; and then continue as for a BASIC error (SHELL-INT-005, SHELL-SCRIPT-005).
- [x] **SHELL-FILE-004**: When the session ends, the shell shall call the interpreter's `CloseFiles` and report a `*interp.StorageError` it returns as SHELL-FILE-003 specifies, setting exit status 1 if it was 0.

## Clock

- [x] **SHELL-CLOCK-001**: When `Config.Clock` is set, the shell shall set it as the interpreter's clock.

## Environment

- [x] **SHELL-ENV-001**: When `shell.Main` runs, it shall set `Config.Env` to the process's environment and `Config.Home` to the user's home directory (empty if unknown); when `Config.Env` is set, `Run` shall set it as the interpreter's environment.

## Terminal output

- [x] **SHELL-SCREEN-001**: When `shell.Main` runs, it shall set `Config.Terminal` if stdout is a terminal; whenever the shell reads its settings, it shall set the interpreter's screen to `Config.Terminal`, with colors on unless the setting `NO_COLOR` is non-empty.
- [x] **SHELL-SCREEN-002**: When a session ends and `Config.Terminal` is set, the shell shall write `ESC[0m` to stdout.

## Styling

- [x] **SHELL-STYLE-001**: When `shell.Main` runs, it shall set `Config.StderrTerminal` if stderr is a terminal; the shell shall style its own text while `Config.StderrTerminal` is set and the setting `NO_COLOR` is not non-empty.
- [x] **SHELL-STYLE-002**: While the shell styles its own text, it shall write the banner and each `READY.` in the ready style, and BASIC errors, `BREAK` messages, storage failures, the end-of-input message, and errors in the run-commands file in the error style, each as a styled span.
- [x] **SHELL-STYLE-003**: While the shell styles its own text, it shall show typed text in the input style: in the line editor, by setting its prompt to the input style and ending a styled span after each line read, and in the `INPUT` echo at a terminal, writing each typed character as a styled span.
- [x] **SHELL-STYLE-004**: The shell shall end each styled span by writing `ESC[0m` followed by the interpreter's `ScreenState`.
- [x] **SHELL-STYLE-005**: While the shell does not style its own text, it shall write no escape codes to stderr other than the line editor's own, and while it does, it shall write text whose style is empty with no escape codes.

## Settings

- [x] **SHELL-SET-001**: The shell shall read its settings from `Config.Env` (every setting at its default when it is nil) when `Run` starts, and in interactive mode again after the run-commands file has run, the second reading deciding the session's history file, history size, styles, and colors.
- [x] **SHELL-SET-002**: When the shell reads its settings, it shall take the history size from `C64SH_HISTSIZE`, a whole number from 0 written in digits, or 100 if the variable is unset or empty.
- [x] **SHELL-SET-003**: When the shell reads its settings, it shall take the input, ready, and error styles from `C64SH_INPUT_COLOR`, `C64SH_READY_COLOR`, and `C64SH_ERROR_COLOR`, each `ESC[` followed by the value and `m` when the value is one or more groups of digits separated by `;`, no style when the value is empty, and `ESC[36m`, `ESC[32m`, and `ESC[31m` respectively when the variable is unset.
- [x] **SHELL-SET-005**: When the shell reads its settings, it shall set the interpreter's string limit to `C64SH_STRING_LIMIT`, -1 or a whole number from 0 written in digits, or -1 (no limit) if the variable is unset or empty.
- [x] **SHELL-SET-004**: If the reading of the settings that decides them finds `C64SH_HISTSIZE`, `C64SH_STRING_LIMIT`, or a color setting with a value that is not valid, then the shell shall write `c64sh: NAME: invalid value "VALUE"` to stderr for each such variable and use that setting's default.

## Run-commands file

- [x] **SHELL-RC-001**: When an interactive session starts, before the banner, the shell shall run the run-commands file: the file named by `C64SH_RC` in `Config.Env` if it is set, none if its value is empty, and otherwise `.c64shrc` in `Config.Home`, or none if `Config.Home` is empty.
- [x] **SHELL-RC-002**: While running the run-commands file, the shell shall handle each non-blank line as if typed in interactive mode (SHELL-MODE-004), with any trailing `\r` removed, writing no `READY.` after it.
- [x] **SHELL-RC-003**: If a line of the run-commands file reports a BASIC error or a storage failure, then the shell shall write `c64sh: `, the file's path, `:`, the line's number in the file, `: `, and the message it would otherwise write (for a storage failure, without its `c64sh: `) to stderr, run no more of the file, and start the session.
- [x] **SHELL-RC-004**: If the run-commands file does not exist, then the shell shall start the session without writing anything; if it exists but cannot be read, then the shell shall write `c64sh: `, its path, `: `, and the reason to stderr and start the session.
- [x] **SHELL-RC-005**: In script mode, the shell shall not run the run-commands file.
