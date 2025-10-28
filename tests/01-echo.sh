#! /bin/bash

MAELSTROM_BIN=~/maelstrom/maelstrom
ECHO_BIN=./bin/echo

$MAELSTROM_BIN test -w echo --bin $ECHO_BIN --node-count 1 --time-limit 10