#!/usr/bin/env c64sh
REM Arithmetic: + - * / and parentheses
REM Numbers print with a space before and after, so these comments
REM show the output in quotes.
PRINT 2+3:REM " 5 "
PRINT 10-4:REM " 6 "
PRINT 5*9:REM " 45 "
PRINT 7/2:REM " 3.5 "
PRINT "5*9=";5*9:REM "5*9= 45 "
REM * and / come before + and -
PRINT 2+3*4:REM " 14 "
PRINT 20-10/2:REM " 15 "
REM Parentheses come first of all
PRINT (2+3)*4:REM " 20 "
PRINT ((1+2)*(3+4)):REM " 21 "
PRINT 100/(2*5):REM " 10 "
REM Operators of the same level work left to right
PRINT 10-4-3:REM " 3 " (that is, (10-4)-3)
PRINT 100/10/2:REM " 5 " (that is, (100/10)/2)
PRINT 8/4*2:REM " 4 " (that is, (8/4)*2)
REM A minus sign in front makes a number negative
PRINT -5:REM "-5 " (the minus sign takes the place of the leading space)
PRINT -2*3:REM "-6 "
PRINT 2*-3:REM "-6 "
PRINT -(2+3):REM "-5 "
PRINT --5:REM " 5 " (minus minus is plus)
PRINT 5--5:REM " 10 "
REM A plus sign in front changes nothing
PRINT +5:REM " 5 "
REM Division keeps up to 9 significant digits
PRINT 1/3:REM " .333333333 "
PRINT 2/3:REM " .666666667 "
PRINT 1/3*3:REM " 1 "
PRINT 22/7:REM " 3.14285714 "
REM Results too small for a C64 become zero
PRINT 1E-30*1E-30:REM " 0 "
REM A space between a number and "(" starts a new item: two numbers
PRINT 2(3):REM " 2  3 "
REM Dividing by zero is an error, so the next line prints RESULT: and
REM then ?DIVISION BY ZERO  ERROR, and the script stops
PRINT "RESULT:";1/0
PRINT "NEVER PRINTED":REM (never runs: the script stopped at the error)
