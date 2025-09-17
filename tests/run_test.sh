#!/bin/bash

if [ $# -ne 1 ]; then
	echo "usage: run-test.sh [testType]"
	exit 100
fi

testMode="-$1"

docker compose -f "docker-compose${testMode}.yml" up --build -d
