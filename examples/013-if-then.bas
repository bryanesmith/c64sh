#!/usr/bin/env c64sh
REM IF ... THEN: running statements only when a condition is true
A=5
IF A>3 THEN PRINT "A IS BIG":REM A IS BIG
IF A<3 THEN PRINT "A IS SMALL":REM (nothing: the condition is false)
REM When the condition is false, the whole rest of the line is skipped,
REM including statements after a colon
IF A>3 THEN PRINT "ONE":PRINT "TWO":REM ONE, then TWO on the next line
IF A<3 THEN PRINT "ONE":PRINT "TWO":REM (nothing at all)
REM Statements before the IF still run
PRINT "BEFORE";:IF A<3 THEN PRINT "NEVER":REM BEFORE (no newline)
PRINT:REM (ends the line above)
REM Any number other than 0 counts as true
IF -1 THEN PRINT "TRUE":REM TRUE
IF .5 THEN PRINT "ALSO TRUE":REM ALSO TRUE
IF 0 THEN PRINT "FALSE":REM (nothing)
REM A string counts as true when it is not empty
N$="ALICE"
IF N$ THEN PRINT "HELLO, ";N$:REM HELLO, ALICE
IF "" THEN PRINT "EMPTY":REM (nothing)
REM Conditions can be combined with AND, OR, and NOT
IF A>1 AND A<10 THEN PRINT "BETWEEN":REM BETWEEN
IF NOT A=5 THEN PRINT "NOT FIVE":REM (nothing)
REM IF can set variables, and IFs can follow one another
IF A=5 THEN B=A*2:PRINT B:REM " 10 "
IF A>1 THEN IF A<3 THEN PRINT "TWO":REM (nothing: the second IF is false)
REM There is no ELSE in BASIC V2; use a second IF with the opposite test
IF A>9 THEN PRINT "BIG":REM (nothing)
IF NOT A>9 THEN PRINT "NOT BIG":REM NOT BIG
REM A skipped part of the line is never checked, so even a mistake there
REM goes unnoticed. The next line prints nothing and causes no error:
IF A<3 THEN PRINT "OOPS"@
REM When the condition is true, the mistake is reached, so the next line
REM prints OOPS and then ?SYNTAX  ERROR, and the script stops
IF A>3 THEN PRINT "OOPS"@
PRINT "NEVER PRINTED":REM (never runs: the script stopped at the error)
