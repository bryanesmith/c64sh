#!/usr/bin/env c64sh
REM A script runs each line as if typed: numbered lines are stored, other
REM lines run at once, and a program the script never runs with RUN runs
REM after its last line. So a script of numbered lines needs no RUN.
30 PRINT "THREE":REM THREE (third, when the program runs at the end)
PRINT "FIRST":REM FIRST (at once: this line has no number)
10 PRINT "ONE":REM ONE (first, when the program runs at the end)
20 PRINT "TWO":REM TWO (second, when the program runs at the end)
PRINT "SECOND":REM SECOND (at once)
REM The script ends here, so the program runs: ONE, TWO, THREE
