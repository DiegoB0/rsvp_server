build:
	@go build -o bin/rspv_backend cmd/main.go

test:
	@go test -v ./...

run: build
	@./bin/rspv_backend

migrate-up:
	go run cmd/migrate/main.go up

migrate-down:
	go run cmd/migrate/main.go down

migrate-create:
	migrate create -ext sql -dir cmd/migrate/migrations $(name)



