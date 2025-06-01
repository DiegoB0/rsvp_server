FROM golang:1.23.0-alpine3.19 AS builder

RUN apk add --no-cache git
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o app ./cmd/main.go

FROM alpine:3.19.1

WORKDIR /root/

COPY --from=builder /app/app .

EXPOSE 8080

CMD ["./app"]

