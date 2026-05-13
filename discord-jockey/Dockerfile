FROM oven/bun

RUN apt-get update && apt-get install -y ffmpeg && rm -rf /var/lib/apt/lists/*

WORKDIR /discord-jockey

COPY package.json bun.lockb ./
RUN bun install --frozen-lockfile

COPY src ./src

CMD ["bun", "src/index.ts"]
