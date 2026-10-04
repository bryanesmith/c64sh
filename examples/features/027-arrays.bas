#!/usr/bin/env c64sh
REM Arrays hold many values under one name, picked out by a subscript.
REM DIM S(4) makes S(0) to S(4): five numbers, all starting at 0.
DIM S(4)
FOR I=0 TO 4:S(I)=I*I:NEXT
PRINT S(0);S(2);S(4):REM " 0  4  16 "
REM String and integer arrays work the same way
DIM D$(6):D$(0)="SUN":D$(1)="MON":D$(6)="SAT"
PRINT D$(0);" ";D$(6):REM SUN SAT
DIM C%(2):C%(1)=3.9:PRINT C%(1):REM " 3 " (integers round down)
REM An array is separate from a plain variable with the same name
S=99:PRINT S;S(4):REM " 99  16 "
REM Two dimensions make a grid; this is a multiplication table
DIM T(3,3)
FOR R=1 TO 3:FOR C=1 TO 3:T(R,C)=R*C:NEXT C,R
REM The next line prints " 1  2  3 ", then " 2  4  6 ", then " 3  6  9 "
FOR R=1 TO 3:FOR C=1 TO 3:PRINT T(R,C);:NEXT C:PRINT:NEXT R
REM An array used without DIM has 10 as the top of each dimension
N(10)=5:PRINT N(10):REM " 5 "
REM Two classic idioms in a program: find the largest value, and swap
REM two elements. A false IF skips the rest of its line, so the NEXT goes
REM on a line of its own.
10 DIM V(4):V(0)=3:V(1)=8:V(2)=1:V(3)=9:V(4)=4
20 M=V(0):FOR I=1 TO 4
30 IF V(I)>M THEN M=V(I)
40 NEXT:PRINT "LARGEST:";M:REM "LARGEST: 9 "
50 X=V(0):V(0)=V(1):V(1)=X:PRINT V(0);V(1):REM " 8  3 "
REM DIM of an array that exists, even one made by use, is an error
60 N(1)=1:DIM N(20)
REM So RUN prints the lines above, then ?REDIM'D ARRAY  ERROR IN 60
RUN
