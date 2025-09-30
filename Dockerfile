# syntax=docker/dockerfile:1

# 1) Build binary
FROM golang:1.23-alpine AS builder
WORKDIR /src

ENV CGO_ENABLED=0

# Cache deps first
COPY go.mod go.sum ./
RUN go mod download

# Copy sources
COPY . .

# Build linux/amd64 static binary named `server`
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /server ./cmd/server

# 2) Get CA certs
FROM alpine:3.20 AS certs
RUN apk --no-cache add ca-certificates

# 3) Minimal runtime
FROM scratch
WORKDIR /

COPY --from=builder /server /server
COPY --from=builder /src/.env /.env
COPY --from=builder /src/migrations /migrations
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# EXPOSE 3001

ENTRYPOINT ["/server"]
