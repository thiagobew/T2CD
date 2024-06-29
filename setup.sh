#!/bin/sh

chmod +x environment.sh
source ./environment.sh

go build cmd/main.go
mv main bin/main

go build client/main.go
mv main bin/client