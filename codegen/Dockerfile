FROM golang:1.25

RUN apt-get update && apt-get install -y --no-install-recommends \
    protobuf-compiler nodejs npm \
    && rm -rf /var/lib/apt/lists/*

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest & \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest & \
    go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest & \
    wait

RUN npm install -g @bufbuild/protoc-gen-es @connectrpc/protoc-gen-connect-es

WORKDIR /codegen
