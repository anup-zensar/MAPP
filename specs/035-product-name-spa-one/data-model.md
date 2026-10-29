# Data Model: Product Name SP&A One

No record type, field, migration or endpoint changes.

## Stored names

| Where | Before | After | Notes |
|---|---|---|---|
| `actor.System.Name` (written into `created_by_name`, `updated_by_name`, `actor_name`, import and run "by" names) | `MAPP` | `SP&A One` | New writes and a re-seed; earlier rows keep "MAPP" (research R5) |
| Default field help `plan.name` (settings catalogue) | "… It must be unique in MAPP." | "… It must be unique in SP&A One." | Seeded default; edited help is the person's text and stays |

## Design tokens (`frontend/src/ui/tokens.css`)

| Token | Light | Dark | Replaces |
|---|---|---|---|
| `--brand-name-amp` | `#003c88` | `#6f9bd1` | — |
| `--brand-name-one` | `#ad8700` | `#ffc600` | — |
| `--brand-tile` | `#003c88` | `#003c88` | `--brand-mark` (retired) |
| `--on-brand-tile` | `#ffc600` | `#ffc600` | — |

## File names the API gives

| File | Before | After |
|---|---|---|
| All work export | `MAPP All work – Grid – 2026 – 28 Oct 2026.xlsx` | `SP&A One All work – Grid – 2026 – 28 Oct 2026.xlsx` |
| PR form | `MAPP PR form – PR-1001 – Original – 28 Oct 2026.xlsx` | `SP&A One PR form – PR-1001 – Original – 28 Oct 2026.xlsx` |
| GSAP extract sample | `mapp-gsap-extract-sample.csv` | `spa-one-gsap-extract-sample.csv` |
