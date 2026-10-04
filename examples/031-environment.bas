#!/usr/bin/env c64sh
REM Environment variables with ENVIRON$ and ENVIRON: a c64sh extension, not part of C64 BASIC V2
REM ENVIRON "NAME=VALUE" sets a variable; ENVIRON$("NAME") reads it
ENVIRON "GREETING=HELLO"
PRINT ENVIRON$("GREETING"):REM HELLO
REM A variable that is not set reads as the empty string
PRINT "[";ENVIRON$("NOT SET");"]":REM []
REM Everything after the first = is the value, even another =
ENVIRON "EQUATION=1+1=2"
PRINT ENVIRON$("EQUATION"):REM 1+1=2
REM The name and value can come from expressions
N$="PLANET":ENVIRON N$+"=EARTH"
PRINT ENVIRON$(N$):REM EARTH
REM Extend PATH by joining with +; strings are unlimited by default, so
REM this works even for a PATH longer than a C64 string's 255 characters
ENVIRON "PATH=/usr/bin"
ENVIRON "PATH="+ENVIRON$("PATH")+":/opt/c64/bin"
PRINT ENVIRON$("PATH"):REM /usr/bin:/opt/c64/bin
REM ENVIRON$(N) is the Nth variable as NAME=VALUE, in order of name,
REM and the empty string past the last
PRINT ENVIRON$(1):REM EQUATION=1+1=2
PRINT "[";ENVIRON$(99);"]":REM []
REM So a loop up to the first empty string lists them all
10 I=1
20 E$=ENVIRON$(I):IF E$="" THEN END
30 PRINT E$:I=I+1:GOTO 20:REM EQUATION=1+1=2, GREETING=HELLO, PATH=/usr/bin:/opt/c64/bin, PLANET=EARTH
RUN
REM An empty value removes the variable
ENVIRON "GREETING="
PRINT "[";ENVIRON$("GREETING");"]":REM []
REM Text without = is an error
ENVIRON "NO EQUALS SIGN"
