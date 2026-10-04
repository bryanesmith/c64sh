#!/usr/bin/env c64sh
REM GOTO n continues a program at line n, so lines can be skipped, or
REM repeated in a loop
10 PRINT "START":REM START
20 GOTO 40
30 PRINT "SKIPPED":REM (never printed: line 20 jumps past it)
40 PRINT "END":REM END
REM RUN prints START, then END
RUN
REM IF ... THEN n jumps to line n only when the condition is true, so this
REM program counts to 5 and then stops
NEW
10 I=I+1
20 PRINT I;:REM " 1  2  3  4  5 " over the whole loop (no newline)
30 IF I<5 THEN 10
40 PRINT:REM (ends the line)
RUN
REM IF ... GOTO n means the same as IF ... THEN n, and GO TO n, with a
REM space, the same as GOTO n. GO and TO are keywords, so names that
REM contain them, like GOLD or TOTAL, cannot be used.
NEW
10 IF A=0 GOTO 30
20 PRINT "A IS NOT 0":REM (never printed: A is 0)
30 PRINT "A IS 0":REM A IS 0
40 GO TO 60
50 PRINT "SKIPPED":REM (never printed: line 40 jumps past it)
60 PRINT "DONE":REM DONE
RUN
REM Typed directly, GOTO n runs the program from line n, keeping the
REM variables; RUN clears them first
NEW
10 PRINT "N IS";N:REM "N IS 7 " after GOTO 10, then "N IS 0 " after RUN
N=7
GOTO 10
RUN
REM A GOTO to a line that does not exist is an error naming the line of
REM the GOTO, so RUN prints BEFORE, then ?UNDEF'D STATEMENT  ERROR IN 20,
REM and the script stops
NEW
10 PRINT "BEFORE":REM BEFORE
20 GOTO 100
RUN
