#!/usr/bin/env c64sh
REM INPUT and GET read the keyboard while a program runs. Run at a
REM terminal, this script asks you. When input is not a terminal, c64sh
REM shows each answer after its prompt, as a C64 screen would. The snapshot
REM test answers from test/snapshot/testdata/019-keyboard-input.input, and
REM the comments below show the output for those answers.
REM
REM INPUT shows its prompt and "? ", then waits for a line
10 INPUT "WHAT IS YOUR NAME";N$:REM WHAT IS YOUR NAME? ALICE (answer ALICE)
20 PRINT "HELLO, ";N$:REM HELLO, ALICE
REM One INPUT can read several values, separated by commas
30 INPUT "TWO NUMBERS";A,B:REM TWO NUMBERS? 3,4 (answer 3,4)
40 PRINT A;"+";B;"=";A+B:REM " 3 + 4 = 7 "
REM When the answer has too few values, INPUT asks for more with "?? "
50 INPUT "X AND Y";X,Y:REM X AND Y? 5, then ?? 6 (answers 5, then 6)
60 PRINT X*Y:REM " 30 "
REM A number INPUT cannot read makes it start again
70 INPUT "AGE";G:REM AGE? TEN, ?REDO FROM START, AGE? 10 (answers TEN, 10)
80 PRINT "NEXT YEAR";G+1:REM "NEXT YEAR 11 "
REM Values left over are ignored, with a warning
90 INPUT "ONE WORD";W$:REM ONE WORD? HI,THERE, then ?EXTRA IGNORED
100 PRINT W$:REM HI
REM Quotes keep a comma inside a string
110 INPUT "CITY";C$:REM CITY? "PARIS, FRANCE"
120 PRINT C$:REM PARIS, FRANCE
REM GET reads one key without waiting, and gives "" when no key has been
REM pressed, so this loop waits for a key (Y in the snapshot)
130 GET K$:IF K$="" THEN 130
140 PRINT "YOU PRESSED ";K$:REM YOU PRESSED Y
150 END
RUN
REM Typed directly, INPUT and GET are errors, so the next line prints
REM ?ILLEGAL DIRECT  ERROR, and the script stops
GET K$
