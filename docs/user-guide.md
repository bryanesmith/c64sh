# c64sh User Guide

c64sh is a shell that speaks Commodore 64 BASIC V2. It runs in an ordinary terminal: interactively, like `bash` or `zsh`, or as the interpreter for BASIC script files.

c64sh is being built one command at a time. This guide describes what works today.

- [Installing](#installing)
- [Interactive sessions](#interactive-sessions)
- [Scripts](#scripts)
- [PRINT](#print)
- [Errors](#errors)
- [Exit status](#exit-status)
- [Differences from a real C64](#differences-from-a-real-c64)
- [Not yet supported](#not-yet-supported)

## Installing

From a copy of the repository:

```sh
make install
```

This builds c64sh and copies it to `~/bin/c64sh`. To install somewhere else:

```sh
make install INSTALL_DIR=/usr/local/bin
```

The install directory must be on your `PATH` for the `c64sh` command and `#!/usr/bin/env c64sh` scripts to work. If `~/bin` is not already there, add this line to your `~/.zshrc` or `~/.bashrc`:

```sh
export PATH="$HOME/bin:$PATH"
```

To try c64sh without installing it, `make run` builds it and starts a session.

## Interactive sessions

Run `c64sh` with no arguments in a terminal:

```
$ c64sh

    **** C64SH BASIC V2 ****

READY.
PRINT "HELLO"
HELLO
READY.
```

Each line you type runs as soon as you press Return; this is the C64's *direct mode*. `READY.` appears after each line, as on a C64. Pressing Return on an empty line does nothing.

End the session with **Ctrl-D** at the start of a line.

The banner and `READY.` are written to stderr, and program output to stdout, so `c64sh > out.txt` saves only what your BASIC lines print.

## Scripts

A script is a text file of BASIC lines. c64sh runs them in order, each in direct mode, as if you had typed them:

```sh
c64sh hello.bas
```

To make a script executable, start it with a `#!` line and mark it executable:

```
#!/usr/bin/env c64sh
PRINT "HELLO FROM A SCRIPT"
```

```sh
chmod +x hello.bas
./hello.bas
```

c64sh also reads a script from piped input:

```sh
echo 'PRINT "HI"' | c64sh
```

In a script:

- The first line is skipped if it starts with `#!`.
- Blank lines are skipped.
- c64sh **stops at the first error**, printing the error and exiting with status 1. Lines after it do not run.
- No banner or `READY.` is printed.

## PRINT

`PRINT` writes text to stdout. `?` is a shortcut for `PRINT`, as on a C64.

| You type | Output |
|---|---|
| `PRINT "HELLO"` | `HELLO` |
| `PRINT"HELLO"` | `HELLO` (the space after `PRINT` is optional) |
| `?"HELLO"` | `HELLO` |
| `PRINT` | a blank line |
| `PRINT "A";"B"` | `AB` |
| `PRINT "A"+"B"` | `AB` |
| `PRINT "A""B"` | `AB` |
| `PRINT "A","B"` | `A`, a tab, then `B` |
| `PRINT "FOO":PRINT "BAR"` | `FOO`, then `BAR` on the next line |

**Separators.** `;` joins items with nothing between them. `,` puts a tab between them. `+` joins two strings into one.

**Staying on the same line.** If a `PRINT` ends with `;` or `,`, no newline is printed, so the next `PRINT` continues the same line:

```
PRINT "HELLO, ";
PRINT "WORLD"
```

prints `HELLO, WORLD`.

**Several statements on one line.** Separate them with `:`. Empty statements are allowed, so `PRINT "A"::PRINT "B"` works.

**Strings.** A string is text between double quotes. Everything inside is printed exactly as written, including spaces and punctuation. The closing quote may be left off at the end of a line: `PRINT "HI` prints `HI`. A string cannot contain a double quote.

A string made by joining with `+` may hold at most 255 characters; longer results cause `?STRING TOO LONG  ERROR`.

**Keywords are uppercase.** `PRINT` must be typed in capitals; `print` is a syntax error. Text inside quotes can use any case.

## Errors

Errors are reported the way a C64 reports them, on stderr:

```
?SYNTAX  ERROR
```

| Error | Cause |
|---|---|
| `?SYNTAX  ERROR` | c64sh cannot understand the line: a misspelled or lowercase keyword, a stray character, or a part of BASIC that c64sh does not support yet. |
| `?STRING TOO LONG  ERROR` | Joining strings with `+` produced more than 255 characters. |

An error stops the rest of its line. Anything printed before the error stays printed, as on a C64:

```
PRINT "A":PRINT "B"@
```

prints `A` and `B`, then `?SYNTAX  ERROR`.

Errors always start on a new line, even if the output before them ended with `;`.

## Exit status

| Status | Meaning |
|---|---|
| 0 | Success, or an interactive session ended with Ctrl-D |
| 1 | A script stopped at a BASIC error, or output could not be written |
| 2 | c64sh was run incorrectly: an unknown option, more than one file, or a file that cannot be read |

For example, `c64sh build.bas && echo done` prints `done` only if the script ran without errors.

## Differences from a real C64

- **`,` prints a tab** rather than moving to the next 10-column print zone.
- **Errors go to stderr** and set a non-zero exit status in scripts.
- **No screen emulation**: no 40-column wrapping, colors, or graphics characters.
- **The banner** reads `C64SH BASIC V2`.

## Not yet supported

These are valid C64 BASIC but currently give `?SYNTAX  ERROR`:

- Numbers and math (`PRINT 1+2`)
- Variables (`A$="HI"`)
- Functions such as `CHR$(34)`
- Program mode: lines with line numbers (`10 PRINT "HELLO"`), `RUN`, `LIST`, `GOTO`
- All other commands

The full syntax c64sh accepts is written down in [`grammar/c64basic.ebnf`](../grammar/c64basic.ebnf).
