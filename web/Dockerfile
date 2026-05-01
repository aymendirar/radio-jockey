FROM node:25-alpine AS builder

WORKDIR /web

COPY package.json ./
RUN npm install

COPY . ./
RUN npm run build

FROM node:25-alpine

WORKDIR /web

COPY --from=builder /web/build ./build
COPY --from=builder /web/node_modules ./node_modules
COPY --from=builder /web/package.json ./package.json

ENV NODE_ENV=production
ENV PORT=3000

EXPOSE 3000

CMD ["node", "build/index.js"]
