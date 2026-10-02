# c64sh

A shell that speaks Commodore 64 BASIC V2, in your terminal. Type a line and it runs, just like the C64's direct mode, or run BASIC scripts like any other command.

c64sh currently supports `PRINT` with strings, numbers, and arithmetic (`+ - * / ^` and parentheses), comparisons, logic (`AND`, `OR`, `NOT`), variables (including `%` integers), and `REM` comments. More of the language is on the way.

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
