# Banking System

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A modern banking system built with **Golang**, **PostgreSQL**, **Redis**, **Kubernetes**, and **GitHub Actions**.  
Implements robust authentication, authorization (RBAC, ACL, ABAC via Casbin), and scalable infrastructure.

---

## Table of Contents

- [Features](#features)
- [Tech Stack](#tech-stack)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
  - [Environment Variables](#environment-variables)
  - [Database & Infrastructure](#database--infrastructure)
  - [Code Generation](#code-generation)
  - [Running the Application](#running-the-application)
  - [Testing](#testing)
- [Authorization & Access Control](#authorization--access-control)
- [API Documentation](#api-documentation)
- [Docker Usage](#docker-usage)
- [Linting](#linting)
- [Contributing](#contributing)
- [License](#license)
- [Contact](#contact)

---

## Features

- User registration and authentication (JWT)
- Role-based, attribute-based, and access control list authorization (Casbin)
- Account management, transfers, and transaction history
- RESTful API with Swagger documentation
- Database migrations and SQL code generation
- Redis caching
- CI/CD with GitHub Actions
- Docker and Kubernetes ready

---

## Tech Stack

- **Backend:** Golang
- **Database:** PostgreSQL
- **Cache:** Redis
- **Authorization:** Casbin v2 (RBAC, ACL, ABAC)
- **Migrations:** golang-migrate
- **SQL Generation:** sqlc
- **Testing & Mocking:** GoMock
- **API Docs:** Swagger (swaggo)
- **CI/CD:** GitHub Actions
- **Containerization:** Docker, Docker Compose, Kubernetes

---

## Getting Started

### Prerequisites

- [Go](https://golang.org/doc/install) >= 1.18
- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/)
- [Make](https://www.gnu.org/software/make/)
- [PostgreSQL](https://www.postgresql.org/)
- [Redis](https://redis.io/)

### Installation

#### Install Required Tools

- **Migrate** ([docs](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate)):
    ```bash
    curl -L https://packagecloud.io/golang-migrate/migrate/gpgkey | sudo apt-key add -
    echo "deb https://packagecloud.io/golang-migrate/migrate/ubuntu/ $(lsb_release -sc) main" | sudo tee /etc/apt/sources.list.d/migrate.list
    sudo apt-get update
    sudo apt-get install -y migrate
    ```

- **Sqlc** ([docs](https://github.com/kyleconroy/sqlc#installation)):
    ```bash
    sudo snap install sqlc
    ```

- **GoMock** ([docs](https://github.com/uber-go/mock)):
    ```bash
    go install go.uber.org/mock/mockgen@latest
    export PATH=$PATH:$(go env GOPATH)/bin
    mockgen -version
    ```

### Environment Variables

Create a `.env` file in the project root and fill in the following:

```env
ENVIRONMENT=develop
ALLOWED_ORIGINS=["http://localhost:3000"]
DB_SOURCE=postgresql://{{username}}:{{password}}@postgres:5432/{{database_name}}?sslmode=disable
MIGRATION_URL=file://db/migration
REDIS_ADDRESS=redis:6379
HTTP_SERVER_ADDRESS=0.0.0.0:8080
TOKEN_SYMMETRIC_KEY=2e3c226355a0770689c808684fbdca40
ACCESS_TOKEN_DURATION=15m
REFRESH_TOKEN_DURATION=24h
EMAIL_SENDER_NAME=
EMAIL_SENDER_ADDRESS=
EMAIL_SENDER_PASSWORD=
FRONTEND_DOMAIN=http://localhost:3000
```

### Database & Infrastructure

- **Create Docker network:**
    ```bash
    make network
    ```

- **Start PostgreSQL:**
    ```bash
    make postgres
    ```

- **Create database:**
    ```bash
    make createdb
    ```

- **Run migrations:**
    ```bash
    make migrateup      # Up all versions
    make migrateup1     # Up 1 version
    make migratedown    # Down all versions
    make migratedown1   # Down 1 version
    ```

- **Start Redis:**
    ```bash
    make redis
    ```

### Code Generation

- **Generate SQL CRUD with sqlc:**
    ```bash
    make sqlc
    ```

- **Create a new DB migration:**
    ```bash
    make new_migration name=<migration_name>
    ```

- **Initialize Go module:**
    ```bash
    go mod init github.com/LamThanhNguyen/banking-system
    ```

- **Install Go packages:**
    ```bash
    go get github.com/some/library
    go mod tidy
    ```

- **Generate DB mocks with GoMock:**
    ```bash
    make mock
    ```

### Running the Application

- **Run server:**
    ```bash
    make server
    ```

### Testing

- **Run tests:**
    ```bash
    make test
    ```

---

## Authorization & Access Control

The API uses **Casbin v2** (Postgres-backed) to implement a layered model:

|  Model    |            Used for            |                  Example                    |
|:---------:|:------------------------------:|:-------------------------------------------:|
|  **RBAC** | Role‑based default permissions | banker → accounts:create                    |
|  **ACL**  | One‑off user overrides         | audit-bot → accounts:read                   |
|  **ABAC** | Attribute rules                | Depositor can update their own profile only |

---

## API Documentation

- **Generate Swagger docs:**
    ```bash
    go install github.com/swaggo/swag/cmd/swag@latest
    echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.bashrc
    source ~/.bashrc
    go get -u github.com/swaggo/gin-swagger
    go get -u github.com/swaggo/files
    swag init -g main.go --output docs
    ```
- **View docs:**  
  Visit [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html) after running the server.

---

## Docker Usage

- **Build and run:**
    ```bash
    chmod +x start.sh
    docker build -t banking-system:latest .
    docker run --name banking-system --network bank-network -p 8080:8080 banking-system:latest
    ```

- **Run with environment variables:**
    ```bash
    docker run --name banking-system --network bank-network -p 8080:8080 -e GIN_MODE=release -e PARAM=VALUE banking-system:latest
    ```

- **Docker Compose:**
    ```bash
    docker compose build
    docker compose up
    docker compose down
    ```

- **Useful Docker commands:**
    ```bash
    docker ps
    docker rm {container-name}
    docker rmi {image-id}
    docker container inspect {container-name}
    docker network create {network-name}
    docker network connect {network-name} {container-name}
    docker network ls
    docker network inspect {network-name}
    docker stop $(docker ps -a -q)
    docker rm -f $(docker ps -a -q)
    docker rmi -f $(docker images -aq)
    ```

---

## Linting

- **Install golangci-lint:**
    ```bash
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.1.6
    echo 'export PATH="$PATH:$HOME/go/bin"' >> ~/.bashrc
    source ~/.bashrc
    ```

- **Run linter:**
    ```bash
    golangci-lint --version
    golangci-lint run
    ```

---

## Contributing

Contributions are welcome! Please open issues or submit pull requests for improvements and bug fixes.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/YourFeature`)
3. Commit your changes (`git commit -am 'Add some feature'`)
4. Push to the branch (`git push origin feature/YourFeature`)
5. Open a pull request

---

## License

This project is licensed under the [MIT License](LICENSE).

---

## Contact

Maintainer: [LamThanhNguyen](https://github.com/LamThanhNguyen)  
For questions, please open an issue or contact via GitHub.

---