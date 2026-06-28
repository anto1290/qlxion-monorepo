# Build stage
FROM golang:1.25.0-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY services/auth-service/go.mod services/auth-service/go.sum ./
COPY pkg/ ../pkg/

# Download dependencies
RUN go mod download

# Copy source code
COPY services/auth-service/ .
COPY pkg/ ../pkg/

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o auth cmd/auth/main.go

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=builder /app/auth .

# Copy migrations
COPY services/auth-service/migrations ./migrations

# Expose port
EXPOSE 8001

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD wget -qO- http://localhost:8001/health || exit 1

# Run
CMD ["./auth"]
