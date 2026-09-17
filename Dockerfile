# Build stage
FROM golang:1.25.0-alpine AS builder

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
# Basic Auth: pass JIRA_URL, JIRA_USERNAME and JIRA_API_TOKEN (see README).
# OAuth: this image is built without embedded credentials, so pass
# JIRA_OAUTH_CLIENT_ID and JIRA_OAUTH_CLIENT_SECRET at runtime, and mount a token
# file produced by running `jira-mcp auth` on the host.
ENTRYPOINT ["/jira-mcp"]
