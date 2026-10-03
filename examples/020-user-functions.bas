#!/usr/bin/env c64sh
REM DEF FN defines a one-line function of one number, used as FN NAME(...)
REM anywhere a value can go. DEF works only in a program.
10 DEF FN SQ(X)=X*X
20 PRINT FN SQ(3);FN SQ(1.5):REM " 9  2.25 "
30 PRINT FN SQ(2)+FN SQ(3):REM " 13 "
REM The parameter belongs to the function: the program's own X is untouched
40 X=100:PRINT FN SQ(4);X:REM " 16  100 "
REM The body can use other variables, with their values at the time of the
REM call, and other functions
50 DEF FN TAX(P)=P*RATE
60 RATE=.25:PRINT FN TAX(80):REM " 20 "
70 RATE=.5:PRINT FN TAX(80):REM " 40 "
80 DEF FN BAD(A)=A+*2
90 DEF FN CU(N)=N*FN SQ(N)
100 PRINT FN CU(3):REM " 27 "
REM Only the first two characters of a name count, as for variables, and a
REM function's name is separate from any variable with the same name
110 SQ=5:PRINT FN SQUARE(5);SQ:REM " 25  5 "
REM The body is checked only when the function is called: line 80 above has
REM a mistake (+* in a row), but defining it caused no error.
REM Calling it does, so RUN prints the lines above, then ?SYNTAX  ERROR IN
REM 120, and the script stops
120 PRINT FN BAD(2):REM (never printed: the call fails)
RUN
