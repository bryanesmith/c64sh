#!/usr/bin/env c64sh
REM Number functions take their argument in parentheses
PRINT ABS(-7);ABS(7):REM " 7  7 "
PRINT SGN(-2);SGN(0);SGN(5):REM "-1  0  1 "
REM INT rounds down, so negative numbers move away from zero
PRINT INT(3.9);INT(-3.1):REM " 3 -4 "
REM Rounding to the nearest whole number is INT(X+.5)
PRINT INT(2.5+.5);INT(2.4+.5):REM " 3  2 "
PRINT SQR(25);SQR(2):REM " 5  1.41421356 "
PRINT EXP(1);LOG(EXP(3)):REM " 2.71828183  3 "
REM Angles are in radians; π is the C64's pi key
PRINT π:REM " 3.14159265 "
PRINT SIN(π/6);COS(0);ATN(1)*4:REM " .5  1  3.14159265 "
REM Degrees to radians: multiply by π/180
5 DEF FN RAD(D)=D*π/180
10 PRINT TAN(FN RAD(45)):REM " 1 " when RUN below
RUN
REM RND(1) gives the next number of a pseudo-random sequence, from 0 up
REM to (but not including) 1. RND with a negative number starts a sequence
REM that is the same every time, so the next lines always print the same.
X=RND(-42)
REM A die roll is INT(RND(1)*6)+1, a whole number from 1 to 6
FOR I=1 TO 10:PRINT INT(RND(1)*6)+1;:NEXT:PRINT:REM " 2  5  6  5  3  4  5  6  3  6 "
X=RND(-42):PRINT INT(RND(1)*6)+1:REM " 2 " (the same sequence again)
REM RND(0) takes a number from the clock instead, different in each run.
REM SQR of a negative number is an error, so the next line prints
REM ?ILLEGAL QUANTITY  ERROR, and the script stops
PRINT SQR(-1)
