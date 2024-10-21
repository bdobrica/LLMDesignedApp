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

## Filling in the blanks for User Management

```sh
cd ~/GitHub/LLMDesignedApp/user-management
go get golang.org/x/crypto/bcrypt
go get gopkg.in/gomail.v2
export SMTP_HOST=...
export SMTP_PORT=...
export SMTP_USERNAME=...
export SMTP_PASSWORD=...
export SMTP_SENDER_EMAIL=...
go run main.go
```

## Creating the Authentication Service

Creating and initializing the `auth-service`:

```sh
cd ~/GitHub/LLMDesignedApp/auth-service
go mod init github.com/bdobrica/LLMDesignedApp/auth-service
go get github.com/gofiber/fiber/v2
go get github.com/golang-jwt/jwt/v5
go get github.com/gocql/gocql
go get golang.org/x/crypto/bcrypt
```

### Creating the go-common package, to reuse password hashing

Creating the `go-common` package:

```sh
cd ~/GitHub/LLMDesignedApp
mkdir go-common
cd go-common
go mod init github.com/bdobrica/LLMDesignedApp/go-common
```

## Adding generate*RandomToken functions to go-common

No additional steps needed.

Updating the go-common package:

```sh
cd ~/GitHub/LLMDesignedApp/auth-service
go get github.com/bdobrica/LLMDesignedApp/go-common@bc40db16676062c93bfedfcd58628eb6e6344533
```

Creating the table for storing the tokens:

```sh
cd ~/GitHub/LLMDesignedApp/go-cassandra-app
cat ./schema.cql | docker compose exec -T cassandra cqlsh
```

Testing that I can generate a token:

```sh
cd ~/GitHub/LLMDesignedApp/user-management
go run .
curl -X POST http://localhost:3000/register \
     -H "Content-Type: application/json" \
     -d '{"username": "john_doe", "email": "john@example.com", "password": "password123"}' | jq
#   % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
#                                  Dload  Upload   Total   Spent    Left  Speed
# 100   301  100   221  100    80   3116   1128 --:--:-- --:--:-- --:--:--  4300
# {
#   "status": true,
#   "data": {
#     "id": "afd8b2aa-8fb3-11ef-b53f-00155d0889c3",
#     "username": "john_doe",
#     "email": "john@example.com",
#     "password": "password123",
#     "email_verified": false,
#     "verification_token": "c3ef23666142974bab96f6263dac6d1f"
#   }
# }
cd ~/GitHub/LLMDesignedApp/auth-service
go run .
curl -X POST http://localhost:3000/login \
     -H "Content-Type: application/json" \
     -d '{"username": "john_doe", "password": "password123"}' | jq
#   % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
#                                  Dload  Upload   Total   Spent    Left  Speed
# 100   361  100   310  100    51   4306    708 --:--:-- --:--:-- --:--:--  5084
# {
#   "data": {
#     "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3Mjk1MjAzMTAsInVzZXJfaWQiOiJhZmQ4YjJhYS04ZmIzLTExZWYtYjUzZi0wMDE1NWQwODg5YzMifQ.iKYJ2zhQyiNY689rAdN6J81M0b1EHMHgHjDSVtWyYkw",
#     "expires_in": 900,
#     "refresh_token": "s0pXQA2kXwVBhHfONWDHVnPyRAZukKaP"
#   },
#   "message": "Login successful",
#   "status": true
# }
```
