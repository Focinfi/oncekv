#!/usr/bin/env bash

rm -rf data/node_3
go run main.go -haddr :11002 -raddr :12002 -join :11000 data/node_3