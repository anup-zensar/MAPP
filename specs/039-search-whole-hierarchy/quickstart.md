# Quickstart: Search the whole hierarchy

1. `go run ./cmd/seed -reset` and `go run ./cmd/api` in `backend/`; `pnpm run dev` in `frontend/`.
2. Sign in as Priya Raman.
3. **US1**: open *FY2026 Lubricants Europe*, type "venue" in "Search name or ID in this plan": the path to the
   activity *Social always-on* opens and the LINE ITEM row *Venue hire* is under it; Enter on it opens the line
   item. Clear the search: no line item rows.
4. **US2**: on Plans, type "venue": *FY2026 Lubricants Europe* is listed with "Found in line item 'Venue hire'
   (L-…)".

Checks: `go test ./internal/plan/... ./internal/app/...`; `pnpm vitest run src/features/plans`;
`pnpm run test:e2e -- plan-search`.
