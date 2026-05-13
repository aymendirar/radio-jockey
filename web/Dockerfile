FROM oven/bun:alpine AS builder

WORKDIR /web

COPY package.json bun.lockb ./
RUN bun install --frozen-lockfile

COPY . ./
RUN bun run build

FROM oven/bun:alpine

WORKDIR /web

COPY --from=builder /web/build ./build
COPY --from=builder /web/node_modules ./node_modules
COPY --from=builder /web/package.json ./package.json

ENV NODE_ENV=production
ENV PORT=3000

EXPOSE 3000

CMD ["bun", "build/index.js"]
