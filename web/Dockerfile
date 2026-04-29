FROM oven/bun:latest AS builder

WORKDIR /web

COPY package.json bun.lockb ./
RUN bun install --frozen-lockfile

COPY . ./
RUN bun run build

FROM node:alpine

WORKDIR /web

COPY --from=builder /web/build ./build
COPY --from=builder /web/package.json ./package.json

ENV NODE_ENV=production
ENV PORT=3000

EXPOSE 3000

CMD ["node", "build/index.js"]
