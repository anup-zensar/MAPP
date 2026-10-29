# Contract changes: `backend/api/openapi.yaml`

Descriptions only — no path, operation, schema, parameter or status changes (B6: backward compatible).
`info.title` ("MAPP API") and the contract's own introduction keep the codename (research R4).

| Operation / schema | Change |
|---|---|
| All work export (`200` description) | Example file name `SP&A One All work – Grid – 2026 – 18 Oct 2026.xlsx` |
| `getImportTemplate` (`200` description) | `SP&A One import template – <plan>.xlsx` |
| GSAP extract sample (`200` description) | `spa-one-gsap-extract-sample.csv` |
| `downloadPurchaseRequestForm` | "Routed by SP&A One — no supporting documents are attached"; file name `SP&A One PR form – PR-1001 – Original – 27 Oct 2026.xlsx` |
| `runApprovalEscalations` | "Events are recorded as made by SP&A One." |
| `unusedColumns` | "Header names SP&A One does not read." |

After the change: `pnpm run api:types` regenerates `frontend/src/api/generated/schema.d.ts` (comments only).
