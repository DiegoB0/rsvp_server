FROM golang:1.23.0-alpine3.19 AS builder

RUN apk add --no-cache git
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Allow switching entrypoint source (main or worker)
ARG BUILD_TARGET=cmd/main.go
RUN go build -o app $BUILD_TARGET

FROM alpine:3.19.1

WORKDIR /root/

COPY --from=builder /app/app .
COPY --from=builder /app/assets ./assets

EXPOSE 8080


CMD ["./app"]

