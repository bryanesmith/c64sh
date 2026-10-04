# Project: Battleship

[Tutorial](../index.md)

Three ships hide on an 8 by 8 sea; you fire until you sink them all. *Battleship* is a good workout for **two-dimensional arrays**: the sea is a table of numbers, and nearly everything the program does is a loop over it.

```
   A B C D E F G H
 1 . . . . . . . .
 2 . X X . . O . .
 3 . . . . . . . .
```

## The sea as a table

```basic
20 DIM B(8,8):REM 0 SEA, 1 SHIP, 2 MISS, 3 HIT
```

Each square holds a code. Firing changes the code, and the display turns codes into characters with one `MID$`:

```basic
640 FOR C=1 TO 8:PRINT MID$("..OX",B(R,C)+1,1);" ";:NEXT
```

Codes 0 and 1 both show as `.`, so ships stay hidden; 2 shows `O`, and 3 `X`. Looking a character up in a string by position is a common BASIC trick.

## Placing the ships

```basic
110 FOR K=1 TO 3:READ L:GOSUB 500:T=T+L:NEXT
120 DATA 2,3,4
```

The ships' lengths come from `DATA`, and `T` adds up how many squares must be hit. The placing subroutine picks a direction and a starting square at random, and starts again if the ship would run off the sea or cross another ship:

```basic
530 R=FN R(8):C=FN R(8)
540 IF C+DX*(L-1)>8 OR R+DY*(L-1)>8 THEN 510
550 FOR I=0 TO L-1
560 IF B(R+DY*I,C+DX*I) THEN 510
570 NEXT
```

`DX` and `DY` are 1 and 0 for a ship across, or 0 and 1 for one going down, so `R+DY*I,C+DX*I` walks along the ship. Jumping out of the loop with `GOTO 510` is fine: the next `FOR I` starts the loop afresh.

## Reading a shot

```basic
240 C=ASC(S$)-64:R=VAL(MID$(S$,2))
250 IF C<1 OR C>8 OR R<1 OR R>8 THEN 270
```

`ASC` gives the code of the first character (`A` is 65), so subtracting 64 turns `A` to `H` into 1 to 8; `VAL` reads the number after it. Anything out of range, including lowercase letters, gets an explanation and another try.

## The complete program

```basic
10 REM BATTLESHIP
20 DIM B(8,8):REM 0 SEA, 1 SHIP, 2 MISS, 3 HIT
30 PRINT TAB(14);"BATTLESHIP"
40 PRINT "THREE SHIPS HIDE ON AN 8 BY 8 SEA. SINK"
50 PRINT "THEM ALL. AIM WITH A LETTER AND A NUMBER,"
60 PRINT "LIKE C4."
70 PRINT:PRINT "PRESS ANY KEY."
80 GET K$:IF K$="" THEN 80
90 X=RND(-TI)
100 DEF FN R(X)=INT(RND(1)*X)+1
110 FOR K=1 TO 3:READ L:GOSUB 500:T=T+L:NEXT
120 DATA 2,3,4
200 REM PLAY
210 GOSUB 600:REM SHOW THE SEA
220 PRINT:INPUT "YOUR SHOT";S$
230 IF LEN(S$)<2 THEN 270
240 C=ASC(S$)-64:R=VAL(MID$(S$,2))
250 IF C<1 OR C>8 OR R<1 OR R>8 THEN 270
260 GOTO 300
270 PRINT "AIM WITH A LETTER FROM A TO H AND A"
280 PRINT "NUMBER FROM 1 TO 8, LIKE C4.":GOTO 220
300 IF B(R,C)>1 THEN PRINT "YOU HAVE ALREADY FIRED THERE.":GOTO 220
310 N=N+1:IF B(R,C)=0 THEN B(R,C)=2:PRINT "SPLASH! A MISS.":GOTO 210
320 B(R,C)=3:H=H+1:PRINT "HIT!"
330 IF H<T THEN 210
340 GOSUB 600:PRINT:PRINT "YOU SANK EVERY SHIP IN";N;"SHOTS!"
350 END
500 REM PLACE A SHIP OF LENGTH L AT RANDOM
510 DX=0:DY=0:IF FN R(2)=1 THEN DX=1:GOTO 530
520 DY=1
530 R=FN R(8):C=FN R(8)
540 IF C+DX*(L-1)>8 OR R+DY*(L-1)>8 THEN 510
550 FOR I=0 TO L-1
560 IF B(R+DY*I,C+DX*I) THEN 510
570 NEXT
580 FOR I=0 TO L-1:B(R+DY*I,C+DX*I)=1:NEXT
590 RETURN
600 REM SHOW THE SEA: . FOR SEA (AND HIDDEN SHIPS), O MISS, X HIT
610 PRINT:PRINT "  ";
620 FOR C=1 TO 8:PRINT " ";CHR$(64+C);:NEXT:PRINT
630 FOR R=1 TO 8:PRINT R;
640 FOR C=1 TO 8:PRINT MID$("..OX",B(R,C)+1,1);" ";:NEXT
650 PRINT:NEXT:RETURN
```
