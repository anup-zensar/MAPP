# ADR-0009: Simulated GSAP feeds behind adapters

**Status:** Accepted · **Date:** 2026-10-20

## Context
Shell's MVP requirements need GSAP data in MAPP: daily exchange rates (BUS.BUD.050) and purchase
orders, invoices, payments and accruals shown where people plan (BUS.FIN.080). GSAP is frozen until about
March 2027, so no live connection can be built or tested now. In Hive9 this data arrives in daily files:
PGSI 941 brings PR, PO and invoice data back, matched on the HX key, and GSAP publishes daily rates. The
product owner decided on 2026-10-20 to build these requirements now with simulated feeds (D-242, D-259,
D-315). ADR-0003 already expects a GSAP worker later.

## Decision
- **One adapter interface per kind of feed, owned by the package that uses the data.** An adapter only
  turns one delivery into rows.
  - Purchase orders, invoices, payments and accruals (feature 015): `financials.Feed` (`Name`, `Source`,
    `Read(ctx, Delivery)` → incoming rows and row problems), with two implementations now: `CSVFeed`, the
    simulated GSAP extract, and `DemoFeed`, the demo rows. Four transaction types: purchase-order lines
    (Committed), invoices with credit notes (Invoiced), payments (Paid) and accruals (Accrued) (D-243).
  - Daily exchange rates (feature 016): `settings.RateFeed` (`Name`, `Read(fileName, content)` → rows),
    with one implementation, `GSAPRateFile`; `ApplyRateRows(ctx, actor, feedName, rows)` is the shared core
    that applies rows under the rate rules.
- **Import endpoints.** Each simulated feed is a file import in two steps — check (every row's outcome,
  nothing saved), then import — of a UTF-8 CSV with fixed columns, sent as base64 JSON, with a downloadable
  sample file:
  - transactions: `POST /api/financials/imports/check`, `POST /api/financials/imports`,
    `GET /api/financials/imports`, `GET /api/financials/imports/template` (permission `finance.import`);
  - rates: `POST /api/exchange-rates/import-check`, `POST /api/exchange-rates/import`,
    `GET /api/exchange-rates/imports`, `GET /api/exchange-rates/import-sample` (permission
    `exchangeRates.manage`).

  Rows with problems are skipped and the rest imported; a row sent again updates or replaces what is there;
  imports never delete; each import is one stored record and one activity-log entry (D-246, D-247, D-254,
  D-259).
- **Demo seed data.** The seed loads fictitious GSAP data in the order of D-306: the demo transactions
  through `DemoFeed` and the same apply code as an import (015 data-model §6), and 30 days of daily rates
  ending with today's seeded rates, recorded as rate import RI-1001 (D-274).
- **The rules stay in the services, the same for every feed.** Checks against Settings (is the currency
  set up), duplicate keys, matching transactions to line items (line-item ID → HX key → PO number → SWBS
  code, D-245), currency conversion on read (D-248, D-303) and every figure belong to the `financials` and
  `settings` services, not to the adapters.
- **Inbound and read-only.** MAPP never creates, changes, deletes or sends GSAP transactions, SWBS
  codes, purchase orders or invoices; links made from GSAP data are corrected in GSAP, not in MAPP
  (D-253).

## Consequences
- **A real feed replaces a simulation by adding one implementation of the same interface:** a PGSI 941
  reader for `financials.Feed`, and an automatic daily-rate feed that calls `ApplyRateRows`. Services, rules,
  the API contract and the screens do not change; the file imports can stay as a fallback.
- A feed that runs on a schedule or connects to GSAP (through SAP Cloud Integration) is a new runtime part
  or job; it gets its own ADR when MAPP is hosted (constitution IV; ADR-0003's GSAP worker).
- Until then, a person with the import permission loads GSAP extracts by hand. Figures are as fresh as the
  last import, and screens name the source and its date ("GSAP data as of …").
- Shell's real extract formats may differ from the simulated columns (Q-012, Q-015); only the adapter's
  reader changes when they are known.
