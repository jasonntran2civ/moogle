FROM node:20-alpine AS build
WORKDIR /app
RUN corepack enable
COPY package.json tsconfig.json ./
RUN pnpm install --frozen-lockfile=false
COPY src ./src
RUN pnpm build

FROM node:20-alpine
WORKDIR /app
ENV NODE_ENV=production MCP_TRANSPORT=http
COPY --from=build /app/node_modules ./node_modules
COPY --from=build /app/dist ./dist
COPY package.json ./
EXPOSE 8082
HEALTHCHECK --interval=30s --timeout=5s CMD wget -q -O- http://localhost:8082/healthz || exit 1
CMD ["node", "dist/server.js"]
