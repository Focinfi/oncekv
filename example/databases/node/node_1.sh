#!/usr/bin/env bash

rm -rf data/node_1

go run main.go 127.0.0.1:11000 127.0.0.1:12000 data/node_1 