# Quickstart: Overspend Control

Run the API and the web app on fresh demo data (`go run ./cmd/seed`, `go run ./cmd/api`, `pnpm run dev`).

## US1 — Refuse on a plan budget (P1)

1. Sign in as **Priya Raman**, open **FY2026 Lubricants Europe**, choose **Edit**. Below Plan budget, under
   "When planned cost goes over this budget", choose **Refuse** and save. The plan shows "Refuses
   overspending"; the activity log shows "Overspend: Warn → Refuse".
2. Open any activity of the plan and choose **Edit**. The Planned cost field says how much the plan has left
   to plan. Type a planned cost that raises the activity by more than that: the field says "This is <X> over
   the budget of plan 'FY2026 Lubricants Europe', which refuses overspending" and Save is refused on the form.
3. Send the same save through the API (`PATCH /api/v1/activities/{id}`): 409 `over_budget`, message naming the
   plan, its budget and the overage; nothing changed.
4. Set the plan back to **Warn**: the same save goes through and the plan's figures show the overage in red.

## US2 — Every path

With "Helix launch" on Refuse and little left to plan: move or copy an activity into it, restore a deleted
activity of it from the recycle bin, split one of its activities, copy the plan — each is refused naming
"Helix launch" when it would go over, and saves otherwise.

## US3 — Global budget and allocation

As **Alex Morgan**, open a global budget, **Edit**, choose Refuse. **Allocate** to a plan whose planned cost does
not fit: refused. In **Edit allocation**, choose Refuse for the plan's budget; open the plan as Priya: the
choice is read-only with "Allocated from '…' — ask Alex Morgan to change it."

## US4 — Settings

As Alex, **Settings › Overspend control**: choose Refuse ("Existing budgets keep their own setting."), save. A
new campaign's form starts on Refuse. As Priya the page is read-only with the reason.

## US5 — Purchase request gate (P1)

As Priya, on an approved activity of 12,000.00 USD add line items of 20,000.00 USD, raise a purchase request on
one, get its SWBS code from the simulated GSAP (as Alex) and choose **Submit for approval**: refused, listing
"Activity '…': line items exceed its planned cost by 8,000.00 USD …" with any other problem.

## Checks

`golangci-lint run --new-from-rev=587eade2 ./...`, `go test -race ./...` with `TEST_DATABASE_URL`,
`pnpm run check`, `pnpm run build`, `pnpm run test:e2e` (`overspend-control.spec.ts` and the full suite).
