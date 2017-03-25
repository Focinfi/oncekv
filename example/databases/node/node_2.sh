#!/usr/bin/env bash

rm -rf data/node_2

go run main.go 127.0.0.1:11001 127.0.0.1:12001 data/node_2