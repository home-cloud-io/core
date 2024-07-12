ARG GO_VERSION=1.21.3
ARG ALPINE_VERSION=3.18

# Build web client (if needed)
FROM node:18 AS web-client-builder
WORKDIR /web
RUN npm i -D @swc/cli @swc/core
RUN rm package*.json

ARG DOMAIN
ARG SERVICE

# Install node modules
COPY ./services/${DOMAIN}/${SERVICE}/web-client/package*.json .
RUN npm install

# Build dist
COPY ./services/${DOMAIN}/${SERVICE}/web-client .
RUN npm run build

# Make sure dist directory exists even if there's no client to build
RUN mkdir -p ./dist

# Build final binary
FROM golang:${GO_VERSION}-alpine AS go-builder
WORKDIR /app
RUN apk add --no-cache git upx

ARG DOMAIN
ARG SERVICE

# We want to populate the module cache based on the go.{mod,sum} files.
COPY ./services/${DOMAIN}/${SERVICE}/go.mod ./go.mod
COPY ./services/${DOMAIN}/${SERVICE}/go.sum ./go.sum

# Fetch go modules
RUN --mount=type=cache,target=/go/pkg/mod \
go mod download
RUN --mount=type=cache,target=/go/pkg/mod \
go mod verify

# Copy over code
COPY ./services/${DOMAIN}/${SERVICE} .
COPY --from=web-client-builder /web/dist ./web-client/dist

# Run tests
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    go test -test.v ./...

# Build the binary
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    GOOS=linux go build -o main ./main.go

# Compress the binary
RUN upx main

# Final image
FROM alpine:${ALPINE_VERSION} AS runner

# Install dependencies
RUN apk add -U --no-cache ca-certificates

# Copy the binary from go-builder
COPY --from=go-builder /app/main /etc/main

# Set context to run main
WORKDIR /etc

# Run
ENTRYPOINT ["/etc/main"]

# Runtime arguments
ARG HTTP_PORT=8080
ARG GRPC_PORT=8090

# Expose needed ports
EXPOSE ${HTTP_PORT}
EXPOSE ${GRPC_PORT}
