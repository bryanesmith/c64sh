# c64sh User Guide

c64sh is a shell that speaks Commodore 64 BASIC V2. It runs in an ordinary terminal: interactively, like `bash` or `zsh`, or as the interpreter for BASIC script files.

c64sh is being built one command at a time. This guide describes what works today.

c64sh is Commodore 64 BASIC V2, plus a few **extensions** for working in a modern terminal. Every section about an extension begins with *c64sh extension*, so you know it won't work on a real C64; [Extensions](#extensions) lists them all.

To learn the language by building a game, start with the [tutorial](tutorial/index.md); for the patterns experienced BASIC programmers used, see [idiomatic BASIC](idioms.md). For hands-on examples, see the numbered scripts in [`examples/features/`](../examples/features/). Each one shows a feature in many forms, with a comment beside every `PRINT` saying what it prints, and each can be run directly: `./examples/features/001-hello-world.bas`.

- [Installing](#installing)
- [Interactive sessions](#interactive-sessions)
- [Scripts](#scripts)
- [PRINT](#print)
- [Numbers](#numbers)
- [Colors and the screen](#colors-and-the-screen)
- [Arithmetic](#arithmetic)
- [Number functions](#number-functions)
- [String functions](#string-functions)
- [The clock](#the-clock)
- [Comparisons](#comparisons)
- [Logic](#logic)
- [IF … THEN](#if--then)
- [Variables](#variables)
- [Arrays](#arrays)
- [DATA statements](#data-statements)
- [Free memory](#free-memory)
- [Programs](#programs)
- [Loops](#loops)
- [Subroutines](#subroutines)
- [Computed jumps](#computed-jumps)
- [Keyboard input](#keyboard-input)
- [User-defined functions](#user-defined-functions)
- [Saving programs](#saving-programs)
- [Data files](#data-files)
- [Environment variables](#environment-variables)
- [Comments](#comments)
- [Errors](#errors)
- [Exit status](#exit-status)
- [Extensions](#extensions)
- [Differences from a real C64](#differences-from-a-real-c64)
- [Not yet supported](#not-yet-supported)
- [Not planned](#not-planned)

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

Each line you type runs as soon as you press Return; this is the C64's *direct mode*. `READY.` appears after each line, as on a C64. A line that starts with a number is stored in the program instead (see [Programs](#programs)), with no `READY.` after it. Pressing Return on an empty line does nothing.

End the session with **Ctrl-D** at the start of a line.

### Editing and history

*c64sh extension: a C64 has a full-screen editor instead, with no history.*

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

History holds the last 100 lines you ran (see [Settings](#settings) to change that) (blank lines are skipped), and is **saved between sessions** in `~/.c64sh_history`, so after restarting c64sh the up arrow still recalls earlier commands. The file is plain text, one command per line, readable only by you.

To keep history somewhere else, set `C64SH_HISTORY` to a file path; to turn saving off, set it to an empty value:

```sh
export C64SH_HISTORY=~/.config/c64sh/history   # a different file
export C64SH_HISTORY=                          # don't save history
```

If the file cannot be read or written, c64sh prints one `c64sh: history: …` warning and keeps history for the current session only. Editing is available when c64sh runs in a terminal; scripts and piped input are read as plain lines.

The banner and `READY.` are written to stderr, and program output to stdout, so `c64sh > out.txt` saves only what your BASIC lines print.

### Shell colors

*c64sh extension: a C64 shows everything in one color.*

In a terminal, c64sh colors its own text so you can tell at a glance what you typed, what your program printed, and what went wrong:

| Text | Color |
|---|---|
| What you type (including answers to `INPUT`) | cyan |
| The banner and `READY.` | green |
| Errors and `BREAK` | red |
| What your program prints | your terminal's normal color, or the colors the program sets (see [Colors and the screen](#colors-and-the-screen)) |

These are your terminal theme's own colors, so they suit light and dark themes, and you can choose others (see [Settings](#settings)). A color your program sets stays in effect after `READY.` and your typing, as on a C64. Set `NO_COLOR=1` to turn the shell's colors off (along with programs' colors). When stderr is redirected to a file, the shell writes plain text.

### Startup file: ~/.c64shrc

*c64sh extension: a C64 has no startup file.*

When an interactive session starts, c64sh first runs `~/.c64shrc`, if it exists, as `zsh` runs `~/.zshrc`. It holds ordinary c64sh BASIC, each line handled as if you typed it, so it can set [settings](#settings) with [`ENVIRON`](#environment-variables), define variables, store program lines, or print a greeting. Scripts never run it.

A sample `~/.c64shrc` that sets every [setting](#settings) to its default, so copying it changes nothing until you edit a value:

```
REM ~/.c64shrc: run at the start of every interactive c64sh session
REM Every setting, at its default value
REM How many lines history keeps; 0 keeps none
ENVIRON "C64SH_HISTSIZE=100"
REM Where history is saved; empty: not saved
ENVIRON "C64SH_HISTORY="+ENVIRON$("HOME")+"/.c64sh_history"
REM Colors of typing, READY., and errors: SGR codes, as in GREP_COLORS
REM (36 cyan, 32 green, 31 red, 1;31 bold red); empty: no color
ENVIRON "C64SH_INPUT_COLOR=36"
ENVIRON "C64SH_READY_COLOR=32"
ENVIRON "C64SH_ERROR_COLOR=31"
REM The most characters a string may hold; -1 is unlimited, 255 is a C64's
ENVIRON "C64SH_STRING_LIMIT=-1"
REM Not set by default; any value but empty turns off all colors
REM ENVIRON "NO_COLOR=1"
REM C64SH_RC (this file's location) must be set before c64sh starts
REM Anything else you like, such as a greeting:
REM PRINT "HELLO, ";ENVIRON$("USER")
```

- **Storing a program line clears variables**, as on a C64, so set variables after any program lines.
- **An error stops the file**, naming the file and line, such as `c64sh: /Users/me/.c64shrc:3: ?SYNTAX  ERROR`, and the session starts anyway.
- **Break glass:** `C64SH_RC= c64sh` starts a session without the file; `C64SH_RC=/path/to/file` runs a different one.
- Settings take effect when the file has run; changing one later in a session takes effect in the next session.

### Settings

*c64sh extension.*

c64sh's settings are environment variables. Set them in [`~/.c64shrc`](#startup-file-c64shrc) with `ENVIRON`, or in the environment c64sh starts in (`export C64SH_HISTSIZE=500`).

| Variable | Does | Default |
|---|---|---|
| `NO_COLOR` | Any value but empty turns off all colors: the shell's and programs' | unset |
| `C64SH_HISTORY` | The file history is saved in; empty: history is not saved | `~/.c64sh_history` |
| `C64SH_HISTSIZE` | How many lines history keeps; `0` keeps none (and leaves the file alone) | `100` |
| `C64SH_INPUT_COLOR` | Color of what you type | `36` (cyan) |
| `C64SH_READY_COLOR` | Color of the banner and `READY.` | `32` (green) |
| `C64SH_ERROR_COLOR` | Color of errors and `BREAK` | `31` (red) |
| `C64SH_STRING_LIMIT` | The most characters a string may hold; `-1` is unlimited, `255` is a C64's (see [Strings](#print)) | `-1` |
| `C64SH_RC` | The startup file; empty: none (set it before c64sh starts) | `~/.c64shrc` |

Colors are SGR codes, as in `GREP_COLORS` and `LS_COLORS`, joined with `;`:

| Code | Color | Code | Color |
|---|---|---|---|
| `30`–`37` | black, red, green, yellow, blue, magenta, cyan, white | `90`–`97` | bright versions of the same |
| `1` | bold (`1;31` is bold red) | `38;2;R;G;B` | any 24-bit color (`38;2;108;94;181` is the C64's light blue) |

An empty color, such as `ENVIRON "C64SH_INPUT_COLOR="`, leaves that text uncolored; set all three empty to turn off the shell's colors while programs keep theirs. A value c64sh can't read is reported once (`c64sh: C64SH_HISTSIZE: invalid value "LOTS"`) and the default is used.

## Scripts

A script is a text file of BASIC lines. c64sh handles them in order, each exactly as if you had typed it: a line with a line number is stored in the program (see [Programs](#programs)), and any other line runs at once:

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
- **Ctrl-C** while a line is running stops the script, printing `BREAK` (or `BREAK IN 20` while a program runs) and exiting with status 130.
- No banner or `READY.` is printed.
- **If the script stored numbered lines but never ran them with `RUN`, `GOTO`, or `GOSUB`, the program runs after the last line.** So a script of numbered lines is an ordinary program, with no `RUN` needed.

For example, this script

```
#!/usr/bin/env c64sh
30 PRINT "THREE"
PRINT "FIRST"
10 PRINT "ONE"
20 PRINT "TWO"
```

prints `FIRST` at once, then runs the program at the end: `ONE`, `TWO`, `THREE`. With a `RUN` line at the end it prints the same; a script that runs its program itself is left as written. See [`examples/features/015-numbered-scripts.bas`](../examples/features/015-numbered-scripts.bas).

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

**`TAB(N)`** moves to column `N`, and **`SPC(N)`** moves `N` columns to the right, for lining up columns anywhere:

```
PRINT "NAME";TAB(12);"SCORE"
PRINT "ALICE";TAB(12);120
```

puts both `SCORE` and ` 120 ` at column 12. If the cursor is already at or past column `N`, `TAB` does nothing. `N` is rounded down and must be from 0 to 255. A `PRINT` that ends with `TAB(…)` or `SPC(…)` leaves the line open, like `;`. `TAB(` and `SPC(` are written with no space before the `(`, and work only in `PRINT`. **`POS(0)`** gives the cursor's column (any value can go in the parentheses), so `PRINT "HELLO";POS(0)` prints `HELLO 5 `.

**Staying on the same line.** If a `PRINT` ends with `;` or `,`, no newline is printed, so the next `PRINT` continues the same line:

```
PRINT "HELLO, ";
PRINT "WORLD"
```

prints `HELLO, WORLD`.

**Several statements on one line.** Separate them with `:`. Empty statements are allowed, so `PRINT "A"::PRINT "B"` works.

**Strings.** A string is text between double quotes. Everything inside is printed exactly as written, including spaces and punctuation. The closing quote may be left off at the end of a line: `PRINT "HI` prints `HI`. A string cannot contain a double quote.

On a C64 a string holds at most 255 characters, and longer results cause `?STRING TOO LONG  ERROR`. c64sh's strings are unlimited by default; set [`C64SH_STRING_LIMIT`](#settings) to `255` for the C64's limit, to check that a program will run on one. See [`examples/features/032-string-length.bas`](../examples/features/032-string-length.bas).

**Keywords are uppercase.** `PRINT` must be typed in capitals; `print` is a syntax error. Text inside quotes can use any case.

## Colors and the screen

As on a C64, printing certain characters controls the screen instead of showing a character:

```
PRINT CHR$(147);CHR$(28);"RED ALERT";CHR$(154)
```

clears the screen, then prints `RED ALERT` in red, and sets the color back to the C64's light blue.

| Code | Effect | Code | Effect |
|---|---|---|---|
| `CHR$(147)` | clear the screen | `CHR$(19)` | cursor to the top left |
| `CHR$(17)` / `CHR$(145)` | cursor down / up | `CHR$(29)` / `CHR$(157)` | cursor right / left |
| `CHR$(18)` / `CHR$(146)` | reverse video on / off | `CHR$(13)` | new line |
| `CHR$(144)` black | `CHR$(5)` white | `CHR$(28)` red | `CHR$(159)` cyan |
| `CHR$(156)` purple | `CHR$(30)` green | `CHR$(31)` blue | `CHR$(158)` yellow |
| `CHR$(129)` orange | `CHR$(149)` brown | `CHR$(150)` light red | `CHR$(151)` dark gray |
| `CHR$(152)` gray | `CHR$(153)` light green | `CHR$(154)` light blue | `CHR$(155)` light gray |

- **The colors are the C64's own**, shown exactly in any terminal that supports 24-bit color (most do).
- **A color stays until another is printed**, as on a C64, so set it back when you are done. c64sh resets the terminal's color when it exits.
- **In a file or a pipe**, these codes are left out (cursor right still becomes a space, and Return a new line), so redirected output is plain text.
- **`NO_COLOR=1`** in the environment turns colors off; the other codes still work.
- **Idiom**: keep codes in variables, such as `RE$=CHR$(28)`, so `PRINT RE$;"DANGER"` reads well.

See [`examples/features/030-screen-codes.bas`](../examples/features/030-screen-codes.bas) for every form.

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

See [`examples/features/006-numbers.bas`](../examples/features/006-numbers.bas) for every form in one script.

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

See [`examples/features/007-arithmetic.bas`](../examples/features/007-arithmetic.bas) and [`examples/features/008-exponents.bas`](../examples/features/008-exponents.bas) for every form.

## Number functions

Each takes its argument in parentheses: `PRINT SQR(16)` prints ` 4 `.

| Function | Gives |
|---|---|
| `ABS(X)` | `X` without its sign: `ABS(-7)` is 7 |
| `INT(X)` | `X` rounded down: `INT(3.9)` is 3, and `INT(-3.1)` is -4 |
| `SGN(X)` | -1, 0, or 1, as `X` is negative, zero, or positive |
| `SQR(X)` | the square root |
| `EXP(X)`, `LOG(X)` | e to the power `X`, and the natural logarithm |
| `SIN(X)`, `COS(X)`, `TAN(X)`, `ATN(X)` | trigonometry, in radians |
| `RND(X)` | a pseudo-random number from 0 up to (not including) 1 |

- **`π`** is pi, 3.14159265, typed as the character `π`: `PRINT 2*π`.
- **Rounding to the nearest whole number** is `INT(X+.5)`.
- **`RND`** follows the C64's rules. `RND(1)`, or any positive number, gives the next number of a sequence. A negative number starts a new sequence determined by it, so `X=RND(-42)` at the start of a program makes it give the same numbers every run. `RND(0)` takes a number from the clock. A die roll is `INT(RND(1)*6)+1`.
- Without reseeding, the sequence is the same in every run, as on a C64 just switched on. c64sh's numbers differ from a C64's, though: it uses its own generator.
- **Errors**: a square root or logarithm of a number it cannot take (`SQR(-1)`, `LOG(0)`) is an `?ILLEGAL QUANTITY  ERROR`, and a string argument a `?TYPE MISMATCH  ERROR`.
- **The function names break variable names that contain them**: `POINT` contains `INT`, and `COST` contains `COS`.

See [`examples/features/024-number-functions.bas`](../examples/features/024-number-functions.bas) for every form.

## String functions

| Function | Gives | Example |
|---|---|---|
| `LEN(S$)` | the number of characters | `LEN("HELLO")` is 5 |
| `LEFT$(S$,N)` | the first `N` characters | `LEFT$("HELLO",2)` is `"HE"` |
| `RIGHT$(S$,N)` | the last `N` characters | `RIGHT$("HELLO",3)` is `"LLO"` |
| `MID$(S$,P,N)` | `N` characters from position `P` (counting from 1); without `N`, the rest | `MID$("HELLO",2,3)` is `"ELL"` |
| `CHR$(N)` | the character with code `N` | `CHR$(65)` is `"A"`, `CHR$(34)` is `"` |
| `ASC(S$)` | the code of the first character | `ASC("A")` is 65 |
| `STR$(X)` | a number as a string, as `PRINT` shows it, without the space after | `STR$(5)` is `" 5"` |
| `VAL(S$)` | the number a string starts with, or 0 | `VAL("3.5 CUPS")` is 3.5 |

- **Asking for more characters than there are** gives what there is: `LEFT$("HI",9)` is `"HI"`, and `MID$("HI",5)` is the empty string.
- **Positions and lengths** are rounded down and must be from 0 to 255; `MID$` positions start at 1, so position 0 is an `?ILLEGAL QUANTITY  ERROR`, as is `ASC("")`.
- **Idioms**: loop over a string's characters with `FOR I=1 TO LEN(A$)` and `MID$(A$,I,1)`; check an answer's first letter with `LEFT$(A$,1)="Y"`; put a quote in a string with `CHR$(34)`.
- **Characters** are Unicode characters. Codes 32 to 126 are the same as on a C64; other codes give the Unicode character with that number.
- **The function names break variable names that contain them**: `VALUE` contains `VAL`, and `LENGTH` contains `LEN`.

See [`examples/features/025-string-functions.bas`](../examples/features/025-string-functions.bas) for every form.

## The clock

**`TI`** counts jiffies, sixtieths of a second, since c64sh started, as a C64's clock counts from when it is switched on. **`TI$`** is the same clock as six digits, `HHMMSS`:

```
TI$="000000"
…
PRINT "THAT TOOK";INT(TI/60);"SECONDS"
```

- **Set the clock** by assigning six digits to `TI$`; anything else is an `?ILLEGAL QUANTITY  ERROR`. `TI` itself cannot be assigned (`TI=0` is a `?SYNTAX  ERROR`). Both wrap to 0 after 24 hours.
- **Seeding `RND`**: `X=RND(-TI)` starts a sequence that depends on the clock. Since a script starts the moment it runs, seed after waiting for a key, as C64 games did, so the player's timing makes each game different:

  ```
  10 PRINT "PRESS A KEY"
  20 GET K$:IF K$="" THEN 20
  30 X=RND(-TI)
  ```

See [`examples/features/029-clock.bas`](../examples/features/029-clock.bas) for every form.

## Comparisons

| Operator | Meaning |
|---|---|
| `=` | equal |
| `<>` | not equal |
| `<` / `>` | less / greater |
| `<=` / `>=` | less or equal / greater or equal |

A comparison is a number: **-1 when true, 0 when false**. `PRINT 1<2` prints `-1 `, and `PRINT 5=6` prints ` 0 `.

- **Comparisons come after arithmetic**: `PRINT 1+1=2` prints `-1 `.
- **Strings compare letter by letter**: `PRINT "APPLE"<"BANANA"` prints `-1 `, and a shorter string that starts a longer one comes first (`"A"<"AB"`). Uppercase letters, digits, and punctuation are ordered as on a C64; lowercase and non-English characters may order differently.
- **The symbols can come in either order**, as on a C64: `><` is the same as `<>`, `=<` as `<=`, and `=>` as `>=`, and spaces between them are allowed. All three together, `<=>`, are always true. A repeated symbol, like `==`, is a `?SYNTAX  ERROR`.
- **`=` means two things**: at the start of a statement it assigns, and after that it compares. `A=B=C` stores in `A` the result of comparing `B` with `C`.
- **A string cannot be compared with a number**: `PRINT "1"=1` is a `?TYPE MISMATCH  ERROR`.
- **Testing calculated decimals for equality** can differ from a real C64 in rare cases, because c64sh stores numbers with more precision; `.1+.2=.3`, for example, depends on the last binary digits of each result.

See [`examples/features/011-comparisons.bas`](../examples/features/011-comparisons.bas) for every form.

## Logic

`AND`, `OR`, and `NOT` combine conditions. Because true is -1 and false is 0, they work on comparisons the way you would expect:

```
PRINT 1<2 AND 3<4     : REM -1 (both true)
PRINT 1>2 OR 3<4      : REM -1 (one true)
PRINT NOT 1=2         : REM -1
```

- **Order**: comparisons first, then `NOT`, then `AND`, then `OR`. So `0 OR 1 AND 0` is `0 OR (1 AND 0)`, which is 0, and `NOT 1=2` is `NOT (1=2)`.
- **They work bit by bit** on whole numbers from -32768 to 32767, as on a C64: `PRINT 12 AND 10` prints ` 8 `, `PRINT 12 OR 10` prints ` 14 `, and `PRINT NOT 5` prints `-6 `. Fractions are rounded down first.
- A number outside -32768 to 32767 is an `?ILLEGAL QUANTITY  ERROR`, and a string is a `?TYPE MISMATCH  ERROR`.

See [`examples/features/012-logic.bas`](../examples/features/012-logic.bas) for every form.

## IF … THEN

`IF` runs the rest of its line only when a condition is true:

```
A=5
IF A>3 THEN PRINT "A IS BIG"
IF A>1 AND A<10 THEN PRINT "BETWEEN"
```

prints `A IS BIG` and `BETWEEN`.

- **A false condition skips the whole rest of the line**, including statements after a colon: `IF A<3 THEN PRINT "ONE":PRINT "TWO"` prints nothing. Statements before the `IF` on the same line still run.
- **Any number other than 0 is true**, so `IF -1 THEN …` and `IF .5 THEN …` both run. A string is true when it is not empty: `IF N$ THEN …` runs when `N$` holds something.
- **`IF … THEN 20` jumps to line 20** of the program when the condition is true (see [Programs](#programs)).
- **There is no `ELSE`** in BASIC V2. Use a second `IF` with the opposite test: `IF NOT A>9 THEN …`.
- **`IF`s can follow one another**: `IF A>1 THEN IF A<3 THEN …` runs only when both are true.
- **A skipped part of the line is never checked**, as on a C64: `IF 0 THEN PRINT "X"@` prints nothing and reports no error, because the mistake is never reached.
- **`THEN` is required.** `IF A>3 PRINT "X"` is a `?SYNTAX  ERROR`.

See [`examples/features/013-if-then.bas`](../examples/features/013-if-then.bas) for every form.

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

- **Three kinds**: a name ending in `$` holds a string (`N$`); a name ending in `%` holds a whole number (`C%`); any other name holds a number (`A`, `HEIGHT`). `A`, `A%`, and `A$` are different variables. Putting a string in a number variable, or a number in a string variable, is a `?TYPE MISMATCH  ERROR`.
- **Integer variables** (`%`) round down what is stored in them: `C%=3.7` stores 3, and `C%=-3.7` stores -4. They hold -32768 to 32767; storing a number outside that range is an `?ILLEGAL QUANTITY  ERROR`. In calculations they are ordinary numbers.
- **Names** start with an uppercase letter, followed by letters and digits. **Only the first two characters count**, so `HEIGHT` and `HE` are the same variable.
- **A name cannot contain a keyword**, including keywords c64sh does not support (`PEEKS` contains `PEEK`, `FREE` contains `FRE`), because the C64 finds keywords anywhere: `PREMIUM=1` is a `?SYNTAX  ERROR` (it contains `REM`), and so is `SCORE=1` (it contains `OR`). It also means `PRINTER` prints the variable `ER`, and `LETTER=1` sets `TE`.
- **Spaces inside a name are ignored**: `A B` is the variable `AB`.
- **A variable never set** is 0, or the empty string.
- **Values last** from line to line, for the whole session or script.

See [`examples/features/009-variables.bas`](../examples/features/009-variables.bas) and [`examples/features/010-integer-variables.bas`](../examples/features/010-integer-variables.bas) for every form.

## Arrays

An array holds many values under one name, picked out by a subscript in parentheses:

```
DIM S(4)
FOR I=0 TO 4:S(I)=I*I:NEXT
PRINT S(3)
```

prints ` 9 `.

- **`DIM S(4)` makes five elements**, `S(0)` to `S(4)`, starting at 0 (or the empty string for a string array, `DIM N$(9)`). One `DIM` can make several: `DIM A(10),B$(5)`.
- **Several dimensions** make a grid: `DIM T(3,3)`, then `T(2,1)=5`.
- **An array used without `DIM`** has 10 as the top of each dimension, so `A(5)=1` works by itself, but `A(11)` does not.
- **Arrays are separate from plain variables**: `S` and `S(1)` are unrelated. Integer arrays (`C%(…)`) round down what is stored, like integer variables.
- **Subscripts** are rounded down. A subscript past the top, or the wrong number of subscripts, is a `?BAD SUBSCRIPT  ERROR`; a negative one is an `?ILLEGAL QUANTITY  ERROR`.
- **`DIM` of an array that already exists**, including one made by using it, is a `?REDIM'D ARRAY  ERROR`. Put `DIM` at the start of a program.
- **Memory**: as on a C64, all arrays together must fit in 38911 bytes (5 bytes per number, 3 per string, 2 per integer), or it is an `?OUT OF MEMORY  ERROR`. `DIM A(7000)` fits; `DIM A(8000)` does not.
- **Arrays are cleared** with the variables (`RUN`, `NEW`, changing the program). Elements can be read with `INPUT`, `GET`, and `INPUT#`, but cannot be loop counters.

See [`examples/features/027-arrays.bas`](../examples/features/027-arrays.bas) for every form.

## Free memory

`FRE(0)` gives the memory left for BASIC, counted as a C64 counts it: the program (each line 5 bytes, plus its text with each keyword as one byte), each variable (7 bytes, plus a string's length), and each array.

```
PRINT FRE(0)
```

prints `-26627` with nothing in use. As on a C64, the result is a signed 16-bit number, so free memory above 32767 shows as negative; add 65536 to get the real figure:

```
PRINT FRE(0)-65536*(FRE(0)<0)
```

prints ` 38909 `.

- **The argument is ignored**, whatever it is: `FRE(0)` and `FRE("X")` are the same.
- **The count is close to a C64's, not exact**: a C64 doesn't store a string assigned straight from the program's text (`A$="HI"`), and c64sh counts it.
- **`FRE` only reports.** As c64sh's strings are unlimited by default, a program can use more than a C64 holds; `FRE` then gives 0. Only arrays can run out of memory (see [Arrays](#arrays)).

See [`examples/features/033-free-memory.bas`](../examples/features/033-free-memory.bas) for every form.

## DATA statements

`DATA` lines keep values inside a program, and `READ` takes them, in order:

```
10 READ N$,A
20 PRINT N$;" IS";A
30 DATA ALICE,12
```

prints `ALICE IS 12 `.

- **Items are separated by commas** and read the way `INPUT` reads answers: put an item in quotes to keep a comma, a colon, or leading spaces in it. Without quotes, an item is plain text, even if it looks like a keyword (`DATA PRINT`). An empty item reads as 0 or the empty string.
- **`READ` goes through every `DATA` statement in the program**, in line order, wherever they are; it does not matter where `DATA` lines sit, and running a `DATA` line does nothing.
- **`RESTORE`** starts again from the first item. `RUN` does too.
- **Idioms**: fill an array with a `FOR` loop of `READ`s; or end a table with a marker value (`DATA …,END`) and stop reading when you reach it.
- **Running out** is an `?OUT OF DATA  ERROR`. An item that is not a number, read into a number variable, is a `?SYNTAX  ERROR` naming the `DATA` line, as on a C64.
- `DATA` typed directly is ignored; `READ` typed directly reads the program's `DATA`.

See [`examples/features/028-data-statements.bas`](../examples/features/028-data-statements.bas) for every form.

## Programs

A line that starts with a number is not run: it is **stored** in the program, as on a C64. `RUN` then runs the stored lines in number order:

```
10 PRINT "HELLO"
20 PRINT "WORLD"
RUN
```

prints `HELLO` and `WORLD`.

- **Line numbers** go from 0 to 63999. Lines are kept in number order, whatever order they are typed in, so leaving gaps (10, 20, 30) leaves room to add lines between them later.
- **Typing a line with a number already used replaces that line.** Typing a number alone (`20`) deletes its line.
- **A stored line is checked only when it runs**: `10 PRINT "A"@` is stored without complaint, and the mistake is reported when line 10 runs.
- **Storing or deleting a line clears all variables**, as on a C64.
- In an interactive session, no `READY.` follows a stored line.

| Command | Effect |
|---|---|
| `RUN` | Clears all variables, then runs the program from its first line. |
| `RUN 20` | Clears all variables, then runs the program from line 20. If there is no line 20: `?UNDEF'D STATEMENT  ERROR`. |
| `LIST` | Shows the whole program in number order, after a blank line, each line as its number, a space, and the line as typed. A `?` is shown as `PRINT`, the keyword it stands for. |
| `LIST 20`, `LIST 20-50`, `LIST -50`, `LIST 20-` | Shows only line 20, lines 20 to 50, lines up to 50, or lines from 20. |
| `GOTO 20` | Continues the program at line 20, keeping all variables. Typed directly, it runs the program from line 20. `GO TO 20` is the same. If there is no line 20: `?UNDEF'D STATEMENT  ERROR`. |
| `NEW` | Erases the program and clears all variables. |
| `END` | Stops the program. |
| `STOP` | Stops the program with `BREAK IN` and its line, as the STOP key (Ctrl-C) does. |
| `CONT` | Continues the program after `STOP`, `END`, or Ctrl-C, with its variables as they are now. |
| `CLR` | Clears all variables, keeping the program. |

- **A program stops** after its last line, at `END`, or at an error. `LIST` and `NEW` also stop a program, and `RUN` in a program starts it again from the beginning, with variables cleared.
- **An error in a program names its line**: `?SYNTAX  ERROR IN 20`.
- **`IF … THEN 20` and `IF … GOTO 20`** jump to line 20 when the condition is true, which is how a program loops:

  ```
  10 I=I+1
  20 PRINT I;
  30 IF I<5 THEN 10
  ```

  prints ` 1  2  3  4  5 `.
- **Ctrl-C stops a running program**, as the C64's STOP key does, printing `BREAK IN 20` with the line it stopped in. In an interactive session, `READY.` follows, and the program and variables are kept.
- **Nothing after `RUN`, `GOTO`, `LIST`, `NEW`, or `END` on the same line runs**, in a program or typed directly: `LIST:PRINT "X"` lists the program but does not print `X`.
- **`STOP` and `CONT` are for finding mistakes**, as on a C64: put `STOP` where you want to look, `RUN`, then `PRINT` the variables (or change them), and `CONT` to go on. In a script, `STOP` ends the script with `BREAK IN` and exit status 1.
- **`CONT` can't continue** (`?CAN'T CONTINUE  ERROR`) after an error, after `CLR`, `RUN`, or `NEW`, or after the program is changed, as on a C64. `CONT` inside a program continues at itself, as on a C64, so it loops until you press Ctrl-C.
- **`CLR` in a running program** also forgets its loops and subroutines, so a later `NEXT` or `RETURN` is an error.
- **These keywords break names that contain them**: `FRIEND=1` is a `?SYNTAX  ERROR` (it contains `END`), as are names containing `RUN`, `NEW`, `LIST`, `GOTO`, `GO` (`GOLD`), `TO` (`TOTAL`), `STOP`, `CONT`, or `CLR`.

See [`examples/features/014-program-mode.bas`](../examples/features/014-program-mode.bas), [`examples/features/016-goto.bas`](../examples/features/016-goto.bas), and [`examples/features/034-program-control.bas`](../examples/features/034-program-control.bas) for every form.

## Loops

`FOR` and `NEXT` repeat the statements between them, counting a variable from a start value to an end value:

```
FOR I=1 TO 5:PRINT I;:NEXT
```

prints ` 1  2  3  4  5 `.

- **`STEP`** sets how much the variable changes each time; it is 1 if left out, and can be negative or a fraction: `FOR I=10 TO 0 STEP -5` counts 10, 5, 0.
- **The loop ends when the variable passes the end value.** The test is made at `NEXT`, so the body always runs at least once, even in `FOR I=5 TO 1`. Afterwards the variable is one step past the end: after `FOR I=1 TO 3:NEXT`, `I` is 4.
- **The end and the step are worked out once**, when `FOR` runs; changing a variable used in them does not change the loop.
- **`NEXT` with no variable** continues the innermost loop. `NEXT I` names the loop, and `NEXT J,I` closes two nested loops at once.
- **Loops can be nested** and can span many lines of a program, or fit on one line, in a program or typed directly.
- **The counter must be a number variable**: `FOR I%=…` is a `?SYNTAX  ERROR`, and a string variable is a `?TYPE MISMATCH  ERROR`.
- **`NEXT` with no matching `FOR`** is a `?NEXT WITHOUT FOR  ERROR`. Starting a loop with a variable that is already counting another loop ends the old loop, and any loops inside it.
- **At most 10 loops can be nested**, as on a C64; the 11th is an `?OUT OF MEMORY  ERROR`. Subroutine calls share the same space (see [Subroutines](#subroutines)).
- **The keywords break names that contain them**: `FORM` and `STEPS` cannot be variable names.

See [`examples/features/017-loops.bas`](../examples/features/017-loops.bas) for every form.

## Subroutines

`GOSUB` runs a subroutine, and `RETURN` comes back to the statement just after the `GOSUB`:

```
10 GOSUB 100
20 PRINT "BACK"
30 END
100 PRINT "IN THE SUBROUTINE"
110 RETURN
```

prints `IN THE SUBROUTINE`, then `BACK`. The `END` keeps the program from running on into the subroutine.

- **`RETURN` comes back mid-line**: in `PRINT "A";:GOSUB 100:PRINT "C"`, the `PRINT "C"` runs after the subroutine.
- **Subroutines can call subroutines**, up to 26 deep, as on a C64; deeper is an `?OUT OF MEMORY  ERROR`. Loops use the same space, so fewer calls fit inside nested loops.
- **A subroutine has no parameters or result.** It works on the program's variables, which are shared by the whole program: set a variable before the `GOSUB`, and read the answer from another after it.
- **Typed directly**, `GOSUB 100` runs the subroutine, then the rest of the line you typed. The variables are kept, as with `GOTO`.
- **`RETURN` with no `GOSUB`** to return to is a `?RETURN WITHOUT GOSUB  ERROR`. This is what happens when a program runs on into its subroutines without an `END`.
- **Loops and subroutines**: `RETURN` ends any loops begun inside the subroutine, and a `NEXT` inside a subroutine cannot continue a loop begun outside it (`?NEXT WITHOUT FOR  ERROR`).

See [`examples/features/018-subroutines.bas`](../examples/features/018-subroutines.bas) for every form.

## Computed jumps

`ON` picks a line from a list by number, which is how a BASIC V2 program makes a menu without a chain of `IF`s:

```
10 INPUT "CHOICE (1-3)";C
20 ON C GOTO 100,200,300
30 PRINT "INVALID":GOTO 10
```

goes to line 100 when `C` is 1, 200 when it is 2, and 300 when it is 3.

- **`ON X GOSUB`** calls the line as a subroutine; its `RETURN` comes back after the whole `ON` statement.
- **`X` is rounded down**: `ON 2.9 GOTO …` takes the second line.
- **When `X` is 0, or larger than the list**, nothing happens, and the program goes on with the next statement (line 30 above).
- **`X` must be from 0 to 255**; anything else is an `?ILLEGAL QUANTITY  ERROR`.
- **Only `GOTO` and `GOSUB`** can follow: `ON X GO TO 100` is a `?SYNTAX  ERROR`, as on a C64. The list holds line numbers, not expressions.
- **`ON` breaks names that contain it**: `MONEY`, `ONE`, and `DONE` cannot be variable names.

See [`examples/features/023-computed-jumps.bas`](../examples/features/023-computed-jumps.bas) for every form.

## Keyboard input

A running program reads the keyboard with `INPUT` and `GET`.

**`INPUT`** shows a prompt and `? `, then waits for a line:

```
10 INPUT "WHAT IS YOUR NAME";N$
20 PRINT "HELLO, ";N$
```

shows `WHAT IS YOUR NAME? `, and after you type `ALICE` and Return, prints `HELLO, ALICE`.

- **The prompt** is optional, and must be a string in quotes followed by `;`: `INPUT A` shows just `? `.
- **Several values** are separated by commas: `INPUT A,B` reads `3,4`. If you give too few, `INPUT` asks for the rest with `?? `; if you give too many, it says `?EXTRA IGNORED` and carries on.
- **Numbers**: if a number variable gets something that is not a number, `INPUT` says `?REDO FROM START` and asks again.
- **Strings**: spaces before a value are skipped; put a value in quotes to keep a comma in it: `"PARIS, FRANCE"`. A colon also ends a value, as on a C64.
- **Pressing Return on an empty line** leaves the variables as they were.
- **What you type is kept exactly**, including lowercase letters, so `IF A$="Y"` does not match a typed `y`. (A C64 keyboard types uppercase.)

**`GET`** reads one key without waiting. If no key has been pressed, it gives an empty string, so a program that must wait loops:

```
10 GET K$:IF K$="" THEN 10
20 PRINT "YOU PRESSED ";K$
```

- **Keys** come as C64 characters: Return is `CHR$(13)`, Backspace `CHR$(20)`, and the arrow keys the C64 cursor keys (up 145, down 17, right 29, left 157).
- **`GET A`** with a number variable takes a digit key's value, or 0 for no key; any other key is a `?SYNTAX  ERROR`.
- **Keys pressed while a program runs** wait until `INPUT` or `GET` reads them, like the C64's keyboard buffer, and are not shown.

Both read **stdin**:

- In an interactive session, the keyboard.
- In a script file, stdin, so answers can come from a file: `c64sh quiz.bas < answers.txt`. When stdin is not a terminal, c64sh shows each answer after its prompt, as a C64 screen would.
- In a script piped into c64sh, the lines that follow, which then do not run as commands:

  ```
  10 INPUT A
  20 PRINT A*2
  RUN
  21
  ```

  prints `? 21` and ` 42 `.

If the input ends while `INPUT` or `GET` is waiting, c64sh stops with `c64sh: stdin: end of input` and exit status 1. Typed directly, `INPUT` and `GET` are an `?ILLEGAL DIRECT  ERROR`, as on a C64: they work only in a program. Ctrl-C stops a program waiting for input with `BREAK`.

See [`examples/features/019-keyboard-input.bas`](../examples/features/019-keyboard-input.bas) for every form.

## User-defined functions

`DEF FN` defines a one-line function of one number, which you then use as `FN` and the name, anywhere a value can go:

```
10 DEF FN SQ(X)=X*X
20 PRINT FN SQ(3)+1
```

prints ` 10 `.

- **One parameter, one number back.** The body is a single expression; for anything longer, use a subroutine.
- **The parameter belongs to the function**: calling `FN SQ(4)` does not change the program's own `X`.
- **The body sees the other variables as they are at the call**, and can call other functions.
- **Names** follow the rules for variables: only the first two characters count, and integer names (`FN A%`) are a `?SYNTAX  ERROR`. A function's name is separate from any variable with the same name.
- **`DEF` works only in a program**; typed directly it is an `?ILLEGAL DIRECT  ERROR`. A function can be called typed directly once the program has defined it.
- **The body is checked when the function is called**, not when it is defined, as on a C64, so a mistake in it is reported at the `FN`.
- **Calling a function not yet defined** is an `?UNDEF'D FUNCTION  ERROR`. Definitions are erased with the variables (`RUN`, `NEW`, or changing the program).
- **A function that calls itself** runs out of room: `?OUT OF MEMORY  ERROR`. At most 9 calls can be in progress at once.

See [`examples/features/020-user-functions.bas`](../examples/features/020-user-functions.bas) for every form.

## Saving programs

`SAVE` writes the program to a file, `LOAD` reads it back, and `VERIFY` checks that they match:

```
10 PRINT "HELLO"
SAVE "HELLO"
NEW
LOAD "HELLO"
RUN
```

prints `HELLO`. The file is `HELLO.bas` in the current directory.

- **The file is text**: a `#!/usr/bin/env c64sh` line, then the program's lines as you typed them. So a saved program is also a script: `c64sh HELLO.bas` runs it. `.bas` is added to a name without an extension; `SAVE "NOTES.TXT"` keeps the name as given.
- **`LOAD` typed directly** replaces the program and clears the variables. **In a running program**, `LOAD` replaces the program and runs the new one from the start, keeping the variables, which is how C64 programs chain to their next part.
- **A file to `LOAD`** may only hold a `#!` first line, blank lines, and numbered lines; anything else is a `?LOAD  ERROR`, and the program is unchanged. A missing file is a `?FILE NOT FOUND  ERROR`.
- **`VERIFY "HELLO"`** gives a `?VERIFY  ERROR` if the program and the file differ.
- **Device numbers** work as on a C64, after the name: `SAVE "HELLO",8`. Tape (1, the default) and the disk drives (8 to 11) all mean the current directory. The keyboard (0), the screen (3), and the printers (4, 5) cannot hold programs: `?ILLEGAL DEVICE NUMBER  ERROR`; any other device is `?DEVICE NOT PRESENT  ERROR`. A third number, the secondary address (`LOAD "HELLO",8,1`), is accepted and ignored.
- **Every file needs a name**: `SAVE` alone is a `?MISSING FILE NAME  ERROR`. (On a C64, tape allows no name.)
- **In an interactive session**, typed directly, the C64's messages appear: `SAVING HELLO`, or `SEARCHING FOR HELLO` and `LOADING`, with tape's `PRESS PLAY ON TAPE` (no key needed).

**Replacing a file on disk.** As on the C64's 1541 drive, saving to a disk drive (8 to 11) does not replace an existing file unless the name starts with `@0:`:

```
SAVE "HELLO",8        : REM refused if HELLO.bas exists
SAVE "@0:HELLO",8     : REM replaces it
```

A C64 refuses silently, blinking its drive light; c64sh says `c64sh: HELLO.bas: file exists (use SAVE "@0:HELLO" to replace it)` on stderr, and a script stops with exit status 1. Saving to tape (`SAVE "HELLO"`, device 1) always replaces the file, as recording over a tape does. A name may also start with `0:`, the drive number, which is ignored.

See [`examples/features/021-saving-programs.bas`](../examples/features/021-saving-programs.bas) for every form.

## Data files

A program can write and read its own data files:

```
10 OPEN 2,8,2,"SCORES,S,W"
20 PRINT#2,"ALICE";",";12
30 CLOSE 2
40 OPEN 2,8,2,"SCORES"
50 INPUT#2,N$,S
60 CLOSE 2
70 PRINT N$;S
```

prints `ALICE 12 `, and leaves the file `SCORES` in the current directory.

**`OPEN F, DEVICE, SECONDARY, "NAME"`** opens logical file `F` (1 to 255); everything after `F` is optional. As on a C64:

| Device | What it is | Notes |
|---|---|---|
| 8 to 11 | Disk drives | A file in the current directory. The secondary address 0 reads and 1 writes; with 2 to 14, the name says: `"NAME,S,W"` writes, `"NAME,S,A"` adds to the end, and `"NAME,S,R"` or plain `"NAME"` reads. Writing replaces an existing file only with `@0:`, as for `SAVE`. Secondary address 15 is the drive's command channel (see [Disk commands and the directory](#disk-commands-and-the-directory)). |
| 1 | Tape (the default) | A file in the current directory: the secondary address 0 (the default) reads, 1 or 2 writes. |
| 0 | Keyboard | Read with `INPUT#` and `GET#`, as `INPUT` and `GET` read stdin. |
| 3, 4, 5 | Screen, printers | Write with `PRINT#` or `CMD`: the output appears with the program's output. |

- **`PRINT# F, items`** writes like `PRINT`, ending each line with a newline unless the items end with `;` or `,`. Note that there is no space between `PRINT` and `#`: `PRINT #2` is a `?SYNTAX  ERROR`, as on a C64.
- **`INPUT# F, variables`** reads values like `INPUT`, without a prompt. A number it cannot read is a `?FILE DATA  ERROR`.
- **`GET# F, variables`** reads one character at a time; a line end reads as `CHR$(13)`, and the end of the file as an empty string. (`GET #F` with a space also works.)
- **`ST`** tells a reading loop when to stop: it is 64 once reading reaches the end of the file, 66 when reading a disk file that could not be opened, and 0 otherwise.

  ```
  20 INPUT#2,A$:PRINT A$:IF ST=0 THEN 20
  ```
- **`CMD F`** sends everything `PRINT` and `LIST` would show to file `F`, until a `PRINT#` to any file: `OPEN 4,4:CMD 4:LIST` lists a program to the printer.
- **`CLOSE F`** finishes the file. Output reaches the disk when the file is closed; files still open are closed when the variables are cleared (`RUN`, `NEW`, changing the program) and when c64sh ends.
- **Commas in `PRINT#`** move to the next print zone of the *screen's* cursor, as on a C64, so `PRINT#2,"A","B"` usually puts 10 spaces between them.
- **Files are text** with Unix line ends, so other tools can read and write them.
- **Errors**: opening a file number already open is `?FILE OPEN  ERROR`, more than 10 open files is `?TOO MANY FILES  ERROR`, a file number not open is `?FILE NOT OPEN  ERROR`, reading an output file (or the screen) is `?NOT INPUT FILE  ERROR`, writing an input file is `?NOT OUTPUT FILE  ERROR`, and opening a missing tape file to read is `?FILE NOT FOUND  ERROR`.
- **Disk errors are not BASIC errors**, as on a 1541: opening a disk file that does not exist, or writing one that exists without `@0:`, carries on, and the drive's status says what went wrong (below). The program decides what to do about it.

See [`examples/features/022-data-files.bas`](../examples/features/022-data-files.bas) for every form.

### Disk commands and the directory

A disk drive has a **command channel**, secondary address 15, as on a 1541. Reading it gives the drive's **status**; writing to it sends **commands**:

```
10 OPEN 15,8,15
20 PRINT#15,"S0:OLD*"
30 INPUT#15,E,E$,T,S
40 PRINT E;E$;T;S
50 CLOSE 15
```

scratches (deletes) every file whose name starts with `OLD`, and prints ` 1 FILES SCRATCHED 3  0 ` if there were three.

**The status** is four values: an error number (0 means all is well), a message, and two numbers (a count of files scratched, or 0). Reading it resets it to `0, OK`. It begins as ` 73 CBM DOS V2.6 1541`, as a 1541's does when switched on.

| Status | Means |
|---|---|
| `0, OK` | The last command or file worked |
| `1, FILES SCRATCHED, N` | `N` files were deleted |
| `62, FILE NOT FOUND` | The file opened, loaded, or renamed does not exist |
| `63, FILE EXISTS` | A file of that name exists (write with `@0:` to replace it) |
| `26, WRITE PROTECT ON` | A format was refused (below) |
| `30` to `34, SYNTAX ERROR` | The command was not understood |

**Commands** are sent with `PRINT#15,"…"`, or as the name in `OPEN 15,8,15,"…"`:

| Command | Does |
|---|---|
| `S0:NAME` | **Scratch**: delete the file. Several names may be given, separated by commas, and `*` matches the rest of a name and `?` any one character: `S0:*` deletes every file in the current directory, so use it with care. |
| `R0:NEW=OLD` | **Rename** `OLD` to `NEW`. |
| `N0:NAME,ID` | **Format**: on a C64 this erases the whole disk. c64sh refuses, with `26, WRITE PROTECT ON`, as a 1541 refuses a protected disk. |
| `I0`, `V0` | Initialize, validate: accepted, with nothing to do. |
| `UJ` | Reset the drive: the status becomes the power-on message. |

The `0` after the letter is the drive number and may be left out (`S:NAME`). A name without an extension also finds a saved program: `S0:GAME` deletes `GAME.bas` if there is no file called `GAME`. Hidden files (names starting with `.`) and directories are never touched.

**The directory**: `LOAD "$",8` loads a listing of the current directory as a program, replacing the one in memory as on a C64, and `LIST` shows it:

```
0 "C64SH           " 00 2A
1    "GAME.bas"         PRG
3    "SCORES"           SEQ
664 BLOCKS FREE.
```

Each line number is a file's size in 254-byte blocks; programs (`.bas`) show as `PRG`, other files as `SEQ`. `LOAD "$:G*",8` lists only the files matching a pattern. The first line is the disk's header, in reverse video on a terminal; c64sh's disk is always called `C64SH`, with 664 blocks free, as an empty 1541 disk has.

See [`examples/features/035-disk-commands.bas`](../examples/features/035-disk-commands.bas) for every form.

## Environment variables

*c64sh extension: not part of C64 BASIC V2. The words are GW-BASIC's, a later Microsoft BASIC.*

Programs can read and set environment variables, such as `HOME`, `USER`, and `PATH`:

```
PRINT ENVIRON$("HOME")
ENVIRON "GREETING=HELLO"
ENVIRON "PATH=";ENVIRON$("PATH");":/opt/bin"
```

| Form | Does |
|---|---|
| `ENVIRON$("NAME")` | The variable's value, or `""` if it is not set |
| `ENVIRON$(N)` | The `N`th variable (from 1, in order of name) as `NAME=VALUE`, or `""` past the last |
| `ENVIRON "NAME=VALUE"` | Sets the variable; an empty value (`ENVIRON "NAME="`) removes it |

- **Long values.** Environment values are often longer than a C64 string's 255 characters, and `PATH` usually is. c64sh's strings are unlimited by default, so `ENVIRON "PATH="+ENVIRON$("PATH")+":/opt/bin"` works. With [`C64SH_STRING_LIMIT`](#settings) set to `255`, `PRINT ENVIRON$("PATH")` still shows the whole value, but storing it in a variable or joining it with `+` is `?STRING TOO LONG  ERROR`, so a value is never cut short.
- **Changes last for the session**, and programs c64sh starts see them; they do not change the shell that started c64sh.
- **Errors.** Text without `=`, or with nothing before it, is `?ILLEGAL QUANTITY  ERROR`; a number is `?TYPE MISMATCH  ERROR`.

See [`examples/features/031-environment.bas`](../examples/features/031-environment.bas) for every form, and [idiomatic BASIC](idioms.md#the-environment) for common patterns.

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
| `?STRING TOO LONG  ERROR` | With a string limit set (`C64SH_STRING_LIMIT`), a string grew longer than the limit. |
| `?TYPE MISMATCH  ERROR` | A string compared with a number or used with `AND`, `OR`, or `NOT`, `+` given a string and a number, or `-`, `*`, `/`, `^`, or a minus sign in front was given a string. |
| `?OVERFLOW  ERROR` | A number, or the result of a calculation, is larger than 1.70141183E+38. |
| `?DIVISION BY ZERO  ERROR` | Dividing by zero. |
| `?UNDEF'D STATEMENT  ERROR` | `RUN`, `GOTO`, or `IF … THEN` with a line number that is not in the program. |
| `?NEXT WITHOUT FOR  ERROR` | `NEXT` with no loop to continue. |
| `?OUT OF MEMORY  ERROR` | More than 10 loops, 26 subroutine calls, or 9 function calls, nested, or arrays larger than a C64's memory. |
| `?ILLEGAL DIRECT  ERROR` | `INPUT`, `GET`, or `DEF` typed directly; they work only in a program. |
| `?FILE NOT FOUND  ERROR` | `LOAD`, `VERIFY`, or `OPEN` (to read or add to it) of a file that does not exist. |
| `?LOAD  ERROR` | `LOAD` of a file that is not a program. |
| `?VERIFY  ERROR` | `VERIFY` of a file that differs from the program. |
| `?MISSING FILE NAME  ERROR` | `LOAD`, `SAVE`, or `VERIFY` without a name. |
| `?ILLEGAL DEVICE NUMBER  ERROR` | `LOAD`, `SAVE`, or `VERIFY` with the keyboard (0), the screen (3), or a printer (4, 5). |
| `?DEVICE NOT PRESENT  ERROR` | `LOAD`, `SAVE`, or `VERIFY` with a device other than tape (1), a disk drive (8 to 11), the keyboard, the screen, or a printer. |
| `?FILE OPEN  ERROR` | `OPEN` with a file number already open. |
| `?FILE NOT OPEN  ERROR` | `PRINT#`, `INPUT#`, `GET#`, or `CMD` with a file number not open. |
| `?NOT INPUT FILE  ERROR` | `INPUT#` or `GET#` with a file opened for writing, the screen, or a printer; `OPEN 0`. |
| `?NOT OUTPUT FILE  ERROR` | `PRINT#` or `CMD` with a file opened for reading, or the keyboard. |
| `?TOO MANY FILES  ERROR` | `OPEN` with 10 files already open. |
| `?FILE DATA  ERROR` | `INPUT#` reading something other than a number into a number variable. |
| `?UNDEF'D FUNCTION  ERROR` | `FN` with a function that has not been defined. |
| `?BAD SUBSCRIPT  ERROR` | An array subscript past the top, or the wrong number of subscripts. |
| `?REDIM'D ARRAY  ERROR` | `DIM` of an array that already exists. |
| `?OUT OF DATA  ERROR` | `READ` with no `DATA` items left. |
| `?RETURN WITHOUT GOSUB  ERROR` | `RETURN` with no `GOSUB` to return to. |
| `?ILLEGAL QUANTITY  ERROR` | A number outside -32768 to 32767 used with `AND`, `OR`, or `NOT`; a negative number raised to a fractional power, such as `(-8)^(1/3)`, or a number outside -32768 to 32767 stored in an integer variable (`C%`). |

An error stops the rest of its line. Anything printed before the error stays printed, as on a C64:

```
PRINT "A":PRINT "B"@
```

prints `A` and `B`, then `?SYNTAX  ERROR`.

Errors always start on a new line, even if the output before them ended with `;`.

An error in a running program adds the number of the line it happened in, as on a C64:

```
?SYNTAX  ERROR IN 20
```

## Exit status

| Status | Meaning |
|---|---|
| 0 | Success, or an interactive session ended with Ctrl-D |
| 1 | A script stopped at a BASIC error, input ended while `INPUT` or `GET` was waiting, a file could not be saved or read, or output could not be written |
| 2 | c64sh was run incorrectly: an unknown option, more than one file, or a file that cannot be read |
| 130 | A script was stopped by Ctrl-C (`BREAK`) |

For example, `c64sh build.bas && echo done` prints `done` only if the script ran without errors.

## Extensions

These features are c64sh's own: they are not part of Commodore 64 BASIC V2 and do not work on a real C64. Each is marked *c64sh extension* where it is described.

**Shell**

| Extension | What it does |
|---|---|
| [Editing and history](#editing-and-history) | Edit the line you are typing, and recall earlier lines, even from earlier sessions |
| [Shell colors](#shell-colors) | Color what you type, `READY.`, and errors differently |
| [Startup file](#startup-file-c64shrc) | Run `~/.c64shrc` at the start of every interactive session |
| [Settings](#settings) | Choose history size, history file, and colors with environment variables |

**Language**

| Extension | What it does |
|---|---|
| [Environment variables](#environment-variables) | `ENVIRON$` reads and `ENVIRON` sets environment variables such as `PATH` |

## Differences from a real C64

- **`,` fills print zones with spaces.** A C64 moves its cursor right on screen instead, leaving whatever was there; in a terminal, output only ever appears after the cursor, so spaces look the same.
- **Errors go to stderr** and set a non-zero exit status in scripts.
- **No screen emulation**: no 40-column wrapping or graphics characters; the screen's control codes, such as colors and clearing the screen, are translated to the terminal's own (see [Colors and the screen](#colors-and-the-screen)), and PETSCII graphics characters are not. So a print zone past column 40 stays on the same line: `PRINT 2,3,4,5,6` prints ` 6 ` at column 40, where a C64 would start a new screen line.
- **Arithmetic uses standard 64-bit floating point**, rounded to the C64's 9 digits when printed. Results match a C64 in nearly every case; a C64's own rounding occasionally differs in the last digit. Rarely, a loop with a fractional `STEP`, such as `FOR I=0 TO 1 STEP .1`, runs a different number of times than on a C64.
- **The banner** reads `C64SH BASIC V2`.
- **Lines can be longer than 80 characters**, typed or in a file. A C64's screen editor stops at two screen lines, 80 characters.
- **No `?FORMULA TOO COMPLEX  ERROR`.** A C64 gives it when an expression needs more than three temporary strings at once, such as `A$+(B$+(C$+D$))`; c64sh evaluates such expressions.
- **Memory limits apply to arrays only**: as on a C64, all arrays together must fit in 38911 bytes, but strings and programs are not limited (see [Free memory](#free-memory)).
- **Strings have no length limit** by default, where a C64's hold 255 characters. No C64 program can catch `?STRING TOO LONG`, so this never changes how a working program behaves; set `C64SH_STRING_LIMIT=255` to get the C64's limit back.
- **The shell colors its own text** in a terminal (see [Shell colors](#shell-colors)); a C64 shows everything in the current color.
- **Ctrl-C stops a program waiting in `INPUT`.** On a C64, the STOP key does nothing until Return is pressed.
- **Typed input keeps lowercase letters**, where a C64 keyboard types uppercase.
- **Programs are saved as text**, never as the C64's tokenized `.prg` files, on purpose: c64sh programs can use lowercase letters, any Unicode character, and extensions, none of which a C64's format can hold. So programs can't be exchanged with a real C64 or an emulator. Every device that holds programs is the current directory.
- **`SAVE` over an existing disk file** prints `c64sh: NAME.bas: file exists (use SAVE "@0:NAME" to replace it)`, where a 1541 only flashes its light, so a save is never lost silently. The drive's status says `63, FILE EXISTS` too.
- **Data files use Unix line ends**, where a C64 writes a carriage return.

## Not yet supported

These are valid C64 BASIC but currently give `?SYNTAX  ERROR`:

- All other commands

These are planned; the [roadmap](https://github.com/bryanesmith/c64sh/issues/34) lists them in the order they will be added.

Every form c64sh accepts is described in this guide, and shown in use in [`examples/features/`](../examples/features/).

## Not planned

These C64 BASIC keywords work directly on the C64's memory and processor:

| Keyword | On a C64 |
|---|---|
| `PEEK(a)` | Returns the byte stored at memory address `a`, such as `PEEK(197)`, the key being pressed. |
| `POKE a,v` | Writes byte `v` to memory address `a`. This is how C64 programs set colors, graphics, and sound: `POKE 53280,0` makes the border black. |
| `SYS a` | Runs machine code starting at address `a`. |
| `WAIT a,m` | Pauses until a bit at address `a` changes, such as a hardware signal. |
| `USR(x)` | Calls a machine-code routine set up beforehand with `POKE`, passing it `x` and returning a number. |

c64sh will not support them: each would need an emulated C64 behind it, with its memory, its video, sound, and I/O chips, and a 6502 processor to run machine code. c64sh runs BASIC in a terminal, without emulating the machine. These keywords give `?SYNTAX  ERROR`.

**Tokenized `.prg` files** are not planned either. A C64 saves programs as tokens in `PRG` files, which emulators and real C64s exchange. c64sh saves programs as text, and its programs can hold what a `PRG` file cannot: lowercase letters, any Unicode character, and c64sh's extensions. Giving up that exchange is deliberate, in favor of a BASIC that is comfortable on a modern machine.
