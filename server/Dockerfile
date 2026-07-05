FROM golang:1.25-alpine AS base

WORKDIR /server

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

FROM base AS builder

RUN go build -o ./main ./src

FROM base AS test

RUN apk add --no-cache ffmpeg

CMD ["go", "test", "./...", "-v"]

FROM alpine

RUN apk add --no-cache curl python3 py3-pip ffmpeg && \
    pip install yt-dlp --break-system-packages

WORKDIR /server

RUN mkdir -p /server/data /server/music

COPY --from=builder /server/main ./main

CMD ["./main"]
