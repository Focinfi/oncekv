#!/usr/bin/env bash

rm -rf data/node_3
oncekv -haddr :11002 -raddr :12002 -join :11000 data/node_3