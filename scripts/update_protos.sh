#!/bin/sh
set -e

mkdir -p server/src/proto client/src/proto

protoc --go_out=server/src/proto --go_opt=paths=source_relative \
       --go-grpc_out=server/src/proto --go-grpc_opt=paths=source_relative \
       -I proto proto/discord_jockey.proto

grpc_tools_ruby_protoc --ruby_out=client/src/proto \
                        --grpc_out=client/src/proto \
                        --rbi_out=client/src/proto \
                        -I proto proto/discord_jockey.proto

echo "successfully generated service protos"
