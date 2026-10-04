#!/usr/bin/env c64sh
REM String length: unlimited by default, where a C64's strings hold at most 255 characters
REM Build a string of 300 characters, ten at a time
A$=""
FOR I=1 TO 30:A$=A$+"0123456789":NEXT
PRINT LEN(A$):REM " 300 "
REM Positions in LEFT$, RIGHT$, and MID$ may go past 255
PRINT MID$(A$,291):REM 0123456789
PRINT LEN(LEFT$(A$,280)):REM " 280 "
REM Past the end gives all there is, as for shorter strings
PRINT LEN(RIGHT$(A$,1000)):REM " 300 "
REM A negative position is still an error
REM (With C64SH_STRING_LIMIT=255 set before c64sh starts, line 5 would
REM stop with ?STRING TOO LONG  ERROR, as on a C64.)
PRINT LEFT$(A$,-1)
