# Test Helpers
This directory contains test helper files for the project.
Helpers in this directory are intended to assist writing and running various tests, such as providing mock implementations, common setup/teardown logic, or utilities for integration testing.

## Tutorial

### Test indexer locally

1. edit .env file
2. edit docker-compose-*.yml if needed
3. (Optional) create log config file
4. run run_test.sh to start the test
5. run stop_test.sh to stop the test
6run clean_test.sh to clean up the test

### edit .env file

You should create .env file to run the test.
Refer to .env.example for the format of .env file.

### edit docker-compose-*.yml if needed

You need not edit docker-compose-*.yml unless you want to run the test with very specific configurations.

### (Optional) create log config file

You can create a log config file `arglog.toml` to change the log output configuration.
The created `arglog.toml` file should be placed in the CWD, and will override the default log configuration.

### run run_test.sh to start the test

use `run_test.sh db_only` to start only the elsaticsearch.
use `run_test.sh no_indexer` to start without the indexer.
use `run_test.sh all` to start everything including indexer.

### run stop_test.sh to stop the test

This just stops containers
Running `clear_test.sh <testMode>` will remove all containers.


### docker-compose-all.yml
A Docker Compose configuration to run the whole Aergoscan stack locally for integration testing, including all core components and dependencies. 

[//]: # (### docker-compose.yml)

[//]: # (A Docker Compose configuration to run the minimal requirement, including indexer and elasticsearch.)

### docker-compose-no_indexer.yml
A Docker Compose configuration to run all components except the indexer.
This is useful for developers who need to debug the indexer outside of Docker, while still running the rest of the stack in containers.

### docker-compose-db_only.yml
A Docker Compose configuration to run elasticsearch only.
This is useful for developers who need to debug the indexer outside of Docker, like no_indexer.

## Helper Scripts


### run_test.sh

Starts all containers defined in **docker-compose.yml**. Use this script to launch the full Aergoscan stack for integration or end-to-end testing.

### stop_test.sh

Stops all containers started by **docker-compose.yml**. Use this script to gracefully shut down all test containers. 

### clean_test.sh

Removes all containers and associated resources created by **docker-compose.yml**. Use this after stopping containers to clean up the test environment.

