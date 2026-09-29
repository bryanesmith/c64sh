#!/usr/bin/env c64sh
REM How PRINT separates items and ends its line
REM "," moves to the next column; c64sh prints it as a tab
PRINT "NAME","SCORE":REM NAME<TAB>SCORE
PRINT "A","B","C":REM A<TAB>B<TAB>C
PRINT ,"INDENTED":REM <TAB>INDENTED
PRINT "A",,"C":REM A<TAB><TAB>C
REM PRINT on its own prints an empty line
PRINT:REM (an empty line)
REM A trailing ";" leaves the line open, so the next PRINT continues it
PRINT "HELLO, ";:REM HELLO, (no newline)
PRINT "WORLD":REM WORLD, completing the line HELLO, WORLD
REM A trailing "," prints a tab and also leaves the line open
PRINT "LEFT",:REM LEFT<TAB>(no newline)
PRINT "RIGHT":REM RIGHT, completing the line LEFT<TAB>RIGHT
REM Extra ";" separators print nothing
PRINT "A";;"B";;;"C":REM ABC
REM ":" separates statements, so one line can hold several PRINTs
PRINT "ONE":PRINT "TWO":REM ONE, then TWO on the next line
PRINT "SAME ";:PRINT "LINE":REM SAME LINE
REM Empty statements between colons do nothing
PRINT "X"::PRINT "Y":REM X, then Y on the next line
::
PRINT "DONE":REM DONE (the line above holds only empty statements)
