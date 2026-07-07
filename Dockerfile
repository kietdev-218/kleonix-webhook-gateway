FROM golang:1.26.4-alpine AS builder

# Install necessary dependencies
RUN apk add --no-cache git ca-certificates tzdata
# Create a non-root user for security
RUN adduser -D -g '' appuser

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build the binary with stripping (-s -w) to significantly reduce the size
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o webhook-gateway ./cmd/gateway/main.go

# Use scratch as the final base image for the absolute smallest surface area and maximum security
FROM scratch

# Import the user and group files from the builder
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Import the CA certificates (if making HTTPS calls) and timezone data
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the statically linked executable
COPY --from=builder /app/webhook-gateway /webhook-gateway

# Run as the unprivileged user
USER appuser:appuser

EXPOSE 3000
ENTRYPOINT ["/webhook-gateway"]
