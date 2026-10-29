# Quickstart: verify the QA fixes (038) with the demo data

## Start

```text
cd backend && go run ./cmd/seed -reset && go run ./cmd/api
cd frontend && pnpm run build && pnpm exec vite preview --host 127.0.0.1   # enforced headers (FR-015)
```

Demo password and people: `docs/product/demo-users-and-data.md`. Reset the data before each P1 check.

## P1

| # | Finding | Steps | Expected |
|---|---|---|---|
| 1 | C-1 | Priya: PR on L-1003 for 15,800.00 GBP → submit; Daniel approves. Priya edits it to 12,000.00 → Submit changes (confirm). Raise a 2nd PR for 3,800.00 on L-1003. Daniel declines the change. | The 2nd PR is refused ("only 0.00 GBP left to request"); after the decline L-1003 shows Available 0.00, never negative |
| 2 | H-4 / H-5 | Mei: Financials; open `/financials?link=notLinked`; open FT-1004 by address. Alex: link FT-1004 to L-1007, then unlink. Settings › Roles: tick "Link financial transactions" without "Import GSAP extracts" on a custom role | Mei: no Not linked tab or count, FT-1004 "no longer exists"; Alex links and unlinks; the role save is refused with the FR-006 message; Planner and Global planner no longer list the permission |
| 3 | H-2 | P-1001 › More › Import from Excel › `frontend/e2e/fixtures/workbook-5000.xlsx` › Check file | Result in ≤ 20 s; Import and Undo import work; a long value shows 200 characters + "…" |
| 4 | H-1 | `/dashboards/executive?currency=XXX`, `/dashboards/regional?team=T-9999` | Dashboard with filter bar; address corrected to the default |
| 5 | H-3 | Priya: open her PR › ⋯ › Delete › confirm | Purchase requests page, toast with Undo; Undo restores |
| 6 | Security | 6 wrong passwords for priya.raman@shell.com; then the right one; DevTools › Network › response headers of `/` and `/api/v1/me`; console during a journey | "Too many sign-in attempts. Try again in 15 minutes." also for the right password; `Content-Security-Policy`, `X-Frame-Options: DENY`, `Referrer-Policy`, `X-Content-Type-Options` present; no CSP violation |
| 7 | L-14, M-3 | `curl -X POST http://127.0.0.1:8080/api/v1/teams -H "Referer: http://evil.example/" …`; sign-in with `"email":"a\u0000b"` | Refused as from another site; 400 field problem, never 500 |

## P2

| Finding | Check |
|---|---|
| M-18 | Add line items above A-1003's planned cost; Daniel opens an open round → warning; Alex: Settings › Overspend control › "Refuse approval" → Daniel's Approve disabled with the reason; Decline works |
| M-1 | A-1011 › Spread over time, display USD then EUR: rows and columns add up |
| M-2, L-2 | P-1001 › SP&A view after line items exceed A-1003's cost: "Line items above planned cost" row, no negative Not yet scheduled; All work Budget & cost, dashboards, Budget by month show the note |
| M-5 | Priya: Home › Activities I own → All work lists the same number |
| M-6 | Sara: plan, campaign, initiative pages — no "Edit budget by month" |
| M-7 – M-10 | Sara and the Requisitioner on a PR; edit a PR with an empty Invoice approver (counter); Submit names the MoA approver; Submit changes asks first |
| M-11 | All work › Export CSV: "Partly approved" |
| M-12 | Empty Save on New global budget and New Shell attribute: all problems |
| M-4, L-28 | Open New team, type, delete the session cookie, Save → sign-in notice; sign in → same address with `#fragment` |
| M-13 | Settings › Currencies › Import rates with QA's `=HYPERLINK` file: that row under Problems |
| M-14 – M-17 | 120-character names at 390 px; Tab into the import problems table; axe on the listed texts |

## P3

Walk spec US14 – US17 scenarios; switch Profile › Display preferences to YYYY-MM-DD and check the Activity log
("Tuesday 2026-10-27") and "Accrued (2026-09)"; export the plan workbook and SUM a money column; submit PR-1001 after
approving its activity.

## Automated

```text
cd backend && go test -race ./...              # with TEST_DATABASE_URL for scope isolation and apibench
cd frontend && pnpm run check && pnpm run test:e2e
```

Then re-run QA's scripts (task T203) from the QA scratchpad (`qa/area1` – `qa/area4`, `qa/verify`) against the
fresh environment and record the result per finding ID in tasks.md.

## Release note

Built-in Global planner and Planner roles no longer hold "Link financial transactions" (D-353): linking is done by
the people who import GSAP extracts, who are the only ones who see transactions that are not linked.
