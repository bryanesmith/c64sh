#!/usr/bin/env c64sh
REM Comparisons: = <> < > <= >=
REM A comparison is a number: -1 when true, 0 when false. Numbers print
REM with a space (or minus sign) before and a space after, so these
REM comments show the output in quotes.
PRINT 1<2:REM "-1 "
PRINT 2<1:REM " 0 "
PRINT 5=5:REM "-1 "
PRINT 5<>6:REM "-1 " (<> means "not equal")
PRINT 3<=3:REM "-1 "
PRINT 4>=5:REM " 0 "
REM Comparisons come after arithmetic
PRINT 1+1=2:REM "-1 " (that is, (1+1)=2)
PRINT 2*3>5:REM "-1 "
REM Strings compare letter by letter, alphabetically
PRINT "APPLE"<"BANANA":REM "-1 "
PRINT "HI"="HI":REM "-1 "
PRINT "A"<"AB":REM "-1 " (a shorter start comes first)
REM The symbols can come in either order, even with spaces between
PRINT 3><4:REM "-1 " (the same as <>)
PRINT 4=<4:REM "-1 " (the same as <=)
PRINT 5=>6:REM " 0 " (the same as >=)
PRINT 1 < > 2:REM "-1 "
REM All three together are always true
PRINT 1<=>2:REM "-1 "
REM Comparisons work left to right: (1<2) is -1, and -1<3
PRINT 1<2<3:REM "-1 "
REM = assigns at the start of a statement, and compares after that
A=5
PRINT A=5:REM "-1 "
B=1:C=2
A=B=C
PRINT A:REM " 0 " (A holds the result of comparing B with C)
REM A string cannot be compared with a number, so the next line prints
REM ?TYPE MISMATCH  ERROR and the script stops
PRINT "1"=1
PRINT "NEVER PRINTED":REM (never runs: the script stopped at the error)
