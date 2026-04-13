FROM golang:1.25-alpine AS builder

WORKDIR /server

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN go build -o ./main ./src
RUN go build -o ./cli ./src/cmd
RUN go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

FROM alpine

RUN apk add --no-cache sqlite python3 py3-pip ffmpeg && \
    pip install yt-dlp --break-system-packages

WORKDIR /server

RUN mkdir -p /server/data /server/music

COPY --from=builder /server/main ./main
COPY --from=builder /go/bin/grpcurl /usr/local/bin/grpcurl

CMD ["./main"]
