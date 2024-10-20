This file records my console commands. They might differ from the ChatGPT suggested ones.

## Basic Bootstrapping

Create the basic application folder:

```sh
cd ~/GitHub/LLMDesignedApp
mkdir go-cassandra-app
cd go-cassandra-app
```

As I'm using WSL, I can also use VSCode inside it, instead of `vim` or `nano`:

```sh
code .
```

Trying to run `go` the first time:

```sh
go
# -bash: go: command not found
```

Actually installing `go`:

```sh
wget https://go.dev/dl/go1.23.2.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.2.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
go version
# go version go1.23.2 linux/amd64
```

Creating the local development `go` setup and testing it that it works:

```sh
mkdir -p ~/go/{bin,src,pkg}
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
mkdir -p ~/go/src/hello
cd ~/go/src/hello
cat > main.go<< EOF
package main

import "fmt"

func main() {
    fmt.Println("Hello, Go!")
}
EOF
go run main.go
# Hello, Go!
```

Starting up Cassandra container:

```sh
cd ~/GitHub/LLMDesignedApp/go-cassandra-app
docker compose up -d
docker compose ps
# NAME        IMAGE              COMMAND                  SERVICE     CREATED          STATUS          PORTS
# cassandra   cassandra:latest   "docker-entrypoint.s…"   cassandra   10 minutes ago   Up 10 minutes   7000-7001/tcp, 7199/tcp, 9160/tcp, 0.0.0.0:9042->9042/tcp, :::9042->9042/tcp
```

Checking that the example `go` app works:

```sh
cd ~/GitHub/LLMDesignedApp/go-cassandra-app
go run main.go
# Existing keyspaces:
# system_auth
# system_schema
# system_distributed
# system
# system_traces
```
