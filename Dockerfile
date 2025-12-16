# =============================================================================
# Stage 1: Build the Go binary
# =============================================================================
# Using Alpine variant for smaller base image (~5MB vs ~800MB for full golang)
FROM golang:1.23-alpine AS builder

# Set working directory for the build
WORKDIR /src

# -----------------------------------------------------------------------------
# Layer caching optimization:
# Copy go.mod and go.sum FIRST, then download dependencies.
# This layer only rebuilds when dependencies change, not when source code changes.
# -----------------------------------------------------------------------------
COPY go.mod go.sum ./
RUN go mod download

# Now copy the rest of the source code
# This layer rebuilds whenever any source file changes
COPY . .

# -----------------------------------------------------------------------------
# Build a static binary:
# - CGO_ENABLED=0: Disable CGO for a fully static binary (no libc dependency)
# - GOOS=linux: Target Linux (container OS)
# - -ldflags="-s -w": Strip debug info for smaller binary
# -----------------------------------------------------------------------------
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /dockstart ./cmd/dockstart


# =============================================================================
# Stage 2: Minimal runtime image
# =============================================================================
# Final image is just Alpine (~5MB) + our binary (~5-10MB) = ~15MB total
# Compare to shipping golang:1.21 which is ~800MB!
FROM alpine:3.19

# Add CA certificates for HTTPS requests (needed if tool ever makes network calls)
RUN apk --no-cache add ca-certificates

# Copy ONLY the binary from the builder stage
# Everything else (Go toolchain, source code) is discarded
COPY --from=builder /dockstart /usr/local/bin/dockstart

# -----------------------------------------------------------------------------
# Security: Run as non-root user
# This follows the principle of least privilege.
# If the container is compromised, attacker has limited permissions.
# -----------------------------------------------------------------------------
RUN adduser -D -u 1000 dockstart
USER dockstart

# Set the binary as the entrypoint
# Arguments passed to `docker run` will be passed to dockstart
ENTRYPOINT ["dockstart"]
