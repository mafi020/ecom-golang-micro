# # ── STAGE 1: BUILD THE BINARY ──
# FROM golang:1.25-alpine AS builder

# WORKDIR /app

# # Install system certificates needed for secure communication
# RUN apk --no-cache add ca-certificates

# # Copy and download module dependencies first
# COPY go.mod go.sum ./
# RUN go mod download

# # Copy the entire codebase
# COPY . .

# # Pass the target binary via build arguments
# ARG SERVICE_PATH

# # OPTIMIZATION: Added --mount flags to reuse Go build and module caches
# RUN --mount=type=cache,target=/root/.cache/go-build \
#     --mount=type=cache,target=/go/pkg/mod \
#     CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /main ./cmd/${SERVICE_PATH}

# # ── STAGE 2: RUNTIME SCRATCH CONTAINER ──
# FROM alpine:3.19 AS final

# WORKDIR /

# # Copy trusted certificates and compiled binary from the builder layer
# COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# COPY --from=builder /main /main

# # Copy migrations folder
# COPY --from=builder /app/migrations /migrations

# ENTRYPOINT ["/main"]







# ── STAGE 1: SYSTEM DEPENDENCIES & BASE CONFIG ──
FROM golang:1.25-alpine AS base
WORKDIR /app

# Install system certificates needed for secure communication
RUN apk --no-cache add ca-certificates git

# Copy and download module dependencies first
COPY go.mod go.sum ./
RUN go mod download

# ── STAGE 2: DEVELOPMENT STAGE (Used by Docker Compose + Air) ──
FROM base AS dev
# Install the Air hot-reloader binary globally inside the container
RUN go install github.com/air-verse/air@latest
COPY . .
# Launch Air referencing your single global configuration profile
CMD ["air", "-c", ".air.toml"]

# ── STAGE 3: PRODUCTION BUILDER ──
FROM base AS builder

# Copy the entire codebase
COPY . .

# Pass the target binary via build arguments
ARG SERVICE_PATH

# Reuses host caches to instantly build lean production binaries
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /main ./cmd/${SERVICE_PATH}

# ── STAGE 4: PRODUCTION RUNTIME SCRATCH CONTAINER ──
FROM alpine:3.19 AS final
WORKDIR /

# Copy trusted certificates and compiled binary from the builder layer
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /main /main

# Copy migrations folder
COPY --from=builder /app/migrations /migrations

ENTRYPOINT ["/main"]


