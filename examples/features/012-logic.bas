#!/usr/bin/env c64sh
REM Logic: AND, OR, and NOT
REM True is -1 and false is 0, so these combine comparisons. Numbers
REM print with a space (or minus sign) before and a space after, so
REM these comments show the output in quotes.
PRINT 1<2 AND 3<4:REM "-1 " (both true)
PRINT 1<2 AND 3>4:REM " 0 " (one false)
PRINT 1>2 OR 3<4:REM "-1 " (one true)
PRINT NOT 1=2:REM "-1 " (that is, NOT (1=2))
REM AND comes before OR, so this is 0 OR (1 AND 0)
PRINT 0 OR 1 AND 0:REM " 0 "
PRINT (0 OR 1) AND 1:REM " 1 " (bit by bit: 0 OR 1 is 1, and 1 AND 1 is 1)
REM They work bit by bit on whole numbers from -32768 to 32767
PRINT 12 AND 10:REM " 8 " (binary 1100 AND 1010 is 1000)
PRINT 12 OR 10:REM " 14 " (binary 1110)
PRINT NOT 0:REM "-1 " (every bit flipped)
PRINT NOT 5:REM "-6 "
REM Fractions are rounded down first
PRINT 7.9 AND 15:REM " 7 "
REM NOT takes in everything up to the next AND or OR
PRINT 1+NOT 0+1:REM "-1 " (that is, 1+NOT (0+1))
REM With variables
A=5:B=10
PRINT A<B AND B<20:REM "-1 "
OK=A>3 OR B>100
PRINT OK:REM "-1 "
REM A number outside -32768 to 32767 cannot be used, so the next line
REM prints ?ILLEGAL QUANTITY  ERROR and the script stops
PRINT 40000 AND 1
PRINT "NEVER PRINTED":REM (never runs: the script stopped at the error)
