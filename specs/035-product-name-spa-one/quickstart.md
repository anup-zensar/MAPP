# Quickstart: check SP&A One

## Prerequisites

PostgreSQL running locally (natively or `docker compose up -d postgres`), `DATABASE_URL` set as in the
README.

## Run

```sh
cd backend && go run ./cmd/seed && go run ./cmd/api      # re-seed so System changes read "SP&A One"
cd frontend && pnpm run dev
```

## Look

1. Open the app: the sign-in page shows **SP&A One** (no star), heading "Sign in to SP&A One", tab
   "Sign in · SP&A One", favicon a dark-blue tile with a yellow "1".
2. Sign in as any demo person: tab "Home · SP&A One", footer "© SP&A One — demo data", top bar shows the
   name; in the light theme "&" is dark blue and "One" dark gold; switch to dark: "&" light blue, "One"
   Primary Yellow.
3. Narrow the window below 960 px: the top bar shows the "1" tile; Open menu shows the name in the drawer.
4. Settings → Currencies → USD: "USD is SP&A One's base currency; its rate is always 1."
5. All work → Export: the file is "SP&A One All work – …".
6. Activity log: changes made by the product itself are by "SP&A One".

## Verify

```sh
cd backend && go vet ./... && go test -short ./...
cd frontend && pnpm run api:types && pnpm run check && pnpm run test:e2e -- product-name
grep -rn "MAPP" frontend/src --include='*.tsx' --include='*.ts' | grep -v generated   # comments only
```

Expected: all green; the last command lists only code comments (the codename), no strings people see.
