# T2CD

Distributed fault-tolerant tuple space implementation written in Go for the INE5418 - Distributed Computing course from the Computer Science program of the Federal University of Santa Catarina (UFSC)

## Pre-requisites

### Install Golang (version >= 1.19):
- Download the desired version for your OS/architecture from [Golang Downloads](https://golang.org/dl/)
- Decompress and install (Linux):

```
$ sudo tar -xvf <go-downloads-file>.tar.gz -C /usr/local
```

- Add Go to PATH:

```
$ echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.profile
$ source ~/.profile 
```

- Logout for the changes to take effect

**Note**: if there's a previously install Go version (i. e. using your distro's package manager) that doesn't fit the requirements, uninstall it, otherwise your enviroment will likely prefer that version over the manually installed one

## Before running 

- Get dependencies with (run at project directory):
```
$ go mod tidy
```
- Run any `go get <url>` recommended by the command

- Execute setup script
```
$ chmod +x setup.sh
$ source setup.sh
``` 

### Configuration

- All configuration is done in the `setup.sh` file. There are 4 environment variables:
    1. `LEADER_IP`: IP address of the first leader of the Raft cluster;
    2. `NODES`: number of nodes created by the `run.sh`;
    3. `START_SERVER_PORT`: TCP port in which the service will answer requests. Leader starts in this address, and nodes in a same machine should be created with this port more spaced out, since the service creates auxialiary ports from this base port to answer to the clients;
    4. `START_RAFT_PORT`: TCP port to comunicate with other nodes in the Raft cluster.

- To create one node separately (i. e., in another machine) once the first leader is running, run:
```
$ source setup.sh
$ ./bin/main ./bin/main -haddr "<node_ip_address>:$START_SERVER_PORT" -raddr "<node_ip_address>:$START_RAFT_PORT" -id <node_id> -join "$LEADER_IP:$START_SERVER_PORT" ./nodes/<node_id>
```

## Run
- To start the service:
```
$ ./run.sh
```

- To run client and test the application:
```
$ ./bin/client -address $LEADER_IP -port $START_SERVER_PORT
```
