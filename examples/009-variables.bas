#!/usr/bin/env c64sh
REM Variables: setting them with =, and using them
REM Numbers print with a space before and after, so these comments
REM show the output in quotes.
A=5
PRINT A:REM " 5 "
REM LET in front is optional; it means the same thing
LET B=A*2+1
PRINT "B IS";B:REM "B IS 11 "
REM A name ending in $ holds a string
N$="ALICE"
PRINT "HELLO, ";N$:REM "HELLO, ALICE"
PRINT N$+N$:REM "ALICEALICE"
REM A variable can be updated from its own value
A=A+1
PRINT A:REM " 6 "
REM A and A$ are two different variables
A$="SIX"
PRINT A;A$:REM " 6 SIX"
REM A variable that was never set is 0, or empty for a string
PRINT X:REM " 0 "
PRINT "["X$"]":REM "[]"
REM Only the first two characters of a name count
SCORE=100
PRINT SC:REM " 100 " (SCORE and SC are the same variable)
PRINT SCX:REM " 100 " (so is SCX)
REM Spaces inside a name are ignored
MY VALUE=7
PRINT MYVALUE:REM " 7 "
REM The C64 finds keywords even inside names, which leads to surprises
ER=4
PRINTER:REM " 4 " (read as PRINT ER)
LETTER=9
PRINT TE:REM " 9 " (read as LET TER=9, and TER is TE)
REM Variables mix with numbers and strings anywhere
PRINT "TOTAL:";A*SC+1:REM "TOTAL: 601 "
PRINT A,B:REM " 6 " at column 0, " 11 " at column 10
REM A string cannot go in a number variable, so the next line prints
REM ?TYPE MISMATCH  ERROR and the script stops
A="HELLO"
PRINT "NEVER PRINTED":REM (never runs: the script stopped at the error)
