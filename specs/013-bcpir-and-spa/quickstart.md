# Quickstart: SP&A Plan View and BCPIR Marker

1. Seed and start: from `backend/`, `go run ./cmd/seed` then `go run ./cmd/api`; from `frontend/`, `pnpm run dev`.
2. Sign in as Priya Raman (Planner, Lubricants Europe).
3. **BCPIR (US2)**: open Plans — "FY2026 Lubricants Europe" carries the BCPIR marker. Choose "BCPIR plans: Only":
   only it remains. Open it: the header shows the marker. Open All work and choose the filter "BCPIR plans: Only":
   only its records remain, with a removable chip; choose Breakdown "BCPIR plan" to see Yes and No.
4. **SP&A (US1)**: on the plan page choose "SP&A view". Check the headline figures against the plan page's
   Financials card (Committed, Paid as Actual spend) and planned cost; open "Spend by quarter", switch to months;
   choose another dimension; sort the activity table by Actual spend and open a row's line items; select
   Committed and see the activities that make it up.
5. Choose a campaign in the scope picker; Back returns to the whole plan.
6. **Budget and procurement (US3)**: the budget panel shows the plan budget and Planned vs budget; the procurement
   panel lists PR-1001 (Created, counted as Draft). On an activity row choose "Raise purchase request", pick a line
   item: 032's New purchase request form opens.
7. Sign in as a person outside Lubricants Europe and open `/plans/P-1001/spa`: "not found".
