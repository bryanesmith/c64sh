#!/usr/bin/env c64sh
REM SYS runs a program: a c64sh extension, not part of C64 BASIC V2
REM The program is found on the PATH, and each argument is its own string
SYS "echo","HELLO FROM ECHO":REM HELLO FROM ECHO
REM No shell is involved, so spaces, * and $ in an argument are passed as
REM they are, with no quoting
SYS "printf","[%s]\n","TWO WORDS","*","$HOME":REM [TWO WORDS], then [*], then [$HOME]
REM An argument can be any string expression, or a number
N$="WORLD":SYS "echo","HELLO, "+N$,3:REM HELLO, WORLD 3
REM ST holds the program's exit status: 0 for success
SYS "true":PRINT ST:REM " 0 "
SYS "false":PRINT ST:REM " 1 "
REM A shell can still be run on purpose, for pipes and the like
SYS "sh","-c","echo ONE TWO | wc -w | tr -d ' '":REM 2
REM A program that cannot be found is an error. (SYS with an address, as a
REM C64 uses it to call machine code, is ?SYNTAX  ERROR.) The next line
REM prints ?FILE NOT FOUND  ERROR, and the script stops
SYS "no-such-program"
