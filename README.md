# c64sh

A shell that speaks Commodore 64 BASIC V2, in your terminal. Type a line and it runs, just like the C64's direct mode, or run BASIC scripts like any other command.

c64sh currently supports `PRINT` with strings, numbers, and arithmetic (`+ - * / ^` and parentheses), number functions (`INT`, `RND`, `SIN`, …), string functions (`LEN`, `MID$`, `CHR$`, …), comparisons, logic (`AND`, `OR`, `NOT`), `IF … THEN`, `FOR … NEXT` loops, `GOSUB` subroutines, keyboard input (`INPUT`, `GET`), `DEF FN` functions, `SAVE`/`LOAD`, data files (`OPEN`, `PRINT#`, `INPUT#`), variables (including `%` integers) arrays, `DATA` statements, the clock (`TI`, `TI$`), `REM` comments, and programs with numbered lines (`RUN`, `LIST`, `NEW`, `END`, `GOTO`, `ON … GOTO`). More of the language is on the way.

## Build and run

Requires Go 1.27 or later and `make`.

```sh
make            # build bin/c64sh
make run        # build, then start an interactive session
make install    # build, then install to ~/bin/c64sh
make test       # run all tests
```

## Example

```
$ echo 'PRINT "HELLO, ";"WORLD"' | c64sh
HELLO, WORLD
```

See the [user guide](docs/user-guide.md) for everything else.
