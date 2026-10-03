#!/usr/bin/env c64sh
REM OPEN, PRINT#, INPUT#, GET#, CMD, and CLOSE read and write data files.
REM This example writes SCORES and LOG in the current directory.
REM
REM OPEN takes a file number (1 to 255), a device, a secondary address,
REM and a name. On a disk drive (8 to 11), ",S,W" after the name writes a
REM file, ",S,A" adds to it, and ",S,R" or nothing reads it.
OPEN 2,8,2,"SCORES,S,W"
REM PRINT# writes to the file as PRINT writes to the screen
PRINT#2,"ALICE";",";12
PRINT#2,"BOB";",";7
REM CLOSE finishes the file; nothing reaches the disk until then
CLOSE 2
REM INPUT# reads values back, as INPUT reads the keyboard. ST becomes 64
REM when reading reaches the end of the file, which ends this loop
10 OPEN 2,8,2,"SCORES"
20 INPUT#2,N$,S
30 PRINT N$;S:REM "ALICE 12 ", then "BOB 7 "
40 IF ST=0 THEN 20
50 CLOSE 2
RUN
REM GET# reads one character at a time; a line end reads as CHR$(13)
NEW
10 OPEN 2,8,2,"SCORES"
20 GET#2,C$:IF C$<>"," THEN PRINT C$;:GOTO 20:REM ALICE, a letter at a time
30 PRINT:CLOSE 2:REM (ends the line)
RUN
REM CMD sends everything PRINT and LIST would show to a file, until a
REM PRINT# to it. On the screen, the next line prints nothing.
OPEN 3,8,3,"LOG,S,W":CMD 3,"LOG STARTS":PRINT "QUIETLY":PRINT#3,"END":CLOSE 3
REM The screen (3) and printers (4, 5) are the program's output
OPEN 4,4:PRINT#4,"TO THE PRINTER":CLOSE 4:REM TO THE PRINTER
REM ,S,A adds to a file. Its whole contents are read back here:
OPEN 3,8,3,"LOG,S,A":PRINT#3,"MORE":CLOSE 3
10 OPEN 3,8,3,"LOG"
20 INPUT#3,L$:PRINT L$:IF ST=0 THEN 20:REM LOG STARTS, QUIETLY, END, MORE
30 CLOSE 3
RUN
REM Opening a file that does not exist, to read it, is an error, so the
REM next line prints ?FILE NOT FOUND  ERROR, and the script stops
OPEN 5,8,5,"NO SUCH FILE"
