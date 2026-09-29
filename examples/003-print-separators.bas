#!/usr/bin/env c64sh
REM How PRINT separates items and ends its line
REM "," moves to the next print zone. Zones start every 10 columns
REM (0, 10, 20, 30, ...), and c64sh fills the gap with spaces.
REM The ruler printed next shows the column numbers.
PRINT "0123456789012345678901234567890":REM 0123456789012345678901234567890
PRINT "NAME","SCORE":REM NAME      SCORE
PRINT "A","B","C":REM A         B         C
PRINT ,"INDENTED":REM           INDENTED
PRINT "A",,"C":REM A                   C
REM At the start of a zone, "," moves a whole zone: X lands at column 20
PRINT "0123456789","X":REM 0123456789          X
REM Anywhere else, "," moves to the start of the next zone
PRINT "012345678","X":REM 012345678 X
PRINT "01234567890","X":REM 01234567890         X
REM PRINT on its own prints an empty line
PRINT:REM (an empty line)
REM A trailing ";" leaves the line open, so the next PRINT continues it
PRINT "HELLO, ";:REM HELLO, (no newline)
PRINT "WORLD":REM WORLD, completing the line HELLO, WORLD
REM A trailing "," moves to the next zone and also leaves the line open
PRINT "LEFT",:REM LEFT and spaces up to column 10 (no newline)
PRINT "RIGHT":REM RIGHT, completing the line LEFT      RIGHT
REM The column carries over, so a later PRINT still lines up with the zones
PRINT "AB";:REM AB (no newline)
PRINT ,"X":REM spaces up to column 10, then X, completing AB        X
REM Extra ";" separators print nothing
PRINT "A";;"B";;;"C":REM ABC
REM ":" separates statements, so one line can hold several PRINTs
PRINT "ONE":PRINT "TWO":REM ONE, then TWO on the next line
PRINT "SAME ";:PRINT "LINE":REM SAME LINE
REM Empty statements between colons do nothing
PRINT "X"::PRINT "Y":REM X, then Y on the next line
::
PRINT "DONE":REM DONE (the line above holds only empty statements)
