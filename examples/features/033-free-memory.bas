#!/usr/bin/env c64sh
REM FRE: the memory left for BASIC, counted and shown as a C64 shows it
REM With nothing in use there are 38909 bytes free, but FRE gives a signed
REM 16-bit number, so anything above 32767 comes out negative
PRINT FRE(0):REM -26627
REM Adding 65536 to a negative result gives the real figure
PRINT FRE(0)-65536*(FRE(0)<0):REM " 38909 "
REM The argument is ignored, and may be a number or a string
PRINT FRE("ANYTHING")=FRE(0):REM -1
REM Each variable takes 7 bytes
A=1
PRINT FRE(0)-65536*(FRE(0)<0):REM " 38902 "
REM A string takes 7 bytes, plus one per character
B$="HELLO"
PRINT FRE(0)-65536*(FRE(0)<0):REM " 38890 "
REM An array of 10 numbers takes 5+2+5*10 bytes
DIM C(9)
PRINT FRE(0)-65536*(FRE(0)<0):REM " 38833 "
REM A program line takes 5 bytes plus its text, each keyword one byte;
REM storing a line also clears the variables and arrays. The program
REM never runs, so it runs after the last line, printing HI
10 PRINT "HI"
PRINT FRE(0)-65536*(FRE(0)<0):REM " 38898 "
