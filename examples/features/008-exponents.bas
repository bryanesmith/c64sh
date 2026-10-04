#!/usr/bin/env c64sh
REM Exponents with ^ (or the up-arrow, as on the C64 keyboard)
REM The output is shown in quotes, since numbers print with spaces.
PRINT 2^3:REM " 8 "
PRINT 2↑3:REM " 8 " (the up-arrow means the same as ^)
PRINT 10^2:REM " 100 "
PRINT 2^10:REM " 1024 "
REM A fractional exponent takes a root
PRINT 9^.5:REM " 3 "
PRINT 27^(1/3):REM " 3 "
PRINT 2^.5:REM " 1.41421356 "
REM A negative exponent divides
PRINT 2^-1:REM " .5 "
PRINT 10^-2:REM " .01 "
REM ^ comes before everything else, even a minus sign in front
PRINT -2^2:REM "-4 " (that is, -(2^2))
PRINT (-2)^2:REM " 4 "
PRINT 2*3^2:REM " 18 " (that is, 2*(3^2))
PRINT 3^2*2:REM " 18 " (that is, (3^2)*2)
PRINT 1+2^3:REM " 9 "
REM Several ^ work left to right, unlike in most of mathematics
PRINT 2^3^2:REM " 64 " (that is, (2^3)^2, not 2^9)
PRINT 2^(3^2):REM " 512 "
REM A minus sign in an exponent takes in any ^ after it, but not * or /
PRINT 2^-1^2:REM " .5 " (that is, 2^(-(1^2)))
PRINT 2^-3*4:REM " .5 " (that is, (2^-3)*4)
REM Special cases, as on a C64
PRINT 5^0:REM " 1 "
PRINT 0^0:REM " 1 "
PRINT 0^5:REM " 0 "
PRINT 0^-1:REM " 0 " (no division by zero)
PRINT (-2)^3:REM "-8 " (a negative number to a whole power is fine)
PRINT -8^(1/3):REM "-2 " (that is, -(8^(1/3)))
REM Results too big for a C64 are errors, and too small become zero
PRINT 10^-50:REM " 0 "
REM A negative number to a fractional power has no real answer, so the
REM next line prints ?ILLEGAL QUANTITY  ERROR and the script stops
PRINT (-8)^(1/3)
PRINT "NEVER PRINTED":REM (never runs: the script stopped at the error)
