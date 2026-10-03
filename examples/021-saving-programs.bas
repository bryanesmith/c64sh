#!/usr/bin/env c64sh
REM SAVE, LOAD, and VERIFY keep programs in files. This example saves
REM DEMO.bas, PART2.bas, and NOTE.TXT in the current directory.
10 PRINT "HELLO FROM DEMO":REM HELLO FROM DEMO
SAVE "DEMO"
REM NEW erases the program, and LOAD brings it back from DEMO.bas
NEW
LOAD "DEMO"
RUN
REM VERIFY checks that the program matches the file. They match, so the
REM next line prints SAME
VERIFY "DEMO":PRINT "SAME":REM SAME
REM The file is text: a #! line, then the lines as typed, so DEMO.bas is
REM also a script that c64sh can run.
REM Tape (device 1, the default) and the disk drives (8 to 11) are all the
REM current directory. This program becomes PART2.bas:
NEW
10 PRINT "PART 2 SEES N =";N:REM "PART 2 SEES N = 7 " (when chained below)
SAVE "PART2",8
REM LOAD in a running program loads the new program and runs it from the
REM start, keeping the variables: this is how C64 programs chain
NEW
10 N=7
20 LOAD "PART2",8
30 PRINT "NEVER REACHED":REM (never printed: line 20 loaded PART2)
RUN
REM A name with an extension keeps it; any other name gets .bas
SAVE "NOTE.TXT",8
REM Saving to tape replaces a file. Saving to disk replaces one only when
REM the name starts with @0:, as on a C64's 1541 drive
SAVE "DEMO"
SAVE "@0:PART2",8
REM Without @0:, the disk keeps the old file. A C64 would only blink its
REM drive light; c64sh says so on stderr, and the script stops:
REM c64sh: PART2.bas: file exists (use SAVE "@0:PART2" to replace it)
SAVE "PART2",8
