# 1. The title screen

[Tutorial](index.md) · Next: [A world in DATA](02-a-world-in-data.md)

Every program starts with a line number. Type this, and run it:

```basic
10 REM THE LOST AMULET
```

Nothing happens, and that is right: `REM` is a remark, a comment for people reading the program. Line numbers decide the order in which lines run, not the order you type them, so programmers number in tens (10, 20, 30) to leave room for lines added later.

## Printing

The title screen is a subroutine at line 8000:

```basic
8000 REM TITLE SCREEN
8010 PRINT TAB(12);"THE LOST AMULET"
8020 PRINT TAB(12);"---------------"
```

`PRINT` writes text, and `TAB(12)` moves to column 12 first, centering the title on a 40-column C64 screen. The `;` joins the items without spaces.

## Waiting for a key

```basic
8060 PRINT:PRINT "PRESS ANY KEY TO BEGIN."
8070 GET K$:IF K$="" THEN 8070
```

`GET K$` reads one key *without waiting*: if no key has been pressed, `K$` is the empty string. So line 8070 checks, and if there was no key, goes back to itself. This little loop is how every C64 program waits for a key. `PRINT` on its own prints an empty line, and `:` puts several statements on one line.

## Structure: GOSUB and RETURN

The main program is at the top, and it calls the title screen as a subroutine:

```basic
30 GOSUB 8000:REM TITLE SCREEN
50 END
```

`GOSUB 8000` runs the lines from 8000 until `RETURN`, then comes back. `END` stops the program before it runs on into the subroutines. This layout, a short main program at the top that calls subroutines numbered in the thousands, is how experienced BASIC programmers kept big programs readable: each job has a block of line numbers of its own.

Run the program with `c64sh adventure.bas` (or type `RUN`), and press a key.

## At the prompt

If you are typing at the `c64sh` prompt, two commands help: `LIST` shows the program in line order, and `SAVE "ADVENTURE"` keeps it in the file `ADVENTURE.bas`, which `LOAD "ADVENTURE"` brings back. To change a line, type it again with the same number; to delete one, type its number alone.

## The program so far

```basic
10 REM THE LOST AMULET
30 GOSUB 8000:REM TITLE SCREEN
50 END
8000 REM TITLE SCREEN
8010 PRINT TAB(12);"THE LOST AMULET"
8020 PRINT TAB(12);"---------------"
8030 PRINT "THE AMULET OF DAWN LIES SOMEWHERE IN THE"
8040 PRINT "RUINED CASTLE. FIND IT AND CARRY IT BACK"
8050 PRINT "TO THE GATEHOUSE. TYPE HELP FOR COMMANDS."
8060 PRINT:PRINT "PRESS ANY KEY TO BEGIN."
8070 GET K$:IF K$="" THEN 8070
8080 RETURN
```
