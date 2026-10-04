#!/usr/bin/env c64sh
REM Comments with REM
REM A line starting with REM is ignored, whatever follows it
REM PRINT "THIS IS NOT PRINTED"
REM A comment can follow a statement, after a colon
PRINT "SHOWN":REM SHOWN
REM A comment runs to the end of the line, past colons and keywords
REM THIS:PRINT "IS NOT PRINTED EITHER"
PRINT "BEFORE":REM BEFORE (and this :PRINT "X" is part of the comment)
REM No space is needed after REM: the next line is a comment
REMARKS LIKE THIS ARE COMMENTS TOO
REM Quotes inside a comment are fine: "HELLO", or an unclosed "HI
REM Inside quotes, REM is ordinary text
PRINT "REM":REM REM
PRINT "REMEMBER":REM REMEMBER
REM A comment can be empty; the next line is one
REM
PRINT "DONE":REM DONE
