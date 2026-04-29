#!/bin/sh
set -e

mkdir -p server/src/proto discord-jockey/src/proto web/src/lib/proto

protoc --go_out=server/src/proto --go_opt=paths=source_relative \
       --go-grpc_out=server/src/proto --go-grpc_opt=paths=source_relative \
       --connect-go_out=server/src/proto --connect-go_opt=paths=source_relative \
       -I proto proto/radio-jockey.proto

protoc --es_out=discord-jockey/src/proto --es_opt=target=ts \
       --connect-es_out=discord-jockey/src/proto --connect-es_opt=target=ts \
       -I proto proto/radio-jockey.proto

protoc --es_out=web/src/lib/proto --es_opt=target=ts \
       --connect-es_out=web/src/lib/proto --connect-es_opt=target=ts \
       -I proto proto/radio-jockey.proto

echo "successfully generated service protos"
