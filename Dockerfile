# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o cloud-ddns .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata || true

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/cloud-ddns .

# Expose ports
EXPOSE 3495 8080

# Run the binary
CMD ["./cloud-ddns"]
