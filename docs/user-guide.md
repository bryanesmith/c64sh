# c64sh User Guide

c64sh is a shell that speaks Commodore 64 BASIC V2. It runs in an ordinary terminal: interactively, like `bash` or `zsh`, or as the interpreter for BASIC script files.

c64sh is being built one command at a time. This guide describes what works today.

For hands-on examples, see the numbered scripts in [`examples/`](../examples/). Each one shows a feature in many forms, with a comment beside every `PRINT` saying what it prints, and each can be run directly: `./examples/001-hello-world.bas`.

- [Installing](#installing)
- [Interactive sessions](#interactive-sessions)
- [Scripts](#scripts)
- [PRINT](#print)
- [Numbers](#numbers)
- [Arithmetic](#arithmetic)
- [Variables](#variables)
- [Comments](#comments)
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

### Editing and history

While typing a line you can edit it and recall earlier lines:

| Key | Does |
|---|---|
| Up / Down (or Ctrl-P / Ctrl-N) | Recall earlier / later lines from this session |
| Left / Right | Move within the line |
| Home / End (or Ctrl-A / Ctrl-E) | Jump to the start / end of the line |
| Backspace / Delete | Delete a character |
| Ctrl-U / Ctrl-K / Ctrl-W | Delete to the start of the line / to its end / the previous word |
| Ctrl-L | Clear the screen |
| Ctrl-C | Discard the line you are typing |
| Ctrl-D | On an empty line, end the session |

History holds the last 100 lines you ran in the session (blank lines are skipped) and is not saved when c64sh exits. Editing is available when c64sh runs in a terminal; scripts and piped input are read as plain lines.

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
| `PRINT "A","B"` | `A`, then `B` at column 10 (`A         B`) |
| `PRINT "FOO":PRINT "BAR"` | `FOO`, then `BAR` on the next line |

**Separators.** `;` joins items with nothing between them. `+` joins two strings into one. `,` moves to the next **print zone**: zones start every 10 columns (0, 10, 20, …), and c64sh fills the gap with spaces, so commas line up columns:

```
PRINT "NAME","SCORE"
PRINT "ALICE","12"
```

prints

```
NAME      SCORE
ALICE     12
```

If the cursor is already at the start of a zone, `,` moves a whole zone: `PRINT "0123456789","X"` puts `X` at column 20. The column carries over from one `PRINT` to the next when a line is left open with `;` or `,`, so later output still lines up with the zones.

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

## Numbers

Numbers can be written as whole numbers (`45`), with a decimal point (`3.14`, `.5`, `5.`), or with `E` for "times ten to the power of" (`1E3` is 1000, `1.5E-3` is .0015). A lone `.` is zero.

`PRINT` shows a number the way a C64 does:

- **A space before it**, where a minus sign would go, **and a space after it**: `PRINT 45` prints ` 45 `, and `PRINT "5*9=";45` prints `5*9= 45 `.
- **Rounded to 9 significant digits**: `PRINT 3.14159265358979` prints ` 3.14159265 `.
- **No zero before the decimal point, and no trailing zeros**: `PRINT 0.5` prints ` .5 `, and `PRINT 2.50` prints ` 2.5 `.
- **In scientific notation** from 1E9 up and below .01: `PRINT 1000000000` prints ` 1E+09 `, and `PRINT .001` prints ` 1E-03 `.

Because each number brings its own spaces, `PRINT 1;2;3` prints ` 1  2  3 `.

A few more rules come straight from the C64:

- **Spaces inside a number are ignored**: `PRINT 1 000 000` prints ` 1000000 `, and `PRINT 1 2` prints ` 12 `.
- **A second decimal point starts a new number**: `PRINT 1.2.3` prints ` 1.2  .3 `.
- **`+` adds numbers** as well as joining strings, but it cannot join a number and a string: `PRINT "AGE: "+42` is a `?TYPE MISMATCH  ERROR`. See [Arithmetic](#arithmetic).
- **Numbers range up to 1.70141183E+38.** A larger number is an `?OVERFLOW  ERROR`; a number too small for a C64 (below about 2.9E-39) becomes 0.

See [`examples/006-numbers.bas`](../examples/006-numbers.bas) for every form in one script.

## Arithmetic

| Operator | Meaning | Example | Prints |
|---|---|---|---|
| `+` | add (or join strings) | `PRINT 2+3` | ` 5 ` |
| `-` | subtract | `PRINT 10-4` | ` 6 ` |
| `*` | multiply | `PRINT 5*9` | ` 45 ` |
| `/` | divide | `PRINT 7/2` | ` 3.5 ` |
| `^` (or `↑`) | power | `PRINT 2^3` | ` 8 ` |
| `-` in front | negate | `PRINT -5` | `-5 ` |
| `( )` | group | `PRINT (2+3)*4` | ` 20 ` |

**Order of operations**, as on a C64:

1. Parentheses
2. `^`, from left to right
3. A minus sign in front of a number (negation)
4. `*` and `/`, from left to right
5. `+` and `-`, from left to right

So `PRINT 2+3*4` prints ` 14 `, `PRINT 10-4-3` prints ` 3 `, and `PRINT -2*3` prints `-6 `. Because `^` comes before negation, `PRINT -2^2` prints `-4 `; and because `^` works left to right, `PRINT 2^3^2` prints ` 64 ` (that is, `(2^3)^2`).

A few details:

- **Signs can repeat**: `PRINT --5` prints ` 5 `, and `PRINT 5--5` prints ` 10 `. A `+` in front changes nothing.
- **Division keeps 9 significant digits**: `PRINT 1/3` prints ` .333333333 `.
- **Exponents**: `^` is the C64's up-arrow key (the same character code), and `↑` also works. A fractional exponent takes a root (`PRINT 9^.5` prints ` 3 `), and a negative one divides (`PRINT 2^-1` prints ` .5 `). As on a C64, `0^0` is 1 and `0` to a negative power is 0. A negative number to a fractional power is an `?ILLEGAL QUANTITY  ERROR`.
- **Only `+` works on strings.** `-`, `*`, `/`, `^`, or a minus sign in front of a string is a `?TYPE MISMATCH  ERROR`.
- **Dividing by zero** is a `?DIVISION BY ZERO  ERROR`.
- **A `(` right after a number starts a new item**: `PRINT 2(3)` prints ` 2  3 `. But `-` and `+` always continue the calculation: `PRINT 1 -1` prints ` 0 `.

See [`examples/007-arithmetic.bas`](../examples/007-arithmetic.bas) and [`examples/008-exponents.bas`](../examples/008-exponents.bas) for every form.

## Variables

A variable holds a value for later. Set it with `=` (the word `LET` in front is optional), and use its name anywhere a value can go:

```
A=5
LET B=A*2+1
N$="ALICE"
PRINT "B IS";B
PRINT "HELLO, ";N$
```

prints `B IS 11 ` and `HELLO, ALICE`.

- **Two kinds**: a name ending in `$` holds a string (`N$`); any other name holds a number (`A`, `SCORE`). `A` and `A$` are different variables. Putting a string in a number variable, or a number in a string variable, is a `?TYPE MISMATCH  ERROR`.
- **Names** start with an uppercase letter, followed by letters and digits. **Only the first two characters count**, so `SCORE` and `SC` are the same variable.
- **A name cannot contain a keyword**, because the C64 finds keywords anywhere: `PREMIUM=1` is a `?SYNTAX  ERROR` (it contains `REM`). It also means `PRINTER` prints the variable `ER`, and `LETTER=1` sets `TE`.
- **Spaces inside a name are ignored**: `A B` is the variable `AB`.
- **A variable never set** is 0, or the empty string.
- **Values last** from line to line, for the whole session or script.

See [`examples/009-variables.bas`](../examples/009-variables.bas) for every form in one script.

## Comments

`REM` starts a comment. Everything after it, to the end of the line, is ignored:

```
#!/usr/bin/env c64sh
REM GREET THE USER
PRINT "HELLO":REM SAYS HELLO
```

prints `HELLO`.

- **A comment runs to the end of the line**, even past a colon: `REM A:PRINT "X"` prints nothing.
- **A comment can follow other statements**, after a colon: `PRINT "A":REM SHOW A` prints `A`.
- **Inside `PRINT`, put a colon before `REM`.** `PRINT "A" REM NOTE` prints `A` and then `?SYNTAX  ERROR`, as on a C64, because `PRINT` only ends at a colon or the end of the line.
- **`REM` must be uppercase**, like every keyword, and needs no space after it: `REMARK` is a comment.
- **Inside quotes, `REM` is just text:** `PRINT "REM"` prints `REM`.
- In an interactive session, a comment line is followed by `READY.`, like any other command.

## Errors

Errors are reported the way a C64 reports them, on stderr:

```
?SYNTAX  ERROR
```

| Error | Cause |
|---|---|
| `?SYNTAX  ERROR` | c64sh cannot understand the line: a misspelled or lowercase keyword, a stray character, or a part of BASIC that c64sh does not support yet. |
| `?STRING TOO LONG  ERROR` | Joining strings with `+` produced more than 255 characters. |
| `?TYPE MISMATCH  ERROR` | `+` was given a string and a number, or `-`, `*`, `/`, `^`, or a minus sign in front was given a string. |
| `?OVERFLOW  ERROR` | A number, or the result of a calculation, is larger than 1.70141183E+38. |
| `?DIVISION BY ZERO  ERROR` | Dividing by zero. |
| `?ILLEGAL QUANTITY  ERROR` | A negative number raised to a fractional power, such as `(-8)^(1/3)`. |

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

- **`,` fills print zones with spaces.** A C64 moves its cursor right on screen instead, leaving whatever was there; in a terminal, output only ever appears after the cursor, so spaces look the same.
- **Errors go to stderr** and set a non-zero exit status in scripts.
- **No screen emulation**: no 40-column wrapping, colors, or graphics characters. So a print zone past column 40 stays on the same line: `PRINT 2,3,4,5,6` prints ` 6 ` at column 40, where a C64 would start a new screen line.
- **Arithmetic uses standard 64-bit floating point**, rounded to the C64's 9 digits when printed. Results match a C64 in nearly every case; a C64's own rounding occasionally differs in the last digit.
- **The banner** reads `C64SH BASIC V2`.

## Not yet supported

These are valid C64 BASIC but currently give `?SYNTAX  ERROR`:

- Comparisons and logic (`=`, `<`, `>`, `AND`, `OR`, `NOT`)
- Integer variables (`A%`), arrays (`DIM A(10)`), and the system variables `TI`, `TI$`, and `ST`
- Functions such as `CHR$(34)`
- Program mode: lines with line numbers (`10 PRINT "HELLO"`), `RUN`, `LIST`, `GOTO`
- All other commands

Every form c64sh accepts is described in this guide, and shown in use in [`examples/`](../examples/).
