# ADR-0012: Local container images for reviewers

**Status:** Accepted · **Date:** 2026-10-26

## Context
Business analysts review MAPP on their own Windows laptops until it is deployed in Shell's environment.
Installing Go, Node.js and PostgreSQL, and running three commands in two terminals, is too much for them.
They can install Podman Desktop (free, Apache 2.0) through IT. Constitution 4.0.0 kept the application out
of containers in this phase; 4.1.0 allows images for running it locally. Developers keep running MAPP
natively, and Docker or Podman stays optional for everyone.

## Decision
- **`compose.yaml` builds and runs the whole app from the checkout**: `docker compose up -d --build` or
  `podman compose up -d --build`, then <http://127.0.0.1:3000>. Nothing is pushed to a registry; the
  images are built on the laptop. `docker compose up -d postgres` still starts PostgreSQL alone.
- **`backend/Dockerfile`**: a `golang:1.27.3-alpine` stage builds `api` and `seed` with `CGO_ENABLED=0`,
  `-trimpath` and `-ldflags="-s -w"`; the run stage is `gcr.io/distroless/static-debian13:nonroot`, with no
  shell or package manager, running as user 65532.
- **`frontend/Dockerfile`**: a `node:24-alpine` stage runs `pnpm install --frozen-lockfile` and `pnpm run build`; the run stage is
  `nginxinc/nginx-unprivileged:1.29-alpine` (non-root), which serves the production build, falls back to
  `index.html` for the app's routes, caches the hashed assets for a year and forwards `/api/` to the API
  container with the browser's `Host`, as the Vite development server does. Node and `node_modules` stay
  in the build stage. Vite's development or preview server was considered: no extra config, but the image
  keeps Node and every package, several hundred MB instead of about 50.
- **Layer order**: `go.mod`/`go.sum` and `package.json`/`pnpm-lock.yaml` are copied and installed before
  the code, so a rebuild after a code change reuses the downloaded modules. No BuildKit-only syntax, so
  Docker's classic builder and Podman build the same files. Both images are two stages; only the
  binaries and the built web files reach the final images.
- **Local only stays true**: inside their containers the API and nginx listen on `0.0.0.0` (`APP_HOST`, which accepts only
  `127.0.0.1`, the default, or `0.0.0.0`); compose publishes only the web app,
  on `127.0.0.1:3000`, so the API is reachable only through it.
- **Data**: the `api` and `seed` services use the `postgres` service through `DATABASE_URL`. A one-off
  `seed` service waits for PostgreSQL to be healthy and runs `seed -if-empty` before the API starts: it
  creates and migrates the database, loads the demo data the first time, and keeps what reviewers created
  afterwards. `compose down -v` deletes the database volume, so the next start loads the demo data again.

## Consequences
- A reviewer needs Podman Desktop (or Docker), Git and read access to the repository; each release is
  `git pull` and `podman compose up -d --build`.
- These images are for local review only. Production images, their registry and their scanning are chosen
  by ADR before hosting (constitution, Release Gates).
