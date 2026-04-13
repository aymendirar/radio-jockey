FROM golang:1.25

RUN apt-get update && apt-get install -y --no-install-recommends \
    protobuf-compiler ruby ruby-dev build-essential libc6-i386 \
    && rm -rf /var/lib/apt/lists/*

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest & \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest & \
    go install github.com/sorbet/protoc-gen-rbi@latest & \
    wait

RUN gem install grpc-tools google-protobuf --no-document

WORKDIR /codegen
