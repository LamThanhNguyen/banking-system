# banking-system
Be Banking System. Tech: Golang, Redis, K8s, Github Actions

## Setup local development

### Install tools

- [Migrate](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate)

    ```bash
    $ curl -L https://packagecloud.io/golang-migrate/migrate/gpgkey | apt-key add -
    $ echo "deb https://packagecloud.io/golang-migrate/migrate/ubuntu/ $(lsb_release -sc) main" > /etc/apt/sources.list.d/migrate.list
    $ apt-get update
    $ apt-get install -y migrate
    ```

- [Sqlc](https://github.com/kyleconroy/sqlc#installation)

    ```bash
    sudo snap install sqlc
    ```

- [Gomock](https://github.com/uber-go/mock)

    ```bash
    go install go.uber.org/mock/mockgen@latest
    export PATH=$PATH:$(go env GOPATH)/bin
    mockgen -version
    ```

### Setup infrastructure

- Create the bank-network

    ```bash
    make network
    ```

- Start postgres container:

    ```bash
    make postgres
    ```

- Create simple_bank database:

    ```bash
    make createdb
    ```

- Run db migration up all versions:

    ```bash
    make migrateup
    ```

- Run db migration up 1 version:

    ```bash
    make migrateup1
    ```

- Run db migration down all versions:

    ```bash
    make migratedown
    ```

- Run db migration down 1 version:

    ```bash
    make migratedown1
    ```

- Start the redis:
    ```bash
    make redis
    ```

- Create the .env file and fill in the information:
    ```bash
    ENVIRONMENT=development
    ALLOWED_ORIGINS=
    DB_SOURCE=
    MIGRATION_URL=
    REDIS_ADDRESS=
    HTTP_SERVER_ADDRESS=
    TOKEN_SYMMETRIC_KEY=
    ACCESS_TOKEN_DURATION=15m
    REFRESH_TOKEN_DURATION=24h
    EMAIL_SENDER_NAME=
    EMAIL_SENDER_ADDRESS=
    EMAIL_SENDER_PASSWORD=
    ```

### How to generate code

- Generate SQL CRUD with sqlc:

    ```bash
    make sqlc
    ```

- Create a new db migration:

  ```bash
  make new_migration name=<migration_name>
  ```

- Init Go module

    ```bash
    go mod init github.com/LamThanhNguyen/banking-system
    ```

- Install package

    ```
    go get github.com/some/library
    ```

- Add module requirements and sums

    ```
    go mod tidy
    ```

- Generate DB mock with gomock:

  ```bash
  make mock
  ```

### How to run

- Run server:

  ```bash
  make server
  ```

- Run test:

  ```bash
  make test
  ```

## Authorization & Access Control

The API uses **Casbin v2** backed by Postgres (via a custom pgx adapter) to implement a layered model that combines:

|  Model    |            Used for            |                  Example                    |
|:---------:|:------------------------------:|:-------------------------------------------:|
|  **RBAC** | Role‑based default permissions | banker → accounts:create                    |
|  **ACL**  | One‑off user overrides         | audit-bot → accounts:read                   |
|  **ABAC** | Attribute rules                | Depositor can update their own profile only |

## API Documentations
```
    go install github.com/swaggo/swag/cmd/swag@latest
    echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.bashrc
    source ~/.bashrc
    go get -u github.com/swaggo/gin-swagger
    go get -u github.com/swaggo/files
    swag init -g main.go --output docs
    http://localhost:8080/swagger/index.html
```

## Docker Container
```
    chmod +x start.sh
    docker build -t banking-system:latest .
    docker run --name banking-system --network bank-network -p 8080:8080 banking-system:latest
    docker run --name banking-system --network bank-network -p 8080:8080 -e GIN_MODE=release -e PARAM=VALUE banking-system:latest
    docker compose build
    docker compose up
    docker compose down
    docker ps
    docker rm {container-name}
    docker rmi {iamge-id}
    docker container inspect {container-name}
    docker network create {network-name}
    docker network connect {network-name} {container-name}
    docker network ls
    docker network inspect {network-name}
    docker stop $(docker ps -a -q)
    docker rm -f $(docker ps -a -q)
    docker rmi -f $(docker images -aq)
```

## Linter: golangci-lint
```
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.1.6
    echo 'export PATH="$PATH:$HOME/go/bin"' >> ~/.bashrc
    source ~/.bashrc
    golangci-lint --version
    golangci-lint run
```

## AWS Secrets Manager
```
openssl rand -hex 64
openssl rand -hex 64 | head -c 32
```