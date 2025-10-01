FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download || true

COPY . .
RUN go build -o server ./cmd/server

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/server /usr/local/bin/server
COPY migrations ./migrations
COPY openapi.yaml ./openapi.yaml

ENV GIN_MODE=release
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/server"]
