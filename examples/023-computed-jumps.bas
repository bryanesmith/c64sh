#!/usr/bin/env c64sh
REM ON X GOTO picks a line from a list by number: the first line when X is
REM 1, the second when X is 2, and so on
10 FOR X=1 TO 3
20 ON X GOTO 100,200,300
30 NEXT X
40 END
100 PRINT "ONE":GOTO 30:REM ONE (when X is 1)
200 PRINT "TWO":GOTO 30:REM TWO (when X is 2)
300 PRINT "THREE":GOTO 30:REM THREE (when X is 3)
RUN
REM ON X GOSUB calls the line as a subroutine; RETURN comes back after the
REM whole ON statement. X is rounded down, and when it is 0 or past the end
REM of the list, nothing is called
NEW
10 FOR X=0 TO 4 STEP .5
20 PRINT X;:ON X GOSUB 100,200:PRINT:REM X, what it calls, and a newline
30 NEXT X
40 END
100 PRINT "CALLS 100";:RETURN:REM CALLS 100 (no newline)
200 PRINT "CALLS 200";:RETURN:REM CALLS 200 (no newline)
REM Each line of the loop above prints X, then what it calls, if anything:
REM " 0 ", " .5 ", " 1 CALLS 100", " 1.5 CALLS 100", " 2 CALLS 200",
REM " 2.5 CALLS 200", " 3 ", " 3.5 ", " 4 "
RUN
REM ON works typed directly too
ON 2 GOSUB 100,200:PRINT:REM CALLS 200, then the line ends
REM X must be from 0 to 255, so the next line prints
REM ?ILLEGAL QUANTITY  ERROR, and the script stops
ON -1 GOTO 100
