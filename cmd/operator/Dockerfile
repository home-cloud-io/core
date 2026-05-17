ARG GO_VERSION=1.26.3
ARG NODE_VERSION=22

# Build web client
FROM node:${NODE_VERSION} AS web-client-builder
WORKDIR /web

# Install node modules
COPY ./services/platform/operator/web-client/package*.json .
RUN npm install

# Build dist
COPY ./services/platform/operator/web-client .
ENV NODE_ENV=production
RUN npm run build

# Build final binary
FROM golang:${GO_VERSION}-alpine AS go-builder
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
COPY --from=web-client-builder /web/dist ./services/platform/operator/web-client/dist

# Build the binary
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o main ./services/platform/operator/main.go

# Final image
FROM scratch
COPY --from=go-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=go-builder /build/main /etc/main

WORKDIR /etc
ENTRYPOINT ["/etc/main"]
