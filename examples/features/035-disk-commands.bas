#!/usr/bin/env c64sh
REM The disk drive: its command channel, its status, and its directory
REM A subroutine that reads drive 8's status from its command channel,
REM opened with secondary address 15, and prints it
1000 OPEN 15,8,15:INPUT#15,E,E$,T,S:CLOSE 15:PRINT E;E$;T;S:RETURN:REM the status
REM Before anything else, the status is the message a 1541 gives at power-on
GOSUB 1000:REM " 73 CBM DOS V2.6 1541 0  0 "
REM Once read, the status is OK
GOSUB 1000:REM " 0 OK 0  0 "
REM Make some files to work with, and save the program as PROG.bas
OPEN 2,8,2,"NOTES,S,W":PRINT#2,"HELLO":CLOSE 2
OPEN 2,8,2,"SCORES,S,W":PRINT#2,"ALICE":CLOSE 2
OPEN 2,8,2,"SAVED,S,W":PRINT#2,"BOB":CLOSE 2
SAVE "PROG",8
REM Opening a file that exists to write it, without @0:, leaves it alone
OPEN 2,8,2,"NOTES,S,W":PRINT#2,"GONE?":CLOSE 2
GOSUB 1000:REM " 63 FILE EXISTS 0  0 "
REM Commands are sent as OPEN's name, or with PRINT#. R renames NEW=OLD
OPEN 15,8,15,"R0:LETTERS=NOTES":CLOSE 15
GOSUB 1000:REM " 0 OK 0  0 "
REM S scratches (deletes) files: * matches the rest of a name, ? any one
REM character, and the status counts the files scratched
OPEN 15,8,15:PRINT#15,"S0:S*":CLOSE 15
GOSUB 1000:REM " 1 FILES SCRATCHED 2  0 "
REM N would format the disk, erasing everything; c64sh refuses, as a 1541
REM refuses a write-protected disk
OPEN 15,8,15,"N0:MY DISK,01":CLOSE 15
GOSUB 1000:REM " 26 WRITE PROTECT ON 0  0 "
REM An unknown command is a syntax error
OPEN 15,8,15,"X":CLOSE 15
GOSUB 1000:REM " 31 SYNTAX ERROR 0  0 "
REM LOAD "$",8 loads the directory as a program, replacing the one in
REM memory, and LIST shows it: sizes in blocks, names with extensions, and
REM PRG for programs or SEQ for other files. On a terminal, the header line
REM is in reverse video
LOAD "$",8
LIST
REM A pattern lists only the matching files
LOAD "$:L*",8
LIST
NEW
