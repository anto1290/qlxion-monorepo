# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY api-gateway/go.mod api-gateway/go.sum ./
COPY pkg/ ../pkg/

# Download dependencies
RUN go mod download

# Copy source code
COPY api-gateway/ .
COPY pkg/ ../pkg/

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gateway cmd/gateway/main.go

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=builder /app/gateway .

# Copy config file
COPY api-gateway/gateway.yaml .

# Expose port
EXPOSE 8000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:8000/health || exit 1

# Run
CMD ["./gateway"]
