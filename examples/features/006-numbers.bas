#!/usr/bin/env c64sh
REM Numbers: how to write them, and how PRINT shows them
REM PRINT puts a space before a number (where a minus sign would go)
REM and a space after it, so these comments show the output in quotes.
PRINT 45:REM " 45 "
PRINT 3.14:REM " 3.14 "
REM A zero before the point is never printed, nor are trailing zeros
PRINT 0.5:REM " .5 "
PRINT .5:REM " .5 "
PRINT 2.50:REM " 2.5 "
PRINT 007:REM " 7 "
REM A lone "." is zero
PRINT .:REM " 0 "
REM E means "times ten to the power of"
PRINT 1E3:REM " 1000 "
PRINT 2E+4:REM " 20000 "
PRINT 1.5E-2:REM " .015 "
REM Numbers are rounded to 9 significant digits
PRINT 123.4567891:REM " 123.456789 "
PRINT 3.14159265358979:REM " 3.14159265 "
REM From 1E9 up, and below .01, numbers print in scientific notation
PRINT 999999999:REM " 999999999 "
PRINT 1000000000:REM " 1E+09 "
PRINT 1234567890:REM " 1.23456789E+09 "
PRINT .01:REM " .01 "
PRINT .001:REM " 1E-03 "
PRINT .000123:REM " 1.23E-04 "
REM Spaces inside a number are ignored
PRINT 1 000 000:REM " 1000000 "
PRINT 1 . 5:REM " 1.5 "
REM A second "." starts a new number
PRINT 1.2.3:REM " 1.2  .3 "
REM Each number keeps its own spaces, even when joined with ";"
PRINT 1;2;3:REM " 1  2  3 "
REM Numbers and strings mix freely
PRINT "5*9=";45:REM "5*9= 45 "
PRINT "PI IS ABOUT";3.14:REM "PI IS ABOUT 3.14 "
PRINT "A"1"B":REM "A 1 B"
REM "," puts numbers in print zones
PRINT 2,3,4:REM " 2 " at column 0, " 3 " at column 10, " 4 " at column 20
REM "+" adds numbers
PRINT 1+1:REM " 2 "
PRINT 1.5+2.25+3:REM " 6.75 "
PRINT .1+.2:REM " .3 "
PRINT "TOTAL:";10+20+30:REM "TOTAL: 60 "
REM The largest and smallest numbers
PRINT 1.70141183E+38:REM " 1.70141183E+38 " (the largest C64 number)
PRINT 1E-39:REM " 0 " (too small for a C64, so it becomes zero)
REM "+" cannot join a string and a number, so the next line prints
REM ?TYPE MISMATCH  ERROR and the script stops
PRINT "AGE: "+42
PRINT "NEVER PRINTED":REM (never runs: the script stopped at the error)
