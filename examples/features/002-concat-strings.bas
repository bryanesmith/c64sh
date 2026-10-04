#!/usr/bin/env c64sh
REM Joining strings with ";", with "+", or with nothing between them
REM ";" prints the next item right after the previous one
PRINT "HELLO ";"WORLD":REM HELLO WORLD
PRINT "HEL";"LO ";"WOR";"LD":REM HELLO WORLD
REM "+" joins two strings into one
PRINT "HELLO "+"WORLD":REM HELLO WORLD
PRINT "C"+"6"+"4":REM C64
REM Strings side by side are joined too, with or without a space
PRINT "HELLO ""WORLD":REM HELLO WORLD
PRINT "HELLO " "WORLD":REM HELLO WORLD
REM The styles can be mixed on one line
PRINT "A"+"B";"C" "D":REM ABCD
REM Nothing is added between joined strings, so include spaces yourself
PRINT "HELLO";"WORLD":REM HELLOWORLD
PRINT "HELLO"+" "+"WORLD":REM HELLO WORLD
REM Joining empty strings changes nothing
PRINT ""+"C64"+"":REM C64
PRINT "";"":REM (an empty line)
