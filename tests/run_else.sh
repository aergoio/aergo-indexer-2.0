#!/bin/zsh

SCAN_API_HOST=localhost:3000 AERGO_NODE=192.168.0.104:17845 docker compose -f without-indexer.yml up --build -d
