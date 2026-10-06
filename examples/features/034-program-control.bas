#!/usr/bin/env c64sh
REM LIST ranges, CLR, END and CONT, and STOP
10 PRINT "ONE":REM ONE
20 PRINT "TWO":REM TWO
30 PRINT "THREE":REM THREE
REM LIST n lists one line, LIST n-m a range, LIST -m up to m, LIST n- from n
REM Each listing begins with a blank line, as on a C64
LIST 20
LIST 20-30
LIST -10
LIST 30-
NEW
REM CLR clears the variables, but keeps the program
A=5:B$="HI"
CLR:PRINT A;B$:REM " 0 "
REM CONT continues a program after END (and after STOP or Ctrl-C),
REM keeping its variables: X is set while the program is stopped
10 PRINT "BEFORE END":END
20 PRINT "AFTER END";X:REM AFTER END 42, once CONT continues
RUN
X=42
CONT
REM STOP stops a program with BREAK IN and its line, as the STOP key does.
REM Typed at the prompt, CONT would then continue it; a script stops here
30 PRINT "STOPPING":STOP:PRINT "CONTINUED"
GOTO 30
