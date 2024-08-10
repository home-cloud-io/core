FROM golang:1.21-alpine

RUN apk add --update --no-cache npm

# install buf
RUN go install github.com/bufbuild/buf/cmd/buf@v1.35.1

# install protoc plugins

# GO
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.2
RUN go install connectrpc.com/connect/cmd/protoc-gen-connect-go@v1.16.2
RUN go install github.com/envoyproxy/protoc-gen-validate@v1.0.4
RUN go install github.com/srikrsna/protoc-gen-gotag@v1.0.2

# WEB
RUN npm install typescript tsx
RUN npm install @bufbuild/buf \
    @bufbuild/protoc-gen-es@"^1.0.0" \
    @bufbuild/protobuf@"^1.0.0" \
    @connectrpc/protoc-gen-connect-es@"^1.0.0" \
    @connectrpc/connect@"^1.0.0"
