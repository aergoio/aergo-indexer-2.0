#!/bin/bash

if [ $# -ne 1 ]; then
	echo "usage: stop-test.sh [testType]"
	exit 100
fi

testMode="-$1"

docker compose -f docker-compose${testMode}.yml stop
