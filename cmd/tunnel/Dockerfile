ARG GO_VERSION=1.26.3
ARG ALPINE_VERSION=3.23

# Builder image
FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /build

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod go.sum ./

# Fetch go modules
RUN --mount=type=cache,target=/go/pkg/mod \
go mod download
RUN --mount=type=cache,target=/go/pkg/mod \
go mod verify

# Copy over code
COPY . .

# Build the binary
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o main ./services/platform/tunnel/main.go

# Final image
FROM alpine:${ALPINE_VERSION} AS runner

# Install dependencies
RUN apk add -U --no-cache ca-certificates iptables

# Copy the binary from builder
COPY --from=builder /build/main /etc/main

# Set context to run main
WORKDIR /etc

# Run
ENTRYPOINT ["/etc/main"]
