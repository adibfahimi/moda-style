# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install ca-certificates in builder for copying to scratch
RUN apk add --no-cache git ca-certificates

# Copy go workspace files and all referenced modules
COPY go.work go.work.sum* ./
COPY go.mod* ./
COPY common/ ./common/
COPY services/order-service/ ./services/order-service/
COPY services/auth-service/go.mod ./services/auth-service/go.mod
COPY services/product-service/go.mod ./services/product-service/go.mod
COPY services/cart-service/go.mod ./services/cart-service/go.mod
COPY services/admin-service/go.mod ./services/admin-service/go.mod

# Download dependencies from order-service directory
WORKDIR /app/services/order-service
RUN go mod download

# Build the service as a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o main .

# Run stage
FROM scratch

WORKDIR /app

# Copy ca-certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary from builder
COPY --from=builder /app/services/order-service/main .

EXPOSE 8005

CMD ["./main"]
