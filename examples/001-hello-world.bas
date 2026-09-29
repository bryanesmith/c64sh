#!/usr/bin/env c64sh
REM Printing text with PRINT, and its shortcut "?"
REM PRINT writes a string, then starts a new line
PRINT "HELLO WORLD":REM HELLO WORLD
REM The space after PRINT is optional
PRINT"HELLO WORLD":REM HELLO WORLD
REM "?" is short for PRINT, with or without a space
? "HELLO WORLD":REM HELLO WORLD
?"HELLO WORLD":REM HELLO WORLD
REM Spaces inside quotes are kept; spaces outside them are ignored
PRINT "  HELLO   WORLD  ":REM   HELLO   WORLD   (all spaces kept)
PRINT    "HELLO WORLD"    :REM HELLO WORLD
REM Everything between the quotes is printed exactly as written
PRINT "HELLO, WORLD! 1+2=3; A:B":REM HELLO, WORLD! 1+2=3; A:B
PRINT "lowercase is fine inside quotes":REM lowercase is fine inside quotes
PRINT "":REM (an empty line)
REM The closing quote may be left off at the end of a line.
REM The next line prints HELLO WORLD
PRINT "HELLO WORLD
REM An unclosed string runs to the end of the line, so a colon and
REM REM after it are part of the text. The next line prints HI:REM THERE
PRINT "HI:REM THERE
