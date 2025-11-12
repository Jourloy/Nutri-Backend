# Build stage
FROM golang:1.23.3-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o server ./cmd/server

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /build/server .

# Copy migrations and assets if needed
COPY --from=builder /build/migrations ./migrations
COPY --from=builder /build/assets ./assets

# Expose port
EXPOSE 3002

# Run the application
CMD ["./server"]
