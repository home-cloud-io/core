ARG GO_VERSION=1.26.3

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
    CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o main ./cmd/mdns/main.go

# Final image
FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/main /etc/main

WORKDIR /etc
ENTRYPOINT ["/etc/main"]
