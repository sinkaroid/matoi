# Stage 1: Build the hyper-lightweight Go memory API
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY memapi/ .
# Initialize go module since we are compiling a directory package
RUN go mod init memapi
# Compile to a static binary without debug info to make it tiny (~2MB)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o memapi .

# Stage 2: FlareSolverr
FROM ghcr.io/flaresolverr/flaresolverr:latest

# Copy the compiled Go binary
COPY --from=builder /app/memapi /app/memapi

# Environment variables to optimize for Matoi
ENV LOG_LEVEL=info
ENV LOG_HTML=false
ENV HEADLESS=true

# Expose standard FlareSolverr port and our new Memory API port
EXPOSE 8191 8192

# Run the Go memory API in the background, then execute the original FlareSolverr entrypoint/cmd
# The original FlareSolverr runs: /usr/bin/dumb-init -- xvfb-run -s "-screen 0 1920x1080x24 -cc 4 -nolisten tcp -dpi 96 +extension RANDR" python -u /app/src/flaresolverr.py
CMD ["sh", "-c", "/app/memapi & exec /usr/bin/dumb-init -- xvfb-run -s \"-screen 0 1920x1080x24 -cc 4 -nolisten tcp -dpi 96 +extension RANDR\" python -u /app/src/flaresolverr.py"]
