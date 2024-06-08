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

### Install Atomix
- Install Go client
```
$ GO111MODULE=on go get github.com/atomix/go-sdk
$ go get github.com/atomix/go-sdk/pkg/atomix
```

### Installing Atomix runtime
- Install [Helm](https://helm.sh/docs/intro/install/)

```
curl https://baltocdn.com/helm/signing.asc | gpg --dearmor | sudo tee /usr/share/keyrings/helm.gpg > /dev/null
sudo apt-get install apt-transport-https --yes
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/helm.gpg] https://baltocdn.com/helm/stable/debian/ all main" | sudo tee /etc/apt/sources.list.d/helm-stable-debian.list
sudo apt-get update
sudo apt-get install helm
```

- Install Atomix runtime
```
helm repo add atomix https://charts.atomix.io
helm repo update
helm install -n kube-system atomix-runtime atomix/atomix-runtime --wait
```

- Deploying a data store (UNDER REVIEW)
```
> data-store.yaml
```
```yaml
apiVersion: sharedmemory.atomix.io/v1beta1
kind: SharedMemoryStore
metadata:
  name: my-data-store
spec: {}
```
obs.: Atomix runtime installation based on this [link](https://atomix.io/getting-started/#writing-an-application)

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
