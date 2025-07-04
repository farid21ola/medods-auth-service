# --- Build stage ---
FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o auth-service ./cmd/auth-service

# --- Run stage ---
FROM alpine:latest

WORKDIR /app

# Установим сертификаты и dockerize для ожидания БД
RUN apk add --no-cache ca-certificates wget \
    && wget -O /usr/local/bin/dockerize https://github.com/jwilder/dockerize/releases/download/v0.6.1/dockerize-alpine-linux-amd64-v0.6.1.tar.gz \
    && tar -C /usr/local/bin -xzvf /usr/local/bin/dockerize \
    && chmod +x /usr/local/bin/dockerize

COPY --from=builder /app/auth-service /app/auth-service

EXPOSE 8081

CMD ["/usr/local/bin/dockerize", "-wait", "tcp://db:5432", "-timeout", "60s", "./auth-service"] 