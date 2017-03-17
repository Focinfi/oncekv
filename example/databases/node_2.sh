#!/usr/bin/env bash

rm -rf data/node_2

oncekv -haddr :11001 -raddr :12001 -join :11000 data/node_2