# 2. A world in DATA

[Tutorial](index.md) · Previous: [The title screen](01-the-title-screen.md) · Next: [Getting around](03-getting-around.md)

The castle has seven rooms. Rather than write seven blocks of `PRINT` statements, we describe the rooms as data, and write one subroutine that can describe any of them. This is the heart of a text adventure, and of most good BASIC programs: **keep the facts in `DATA`, and the logic in code.**

## The rooms

```basic
10000 REM ROOMS: NAME, DESCRIPTION, DARK, EXITS N S E W
10010 DATA 7
10020 DATA GATEHOUSE,"A CRUMBLING GATEHOUSE. A PATH LEADS NORTH.",0,2,0,0,0
10030 DATA COURTYARD,"A WEEDY COURTYARD WITH DOORS ALL AROUND.",0,5,1,3,4
```

The first item is the number of rooms. Then each room has a name, a description, whether it is dark (1 or 0; we use this in chapter 6), and the rooms its exits lead to, north, south, east, and west, with 0 for no exit. The courtyard's exits lead north to room 5, south to room 1, and so on. A description contains commas, so it goes in quotes.

`DATA` lines can be anywhere in the program; putting them at the end, with high line numbers, keeps them out of the way.

## Reading the rooms

```basic
9010 READ NR:DIM RN$(NR),RD$(NR),DK(NR),EX(NR,4)
9020 FOR I=1 TO NR:READ RN$(I),RD$(I),DK(I)
9030 FOR D=1 TO 4:READ EX(I,D):NEXT D,I
```

`READ` takes the next item from the `DATA` lines. Line 9010 reads the number of rooms into `NR`, then `DIM` makes **arrays** of that size: `RN$(I)` is room `I`'s name, `RD$(I)` its description, `DK(I)` its darkness, and `EX(I,D)` the room you reach going in direction `D` (1 north, 2 south, 3 east, 4 west). `EX` has two subscripts, so it is a table.

The `FOR` loops count `I` through the rooms, and `D` through the directions; `NEXT D,I` ends both. A name ending in `$` holds a string.

**Variable names**: only the first two characters count, so programmers kept names short: `RN$` for "room name", `EX` for "exits". And a name cannot contain a keyword: a C64 finds keywords anywhere, so a name like `ON$` would be read as the keyword `ON` followed by `$`. Short names avoid that trap.

## Describing a room

`R` is the room the player is in. The description subroutine prints its name, its description, and its exits:

```basic
1020 PRINT:PRINT RN$(R):PRINT RD$(R)
1030 E$="":FOR D=1 TO 4
1040 IF EX(R,D) THEN E$=E$+" "+MID$("NSEW",D,1)
1050 NEXT:PRINT "EXITS:";E$
```

`E$` starts empty, and for each direction with an exit, line 1040 adds a space and the direction's letter: `MID$("NSEW",D,1)` is the `D`th letter of `"NSEW"`. `IF EX(R,D)` is true when the exit is not 0.

Notice that the loop spans three lines. That is deliberate: when an `IF` is false, BASIC skips **the rest of its line**, so if `NEXT` were on the same line as the `IF`, a false condition would skip the `NEXT` too. Putting the `IF` on a line of its own, inside the loop, is the idiom.

Line 40 calls the setup and then describes the room; line 9110 puts the player in room 1, the gatehouse.

## The program so far

```basic
10 REM THE LOST AMULET
20 GOSUB 9000:REM READ THE WORLD
30 GOSUB 8000:REM TITLE SCREEN
40 GOSUB 1000:REM DESCRIBE THE ROOM
50 END
1000 REM DESCRIBE ROOM R
1020 PRINT:PRINT RN$(R):PRINT RD$(R)
1030 E$="":FOR D=1 TO 4
1040 IF EX(R,D) THEN E$=E$+" "+MID$("NSEW",D,1)
1050 NEXT:PRINT "EXITS:";E$
1099 RETURN
8000 REM TITLE SCREEN
8005 PRINT CHR$(147):REM CLEAR THE SCREEN
8010 PRINT TAB(12);CHR$(158);"THE LOST AMULET";CHR$(154)
8020 PRINT TAB(12);"---------------"
8030 PRINT "THE AMULET OF DAWN LIES SOMEWHERE IN THE"
8040 PRINT "RUINED CASTLE. FIND IT AND CARRY IT BACK"
8050 PRINT "TO THE GATEHOUSE. TYPE HELP FOR COMMANDS."
8060 PRINT:PRINT "PRESS ANY KEY TO BEGIN."
8070 GET K$:IF K$="" THEN 8070
8080 RETURN
9000 REM READ THE WORLD FROM DATA
9010 READ NR:DIM RN$(NR),RD$(NR),DK(NR),EX(NR,4)
9020 FOR I=1 TO NR:READ RN$(I),RD$(I),DK(I)
9030 FOR D=1 TO 4:READ EX(I,D):NEXT D,I
9110 R=1:RETURN
10000 REM ROOMS: NAME, DESCRIPTION, DARK, EXITS N S E W
10010 DATA 7
10020 DATA GATEHOUSE,"A CRUMBLING GATEHOUSE. A PATH LEADS NORTH.",0,2,0,0,0
10030 DATA COURTYARD,"A WEEDY COURTYARD WITH DOORS ALL AROUND.",0,5,1,3,4
10040 DATA WELL,"AN OLD WELL. SOMETHING GLINTS BELOW.",0,0,0,0,2
10050 DATA STABLES,"EMPTY STABLES THAT SMELL OF OLD HAY.",0,0,0,2,0
10060 DATA GREAT HALL,"A VAST HALL. STAIRS GO DOWN TO THE NORTH.",0,6,2,7,0
10070 DATA CRYPT,"A COLD CRYPT FULL OF DUSTY TOMBS.",1,0,5,0,0
10080 DATA KITCHEN,"A KITCHEN WITH A CRACKED OVEN.",0,0,0,0,5
```
