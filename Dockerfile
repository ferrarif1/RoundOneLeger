# syntax=docker/dockerfile:1
FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o server ./cmd/server

FROM alpine:3.18

WORKDIR /app

COPY --from=builder /app/server /usr/local/bin/server

EXPOSE 8080

CMD ["server"]
