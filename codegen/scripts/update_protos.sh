#!/bin/sh
set -e

mkdir -p server/src/proto client/src/proto

protoc --go_out=server/src/proto --go_opt=paths=source_relative \
       --go-grpc_out=server/src/proto --go-grpc_opt=paths=source_relative \
       --connect-go_out=server/src/proto --connect-go_opt=paths=source_relative \
       -I proto proto/radio.proto

grpc_tools_ruby_protoc --ruby_out=discord-jockey/src/proto \
                        --grpc_out=discord-jockey/src/proto \
                        --rbi_out=discord-jockey/src/proto \
                        -I proto proto/radio.proto

echo "successfully generated service protos"
