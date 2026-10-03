#!/usr/bin/env c64sh
REM DATA lines hold values inside a program; READ takes them in order
10 READ N$,A
20 PRINT N$;" IS";A:REM "ALICE IS 12 ", the first time through
30 DATA ALICE,12
RUN
REM A loop reads a table of values. A common idiom ends the table with a
REM marker value, here "END"
NEW
10 READ N$:IF N$="END" THEN 40
20 READ P:PRINT N$;TAB(10);P:REM each name, at column 10 its price
30 GOTO 10
40 PRINT "THAT'S ALL":REM THAT'S ALL
100 DATA APPLE,.5,BREAD,2.25
110 DATA CHEESE, 4 , END
REM RUN prints APPLE  .5 , BREAD  2.25 , CHEESE  4 (each price at column
REM 10), then THAT'S ALL
RUN
REM Another idiom fills an array from DATA. Quotes keep commas, colons,
REM and leading spaces in a string; without quotes, keywords are text.
NEW
10 DIM C$(3):FOR I=0 TO 3:READ C$(I):NEXT
20 FOR I=3 TO 0 STEP -1:PRINT C$(I);"|";:NEXT:PRINT:REM PRINT|  LEFT|B,C|A:|
30 DATA "A:","B,C","  LEFT",PRINT
RUN
REM RESTORE starts again from the first DATA item
NEW
10 READ A,B:RESTORE:READ C:PRINT A;B;C:REM " 1  2  1 "
20 DATA 1,2
RUN
REM Reading past the last item is an error, so RUN prints
REM ?OUT OF DATA  ERROR IN 10, and the script stops
NEW
10 READ A,B
20 DATA 1
RUN
