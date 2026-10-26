# The web app for running MAPP locally with compose (ADR-0012): built once, served by nginx, which also
# forwards /api to the API container as the development server does. Not a production image.

FROM node:24-alpine AS build
WORKDIR /app
# Dependencies first, so a change to the code alone reuses this layer.
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN corepack enable && COREPACK_ENABLE_DOWNLOAD_PROMPT=0 pnpm install --frozen-lockfile
COPY . .
RUN pnpm run build

# Only the built files reach this image; it runs as a non-root user and listens on 8080.
FROM nginxinc/nginx-unprivileged:1.29-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 8080
