#!/usr/bin/env c64sh
REM String functions measure strings, take them apart, and build them
A$="COMMODORE"
PRINT LEN(A$):REM " 9 "
PRINT LEFT$(A$,3);" ";RIGHT$(A$,4):REM COM DORE
REM MID$ counts from 1; without a length it runs to the end
PRINT MID$(A$,4,3);" ";MID$(A$,7):REM MOD ORE
REM Asking for more than there is gives what there is
PRINT LEFT$(A$,99);"|";MID$(A$,20);"|":REM COMMODORE||
REM CHR$ and ASC turn codes into characters and back
PRINT CHR$(72);CHR$(73);ASC("A"):REM "HI 65 "
PRINT CHR$(34);"QUOTED";CHR$(34):REM "QUOTED"
REM STR$ turns a number into a string, with PRINT's leading space;
REM VAL reads the number a string starts with
PRINT "[";STR$(42);"]";VAL("3.5 CUPS")*2:REM "[ 42] 7 "
REM A loop over a string's characters, idiomatic BASIC:
FOR I=LEN(A$) TO 1 STEP -1:PRINT MID$(A$,I,1);:NEXT:PRINT:REM ERODOMMOC
REM Checking just the first letter of an answer
B$="YES PLEASE":IF LEFT$(B$,1)="Y" THEN PRINT "AGREED":REM AGREED
REM Building a string of repeated characters
L$="":FOR I=1 TO 5:L$=L$+"*":NEXT:PRINT L$:REM *****
REM MID$ positions start at 1, so position 0 is an error, and the next
REM line prints ?ILLEGAL QUANTITY  ERROR, and the script stops
PRINT MID$(A$,0,1)
