# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install ca-certificates in builder for copying to scratch
RUN apk add --no-cache git ca-certificates

# Copy go workspace files and all referenced modules
COPY go.work go.work.sum* ./
COPY go.mod* ./
COPY common/ ./common/
COPY services/auth-service/go.mod ./services/auth-service/go.mod
COPY services/product-service/go.mod ./services/product-service/go.mod
COPY services/cart-service/go.mod ./services/cart-service/go.mod
COPY services/order-service/go.mod ./services/order-service/go.mod
COPY services/admin-service/ ./services/admin-service/

# Download dependencies from admin-service directory
WORKDIR /app/services/admin-service
RUN go mod download

# Build the application as a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o main .

# Final stage
FROM scratch

WORKDIR /app

# Copy ca-certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary from builder
COPY --from=builder /app/services/admin-service/main .

# Expose port
EXPOSE 8004

# Run the application
CMD ["./main"]
