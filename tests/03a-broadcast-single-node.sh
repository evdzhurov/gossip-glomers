#! /bin/bash

MAELSTROM_BIN=~/maelstrom/maelstrom
NODE_BIN=./bin/broadcast

go build -o ./bin/broadcast ./cmd/broadcast

$MAELSTROM_BIN test -w broadcast --bin $NODE_BIN --node-count 1 --time-limit 20 --rate 10