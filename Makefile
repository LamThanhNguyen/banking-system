DB_URL=postgresql://root:secret@localhost:5432/banking_system?sslmode=disable

network:
	docker network create bank-network

postgres:
	docker run --name postgres --network bank-network -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:17-alpine

createdb:
	docker exec -it postgres createdb --username=root --owner=root banking_system

dropdb:
	docker exec -it postgres dropdb banking_system

migrateup:
	migrate -path db/migration -database "$(DB_URL)" -verbose up

migrateup1:
	migrate -path db/migration -database "$(DB_URL)" -verbose up 1

migratedown:
	migrate -path db/migration -database "$(DB_URL)" -verbose down

migratedown1:
	migrate -path db/migration -database "$(DB_URL)" -verbose down 1

new_migration:
	migrate create -ext sql -dir db/migration -seq $(name)

sqlc:
	sqlc generate

test:
	go test -v -cover -short ./...

build:
	go build -o main main.go

server:
	swag init -g main.go --output docs
	go run main.go

mock:
	mockgen -package mockdb -destination db/mock/store.go github.com/LamThanhNguyen/banking-system/db/sqlc Store
	mockgen -package mockwk -destination worker/mock/distributor.go github.com/LamThanhNguyen/banking-system/worker TaskDistributor

redis:
	docker run --name redis --network bank-network -p 6379:6379 -d redis:7-alpine

build-container-local:
	docker build -t banking-system:local -f Dockerfile.local .

run-container-local:
	docker run --rm --name banking-system-local \
	--network bank-network \
	-p 8080:8080 \
    -e DB_SOURCE=postgresql://root:secret@postgres:5432/banking_system?sslmode=disable \
    -e REDIS_ADDRESS=redis:6379 \
    banking-system:local

run-compose-local:
	docker compose -f docker-compose-local.yaml up --build

stop-compose-local:
	docker compose -f docker-compose-local.yaml down

.PHONY: network postgres createdb dropdb migrateup migrateup1 migratedown migratedown1 new_migration sqlc test build server mock redis build-container-local run-container-local run-compose-local stop-compose-local
