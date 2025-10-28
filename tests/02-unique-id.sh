#! /bin/bash

MAELSTROM_BIN=~/maelstrom/maelstrom
NODE_BIN=./bin/unique-id

go build -o ./bin/unique-id ./cmd/unique-id

$MAELSTROM_BIN test -w unique-ids --bin $NODE_BIN --time-limit 30 --rate 1000 --node-count 3 --availability total --nemesis partition