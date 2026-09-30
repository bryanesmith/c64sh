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
- [x] **SHELL-MODE-004**: In both interactive and script mode, the shell shall execute each non-blank line in direct mode, immediately and independently of other lines.

## Interactive mode

- [x] **SHELL-INT-001**: When interactive mode starts, the shell shall write to stderr a blank line, `    **** C64SH BASIC V2 ****`, a blank line, and `READY.`, each followed by a newline.
- [x] **SHELL-INT-002**: In interactive mode, after each non-blank line has run, whether it succeeded or reported a BASIC error, the shell shall write `READY.` followed by a newline to stderr.
- [x] **SHELL-INT-003**: In interactive mode, before writing `READY.`, the shell shall call the interpreter's `FreshLine`, so that stdout gains a newline when program output left the line unfinished.
- [x] **SHELL-INT-004**: In interactive mode, when a blank line is entered, the shell shall execute nothing and write nothing.
- [x] **SHELL-INT-005**: In interactive mode, when a line reports a BASIC error, the shell shall continue reading the next line.
- [x] **SHELL-INT-006**: In interactive mode, when the end of input is reached, the shell shall write a newline to stderr and exit with status 0.

## Script mode

- [x] **SHELL-SCRIPT-001**: In script mode, the shell shall write no banner and no `READY.`.
- [x] **SHELL-SCRIPT-002**: In script mode, if the first line of input begins with `#!`, the shell shall skip that line.
- [x] **SHELL-SCRIPT-003**: In script mode, the shell shall process a line beginning with `#!` that is not the first line as ordinary input.
- [x] **SHELL-SCRIPT-004**: In script mode, the shell shall skip blank lines.
- [x] **SHELL-SCRIPT-005**: In script mode, if a line reports a BASIC error, the shell shall run no later lines and exit with status 1.
- [x] **SHELL-SCRIPT-006**: In script mode, when every line has run without a BASIC error, the shell shall exit with status 0.
- [x] **SHELL-SCRIPT-007**: When an executable file whose first line is `#!/usr/bin/env c64sh` is run and `c64sh` is on `PATH`, the file's remaining lines shall run in script mode.
- [x] **SHELL-SCRIPT-008**: In script mode, when input ends, the shell shall write nothing further, even if the output written to stdout does not end with a newline.

## Line handling

- [x] **SHELL-LINE-001**: For both interactive and script input, the shell shall remove a line's trailing `\n` and, if then present, a trailing `\r` before processing it.
- [x] **SHELL-LINE-002**: For both interactive and script input, the shell shall process a final line that has no line terminator.
- [x] **SHELL-LINE-003**: The shell shall read input lines of any length without error.
- [x] **SHELL-LINE-004**: The shell shall treat a line that is empty or consists only of spaces and tabs as blank.
- [x] **SHELL-LINE-005**: When a line contains a syntax error, the shell shall execute the statements the parser returned (the statements completed before the error and, if the error is inside a `PRINT`, that `PRINT`'s items before the error), and then report exactly one error: the first BASIC error that execution returned, or else the SYNTAX error.
- [x] **SHELL-LINE-006**: If executing a line's statements returns a BASIC error, the shell shall report that error and not report a syntax error from the same line.
- [x] **SHELL-LINE-007**: If writing program output to stdout fails for a reason other than a closed pipe (such as a full disk), then the shell shall stop, write nothing further, and exit with status 1.
- [x] **SHELL-LINE-008**: If stdout is a pipe whose reader has exited, then c64sh shall be terminated silently by Go's default SIGPIPE handling on its next write to stdout, writing no error message.

## Error display

- [x] **SHELL-ERR-001**: When the shell reports a BASIC error, it shall write to stderr `?`, the error's C64 name, two spaces, `ERROR`, and a newline (such as `?SYNTAX  ERROR` and `?STRING TOO LONG  ERROR`).
- [x] **SHELL-ERR-002**: When the shell reports a BASIC error, it shall first call the interpreter's `FreshLine`, so that stdout gains a newline when program output left the line unfinished.
- [x] **SHELL-ERR-003**: `basicerr.Error`'s `Error` method shall return the error kind's C64 name: `SYNTAX` for `Syntax`, `STRING TOO LONG` for `StringTooLong`, `TYPE MISMATCH` for `TypeMismatch`, `OVERFLOW` for `Overflow`, and `DIVISION BY ZERO` for `DivisionByZero`.
