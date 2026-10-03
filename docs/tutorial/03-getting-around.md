# 3. Getting around

[Tutorial](index.md) · Previous: [A world in DATA](02-a-world-in-data.md) · Next: [Understanding words](04-understanding-words.md)

Now the player can move. A text adventure is one loop: ask for a command, carry it out, and ask again.

## The main loop

First delete line 50, the `END` after the title: type `50` alone at the prompt, or delete the line in your file. Then add the loop:

```basic
100 REM MAIN LOOP
110 PRINT:INPUT "WHAT NOW";C$
120 IF C$="" THEN 110
125 IF C$="Q" THEN END
130 VB$=C$:NN$="":GOSUB 3000
200 GOTO 110
```

`INPUT "WHAT NOW";C$` prints its prompt and a question mark, and waits for a line, which goes into `C$`. If the player just presses Return, line 120 asks again: `IF … THEN 110` jumps to line 110 when the condition is true. `Q` quits for now. Line 130 hands the command to the movement subroutine (we split commands into a verb and a noun in the next chapter), and line 200 loops back with `GOTO`.

## Moving

```basic
3010 D$=LEFT$(VB$,1):IF D$="G" THEN D$=LEFT$(NN$,1)
3020 D=0:FOR I=1 TO 4
3030 IF MID$("NSEW",I,1)=D$ THEN D=I
3040 NEXT
3050 IF D=0 THEN PRINT "GO WHERE?":RETURN
3060 IF EX(R,D)=0 THEN PRINT "YOU CAN'T GO THAT WAY.":RETURN
3080 R=EX(R,D):GOSUB 1000:RETURN
```

`LEFT$(VB$,1)` is the first letter of the command, so `N` and `NORTH` both mean north. Lines 3020 to 3040 look the letter up in `"NSEW"`, giving `D` from 1 to 4, or 0 if it is not a direction.

Then come the checks, each ending the subroutine early with `RETURN` if something is wrong: not a direction, or no exit that way. This **check, then return** pattern keeps each check on its own line and avoids deep nesting. If both checks pass, line 3080 moves the player through the exit, and describes the new room.

(Line 3010's test for `G` is ready for the next chapter, where `GO NORTH` arrives with `NORTH` in `NN$`.)

## The program so far

```basic
10 REM THE LOST AMULET
20 GOSUB 9000:REM READ THE WORLD
30 GOSUB 8000:REM TITLE SCREEN
40 GOSUB 1000:REM DESCRIBE THE ROOM
100 REM MAIN LOOP
110 PRINT:INPUT "WHAT NOW";C$
120 IF C$="" THEN 110
125 IF C$="Q" THEN END
130 VB$=C$:NN$="":GOSUB 3000
200 GOTO 110
1000 REM DESCRIBE ROOM R
1020 PRINT:PRINT RN$(R):PRINT RD$(R)
1030 E$="":FOR D=1 TO 4
1040 IF EX(R,D) THEN E$=E$+" "+MID$("NSEW",D,1)
1050 NEXT:PRINT "EXITS:";E$
1099 RETURN
3000 REM GO
3010 D$=LEFT$(VB$,1):IF D$="G" THEN D$=LEFT$(NN$,1)
3020 D=0:FOR I=1 TO 4
3030 IF MID$("NSEW",I,1)=D$ THEN D=I
3040 NEXT
3050 IF D=0 THEN PRINT "GO WHERE?":RETURN
3060 IF EX(R,D)=0 THEN PRINT "YOU CAN'T GO THAT WAY.":RETURN
3080 R=EX(R,D):GOSUB 1000:RETURN
8000 REM TITLE SCREEN
8010 PRINT TAB(12);"THE LOST AMULET"
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
