FROM oven/bun:latest AS builder

WORKDIR /client

COPY package.json bun.lockb ./
RUN bun install --frozen-lockfile

COPY . ./

FROM oven/bun:latest

WORKDIR /client

COPY --from=builder /client ./

CMD ["bun", "run", "src/index.ts"]
