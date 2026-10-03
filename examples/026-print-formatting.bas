#!/usr/bin/env c64sh
REM TAB(N) moves to column N; SPC(N) moves right N columns. The ruler
REM shows the columns:
PRINT "0123456789012345678901234567890":REM 0123456789012345678901234567890
PRINT "NAME";TAB(12);"SCORE";TAB(22);"LEVEL":REM NAME        SCORE     LEVEL
PRINT "ALICE";TAB(12);120;TAB(22);3:REM "ALICE        120       3 "
REM If the cursor is already past column N, TAB does nothing
PRINT "A VERY LONG NAME";TAB(12);"X":REM A VERY LONG NAMEX
PRINT "A";SPC(5);"B":REM A     B
REM TAB and SPC work with any expression, like a bar chart:
FOR I=1 TO 3:PRINT I;SPC(I*2);"#":NEXT
REM The loop prints " 1   #", then " 2     #", then " 3       #"
REM Ending a PRINT with TAB or SPC leaves the line open, like ;
PRINT "LEFT";TAB(10):PRINT "RIGHT":REM LEFT      RIGHT
REM POS(0) tells which column the cursor is in
PRINT "HELLO";:PRINT POS(0):REM "HELLO 5 "
REM Columns are 0 to 255, so the next line prints ?ILLEGAL QUANTITY
REM ERROR, and the script stops
PRINT TAB(300);"X"
