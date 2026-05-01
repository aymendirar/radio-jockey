FROM node:25-alpine AS builder

WORKDIR /discord-jockey

COPY package.json package-lock.json ./
RUN npm ci

COPY src ./src
COPY tsconfig.json ./
RUN npm run build

FROM node:25-slim

RUN apt-get update && apt-get install -y ffmpeg && rm -rf /var/lib/apt/lists/*

WORKDIR /discord-jockey

COPY --from=builder /discord-jockey ./

CMD ["node", "dist/index.js"]
