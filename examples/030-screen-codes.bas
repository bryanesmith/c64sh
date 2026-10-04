#!/usr/bin/env c64sh
REM Printing certain characters controls the screen instead of showing a
REM character, as on a C64. In a terminal they become its escape codes;
REM when output is a file or a pipe, as in the snapshot test, the codes
REM that only change the screen are left out, so the comments below show
REM that plain output, and what a terminal shows.
REM
REM CHR$(147) clears the screen and CHR$(19) moves the cursor home
PRINT CHR$(147);"A CLEAN SCREEN":REM A CLEAN SCREEN (after clearing)
REM The 16 colors: CHR$(28) red, CHR$(30) green, CHR$(31) blue,
REM CHR$(158) yellow, CHR$(5) white, CHR$(154) light blue, and more
PRINT CHR$(28);"RED ";CHR$(30);"GREEN ";CHR$(154);"LIGHT BLUE":REM RED GREEN LIGHT BLUE
REM A color stays until another is printed, so set it back afterwards
PRINT CHR$(158);"WARNING:";CHR$(154);" LOW FUEL":REM WARNING: LOW FUEL
REM CHR$(18) turns reverse video on, CHR$(146) off
PRINT CHR$(18);" MENU ";CHR$(146);" CHOOSE ONE":REM " MENU  CHOOSE ONE"
REM CHR$(29) moves the cursor right, and becomes a space everywhere
PRINT "A";CHR$(29);CHR$(29);"B":REM A  B
REM CHR$(13) is Return: it starts a new line
PRINT "LINE 1";CHR$(13);"LINE 2":REM LINE 1, then LINE 2
REM Keeping the codes in variables makes programs readable
RE$=CHR$(28):LB$=CHR$(154):CL$=CHR$(147)
PRINT RE$;"DANGER!";LB$:REM DANGER!
REM NO_COLOR=1 in the environment turns colors off in a terminal.
