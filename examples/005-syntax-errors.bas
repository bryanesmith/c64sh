#!/usr/bin/env c64sh
REM Common mistakes, and the ?SYNTAX  ERROR they cause
REM A script stops at its first error, so only one mistake below
REM actually runs. The others are described here:
REM
REM   print "HI"          keywords must be uppercase: PRINT "HI"
REM   PRINT "A"+          "+" needs a string after it: PRINT "A"+"B"
REM   PRINT 1+2           numbers are not supported yet
REM   10 PRINT "HI"       line numbers (program mode) are not supported yet
REM   PRINT "HI" REM X    a comment needs a colon first: PRINT "HI":REM X
REM
REM Output from before an error still appears
PRINT "THIS LINE WORKS":REM THIS LINE WORKS
REM The next line prints OK, then fails with ?SYNTAX  ERROR, because
REM PRINT ends only at ":" or the end of the line, not at REM
PRINT "OK" REM this comment needed a colon before it
PRINT "NEVER PRINTED":REM (never runs: the script stopped at the error)
