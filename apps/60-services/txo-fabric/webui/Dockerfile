# Build the browser artifact independently from the private BFF runtime.
# Runtime GitOps pins the complete image by immutable digest.
FROM node:22.16.0-alpine3.21 AS build-client
WORKDIR /app
COPY client/package.json client/package-lock.json ./client/
RUN npm ci --prefix client
COPY client ./client
RUN npm run build --prefix client

FROM node:22.16.0-alpine3.21 AS runtime-deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --omit=dev

# Private-only runtime; no browser handler is mounted while gates are OFF.
FROM node:22.16.0-alpine3.21
WORKDIR /app
ENV NODE_ENV=production
ENV TXO_WEBUI_BFF_BROWSER_ENABLED=false
COPY --from=runtime-deps /app/node_modules ./node_modules
COPY package.json ./package.json
COPY server ./server
COPY --from=build-client /app/client/dist /app/client
RUN chown -R 10001:10001 /app
USER 10001:10001
EXPOSE 3000
CMD ["node", "server/private-bff-main.mjs"]
