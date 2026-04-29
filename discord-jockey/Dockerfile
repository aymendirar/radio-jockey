FROM oven/bun:latest AS builder

WORKDIR /discord-jockey

COPY package.json bun.lockb ./
RUN bun install --frozen-lockfile

COPY . ./

FROM oven/bun:latest

RUN apt-get update && apt-get install -y ffmpeg && rm -rf /var/lib/apt/lists/*

WORKDIR /discord-jockey

COPY --from=builder /discord-jockey ./

CMD ["bun", "run", "src/index.ts"]
