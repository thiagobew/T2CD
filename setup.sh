#!/bin/sh
export LEADER_IP="localhost"
export NODES=3
export START_SERVER_PORT=11000
export START_RAFT_PORT=15000

go build cmd/main.go
mv main bin/main

go build client/main.go
mv main bin/client

chmod +x ./run.sh