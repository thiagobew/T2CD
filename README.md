# T2CD

Distributed fault-tolerant tuple space implementation written in Go for the INE5418 - Distributed Computing course from the Computer Science program of the Federal University of Santa Catarina (UFSC)

## Pre-requisites

Install Golang (version >= 1.19):
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

Get dependencies with (run at project directory):
```
$ go mod tidy
```

## Run
To compile and run, execute:
```
go run cmd/main.go
```