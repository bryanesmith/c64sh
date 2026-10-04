#!/usr/bin/env c64sh
REM Program mode: a line that starts with a number is stored, not run
REM Stored lines run in number order when RUN is typed
20 PRINT "WORLD":REM WORLD
10 PRINT "HELLO":REM HELLO
REM RUN prints HELLO, then WORLD
RUN
REM Typing a line with the same number replaces it
20 PRINT "THERE":REM THERE
REM RUN prints HELLO, then THERE
RUN
REM LIST shows the program in number order, after a blank line, with each
REM line as typed, except that ? is shown as PRINT, the keyword it stands for
15 ?   "DEAR ";:REM DEAR (no newline)
REM LIST prints a blank line, then these three lines:
REM 10 PRINT "HELLO":REM HELLO
REM 15 PRINT   "DEAR ";:REM DEAR (no newline)
REM 20 PRINT "THERE":REM THERE
LIST
REM RUN prints HELLO, then DEAR THERE
RUN
REM A number alone deletes its line
15
REM RUN with a line number starts there: RUN 20 prints THERE
RUN 20
REM NEW erases the program
NEW
REM RUN clears all variables first, so A starts at 0 each time
10 A=A+1:PRINT A:REM " 1 "
RUN
RUN
REM Storing or deleting a line also clears the variables, as on a C64
B=5
30 REM
PRINT B:REM " 0 "
REM END stops the program; the lines after it do not run
NEW
10 PRINT "BEFORE END":REM BEFORE END
20 END
30 PRINT "AFTER END":REM (never printed: END stopped the program)
RUN
REM A mistake in a stored line is found only when the line runs, and the
REM error names the line. So RUN prints FIRST and SECOND, then
REM ?SYNTAX  ERROR IN 20, and the script stops
NEW
10 PRINT "FIRST":REM FIRST
20 PRINT "SECOND"@:REM SECOND, then the error
30 PRINT "THIRD":REM (never printed: the program stopped at line 20)
RUN
PRINT "NEVER PRINTED":REM (never runs: the script stopped at the error)
