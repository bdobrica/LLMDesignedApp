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

## Basic User Management

Connecting to Cassandra and checking the `users` table, also reseting it after each test:

```sh
cd ~/GitHub/LLMDesignedApp/go-cassandra-app
docker compose exec cassandra cqlsh
cqlsh> use user_management;
cqlsh:user_management> select * from users;
cqlsh:user_management> truncate users;
```

Adding new user (I use `jq` for formatting the output):

```sh
curl -X POST http://localhost:3000/register \
     -H "Content-Type: application/json" \
     -d '{"username":"testuser","email":"email@example.com","password":"password123"}' | jq
#   % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
#                                  Dload  Upload   Total   Spent    Left  Speed
# 100   308  100   227  100    81  13867   4948 --:--:-- --:--:-- --:--:-- 19250
# {
#   "status": true,
#   "data": {
#     "id": "3f052c15-8f0d-11ef-9125-00155d0889c3",
#     "username": "testuser",
#     "email": "email@example.com",
#     "password": "password123",
#     "email_verified": false,
#     "verification_token": "<token>"
#   }
# }
```

Verifying the user:
```sh
curl 'http://localhost:3000/verify/<token>'
# {
#   "status": true,
#   "message": "Email successfully verified"
# }
```

Recover password for user:
```sh
curl -X POST 'http://localhost:3000/recover' \
     -H "Content-Type: application/json" \
     -d '{"email":"email@example.com"}' | jq
#   % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
#                                  Dload  Upload   Total   Spent    Left  Speed
# 100   102  100    69  100    33   8745   4182 --:--:-- --:--:-- --:--:-- 14571
# {
#   "status": true,
#   "message": "Password recovery email sent successfully"
# }
```

Reset password for user:
```sh
curl -X POST 'http://localhost:3000/reset/<token>' \
     -H "Content-Type: application/json" \
     -d '{"password":"newpassword"}' | jq
#   % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
#                                  Dload  Upload   Total   Spent    Left  Speed
# 100    81  100    55  100    26   5641   2666 --:--:-- --:--:-- --:--:--  9000
# {
#   "status": true,
#   "message": "Password successfully reset"
# }
```
