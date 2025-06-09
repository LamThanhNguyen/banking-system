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

build-container:
	docker build -t banking-system:latest .

run-container:
	docker run --name banking-system --network bank-network -p 8080:8080 banking-system:latest

run-compose:
	docker compose up

.PHONY: network postgres createdb dropdb migrateup migrateup1 migratedown migratedown1 new_migration sqlc test build server mock redis build-container run-container run-compose
