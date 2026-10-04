#!/usr/bin/env c64sh
REM TI counts jiffies (sixtieths of a second) since c64sh started, as a
REM C64's clock counts from when it is switched on. TI$ is the same clock
REM as HHMMSS. In the snapshot test the clock stands still at the start,
REM so the comments show what that run prints.
PRINT TI;TI$:REM " 0 000000"
REM Assigning six digits to TI$ sets the clock
TI$="123000"
PRINT TI$;TI:REM "123000 2700000 "
REM Seconds since the clock was last reset, as a timer:
TI$="000000":REM (start the timer)
PRINT "ELAPSED";INT(TI/60);"SECONDS":REM "ELAPSED 0 SECONDS"
REM A classic idiom: seed RND from the clock after waiting for a key, so
REM the moment the player presses it makes every game different. The
REM snapshot test presses K (test/snapshot/testdata/029-clock.input).
10 PRINT "PRESS A KEY":REM PRESS A KEY
20 GET K$:IF K$="" THEN 20
30 X=RND(-TI)
40 PRINT "SEEDED":REM SEEDED
RUN
REM TI cannot be assigned, only TI$, so the next line prints
REM ?SYNTAX  ERROR, and the script stops
TI=0
