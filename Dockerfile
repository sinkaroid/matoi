FROM golang:1.26.4-alpine AS builder

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o matoi main.go

# Minimal run stage
FROM alpine:latest

WORKDIR /app

# Copy the built binary
COPY --from=builder /app/matoi .

# Expose port
EXPOSE 3000

# Set entrypoint
ENTRYPOINT ["./matoi"]
