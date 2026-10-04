#!/usr/bin/env c64sh
REM GOSUB n runs the subroutine starting at line n; RETURN comes back to
REM the statement just after the GOSUB
10 PRINT "MAIN STARTS":REM MAIN STARTS
20 GOSUB 100
30 PRINT "BACK IN MAIN":REM BACK IN MAIN
40 GOSUB 100
50 END
100 PRINT "  IN THE SUBROUTINE":REM "  IN THE SUBROUTINE", printed twice
110 RETURN
REM RUN prints MAIN STARTS, the subroutine's line, BACK IN MAIN, and the
REM subroutine's line again; END stops the program before line 100
RUN
REM RETURN comes back mid-line, and subroutines can call subroutines
NEW
10 PRINT "A";:GOSUB 100:PRINT "D":REM ABCD (B and C come from lines 100, 200)
20 END
100 PRINT "B";:GOSUB 200:RETURN:REM B (no newline)
200 PRINT "C";:RETURN:REM C (no newline)
RUN
REM A subroutine has no parameters or result; it works on the program's
REM variables. This one squares X into R.
NEW
10 X=3:GOSUB 100:PRINT R:REM " 9 "
20 X=R:GOSUB 100:PRINT R:REM " 81 "
30 END
100 R=X*X
110 RETURN
RUN
REM Typed directly, GOSUB runs the subroutine and then the rest of the line
GOSUB 100:PRINT "DONE";R:REM "DONE 81 " (X is still 9 from the program)
REM RETURN with no GOSUB is an error, so RUN prints FIRST, then
REM ?RETURN WITHOUT GOSUB  ERROR IN 20, and the script stops
NEW
10 PRINT "FIRST":REM FIRST
20 RETURN
RUN
