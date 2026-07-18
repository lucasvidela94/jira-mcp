# Build stage
FROM golang:1.24.0-alpine AS builder

WORKDIR /app

# Copy dependency files first for better layer caching.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the static binary.
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" -o jira-mcp .

# Runtime stage
FROM scratch

# Copy CA certificates so HTTPS requests to Jira Cloud succeed.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary.
COPY --from=builder /app/jira-mcp /jira-mcp

# The binary exposes stdio by default; override with --transport http if desired.
ENTRYPOINT ["/jira-mcp"]
