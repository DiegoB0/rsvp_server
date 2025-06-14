build:
	@go build -o bin/rsvp_backend cmd/main.go

test:
	@go test -v ./...

run: build
	@./bin/rsvp_backend

build-worker:
	@go build -o bin/rsvp_worker cmd/worker/main.go

run-worker: build-worker
	@./bin/rsvp_worker

migrate-up:
	go run cmd/migrate/main.go up

migrate-down:
	go run cmd/migrate/main.go down

migrate-create:
	migrate create -ext sql -dir cmd/migrate/migrations $(name)

## Configs
APP_NAME := rsvp-backend
CONTAINER_NAME := $(APP_NAME)-container

## Docker Compose Commands
compose-build:
	docker-compose build

compose-up:
	docker-compose up -d

compose-down:
	docker-compose down

compose-logs:
	docker-compose logs -f

compose-restart:
	docker-compose down && docker-compose up -d

docker-dev: compose-build compose-up

docker-clean:
	docker-compose down --volumes --remove-orphans
	docker rmi $(APP_NAME):latest || true

