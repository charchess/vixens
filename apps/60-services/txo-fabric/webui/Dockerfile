# Private-only runtime; no browser handler is mounted when gate is OFF.
# Runtime GitOps pins the built GHCR image by immutable digest.
FROM node:22.16.0-alpine3.21
WORKDIR /app
ENV NODE_ENV=production
ENV TXO_WEBUI_BFF_BROWSER_ENABLED=false
COPY package.json ./package.json
COPY server ./server
RUN chown -R 10001:10001 /app
USER 10001:10001
EXPOSE 3000
CMD ["node", "server/private-bff-main.mjs"]
