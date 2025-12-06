ARG GO_BASE_IMAGE=mirror.gcr.io/library/golang:1.24-alpine
ARG RUNTIME_BASE_IMAGE=mirror.gcr.io/library/alpine:3.20
ARG NODE_BASE_IMAGE=mirror.gcr.io/library/node:18-alpine

# Build frontend
FROM ${NODE_BASE_IMAGE} AS frontend
ENV ROLLUP_DISABLE_NATIVE=1
WORKDIR /web
COPY web/package*.json ./
RUN npm install --prefer-offline --no-audit --progress=false --no-optional
# Workaround rollup optional native binary resolution on musl
RUN npm install @rollup/rollup-linux-x64-musl --no-audit --progress=false --no-optional || true
COPY web .
RUN npm run build

# Build backend (embed frontend)
FROM ${GO_BASE_IMAGE} AS builder

ENV GOPROXY=https://goproxy.cn,direct \
    GOCACHE=/tmp/.gocache

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download || true

COPY . .
COPY --from=frontend /web/dist ./webembed/dist
RUN go build -o server ./cmd/server

FROM ${RUNTIME_BASE_IMAGE} AS app
RUN apk add --no-cache postgresql-client

WORKDIR /app

COPY --from=builder /app/server /usr/local/bin/server
COPY migrations ./migrations
COPY openapi.yaml ./openapi.yaml

ENV GIN_MODE=release
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/server"]
