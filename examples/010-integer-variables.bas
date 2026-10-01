#!/usr/bin/env c64sh
REM Integer variables: names ending in % hold whole numbers
REM Numbers print with a space before and after, so these comments
REM show the output in quotes.
C%=5
PRINT C%:REM " 5 "
REM A number with a fraction is rounded down when it is stored
C%=3.7
PRINT C%:REM " 3 "
C%=-3.7
PRINT C%:REM "-4 " (down means toward minus infinity)
C%=.5
PRINT C%:REM " 0 "
REM A, A%, and A$ are three different variables
A=1.5:A%=2:A$="THREE"
PRINT A;A%;A$:REM " 1.5  2 THREE"
REM Only the first two characters count, as for other variables
COUNT%=7
PRINT CO%:REM " 7 "
REM Counting with an integer variable
N%=0
N%=N%+1
N%=N%+1
PRINT N%:REM " 2 "
REM In calculations an integer variable is an ordinary number
PRINT N%/4:REM " .5 "
REM Integer variables hold -32768 to 32767
C%=32767
PRINT C%:REM " 32767 "
C%=-32768
PRINT C%:REM "-32768 "
REM TI% is an ordinary integer variable, unlike TI, the C64's clock
TI%=60
PRINT TI%:REM " 60 "
REM A number outside the range cannot be stored, so the next line prints
REM ?ILLEGAL QUANTITY  ERROR and the script stops
C%=40000
PRINT "NEVER PRINTED":REM (never runs: the script stopped at the error)
