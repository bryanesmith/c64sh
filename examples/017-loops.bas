#!/usr/bin/env c64sh
REM FOR ... NEXT repeats the statements between them, counting a variable
REM from a start value to an end value
FOR I=1 TO 5:PRINT I;:NEXT I:PRINT:REM " 1  2  3  4  5 "
REM STEP sets how much the variable changes each time; it can be negative
REM or a fraction
FOR I=0 TO 10 STEP 5:PRINT I;:NEXT:PRINT:REM " 0  5  10 "
FOR I=3 TO 1 STEP -1:PRINT I;:NEXT:PRINT:REM " 3  2  1 "
FOR I=1 TO 2 STEP .5:PRINT I;:NEXT:PRINT:REM " 1  1.5  2 "
REM NEXT with no variable continues the innermost loop. The variable keeps
REM its last value: one STEP past the end
FOR I=1 TO 3:NEXT:PRINT I:REM " 4 "
REM The body always runs at least once, because the test is made at NEXT
FOR I=5 TO 1:PRINT "ONCE":NEXT:REM ONCE
REM The end and the step are worked out once, at FOR
N=2:FOR I=1 TO N:N=9:PRINT I;:NEXT:PRINT:REM " 1  2 "
REM Loops can be nested; NEXT J,I closes both
FOR I=1 TO 3:FOR J=1 TO I:PRINT "*";:NEXT J:PRINT:NEXT I
REM The loop above prints a triangle:
REM *
REM **
REM ***
FOR I=1 TO 2:FOR J=1 TO 2:PRINT I;J;:NEXT J,I:PRINT:REM " 1  1  1  2  2  1  2  2 "
REM In a program, a loop can span many lines
10 FOR X=1 TO 3
20 PRINT X*X;:REM " 1  4  9 " over the whole loop (no newline)
30 NEXT X
40 PRINT:REM (ends the line)
RUN
REM NEXT without a FOR is an error, so the next line prints
REM ?NEXT WITHOUT FOR  ERROR, and the script stops
NEXT
