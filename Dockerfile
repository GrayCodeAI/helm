# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always) -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o helm \
    ./cmd/helm

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates sqlite-libs

# Create non-root user
RUN addgroup -g 1000 -S helm && \
    adduser -u 1000 -S helm -G helm

# Create directories
RUN mkdir -p /data /config && \
    chown -R helm:helm /data /config

# Copy binary from builder
COPY --from=builder /build/helm /usr/local/bin/helm

# Set permissions
RUN chmod +x /usr/local/bin/helm

# Switch to non-root user
USER helm

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health/live || exit 1

# Set environment
ENV HELM_DATA_DIR=/data
ENV HELM_CONFIG_DIR=/config

# Volume mounts
VOLUME ["/data", "/config"]

# Entry point
ENTRYPOINT ["helm"]
CMD ["serve"]
