FROM node:latest AS builder

WORKDIR /discord-jockey

COPY package.json package-lock.json ./
RUN rm package-lock.json && npm install

COPY src ./src
COPY tsconfig.json ./
RUN npm run build

FROM node:current-slim

RUN apt-get update && apt-get install -y ffmpeg && rm -rf /var/lib/apt/lists/*

WORKDIR /discord-jockey

COPY --from=builder /discord-jockey ./

CMD ["node", "dist/index.js"]
