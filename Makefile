build:
	@go build -o bin/rspv_backend cmd/main.go

test:
	@go test -v ./...

run: build
	@./bin/rspv_backend
