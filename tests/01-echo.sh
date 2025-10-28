#! /bin/bash

MAELSTROM_BIN=~/maelstrom/maelstrom
NODE_BIN=./bin/echo

$MAELSTROM_BIN test -w echo --bin $NODE_BIN --node-count 1 --time-limit 10