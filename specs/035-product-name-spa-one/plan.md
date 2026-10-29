# Implementation Plan: Product Name SP&A One

**Branch**: `feature/035-product-name-spa-one` (from `develop`) | **Date**: 2026-10-28 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/035-product-name-spa-one/spec.md`

**Design artifacts**: [research.md](research.md) (R1–R8) · [data-model.md](data-model.md) ·
[contracts/ui-brand.md](contracts/ui-brand.md) · [contracts/openapi.md](contracts/openapi.md) ·
[quickstart.md](quickstart.md)

## Summary

The client renames the product from MAPP to **SP&A One** and makes the name the logo (D-332):

1. **The name is the logo.** `ui/Logo` drops the star tile and renders the name in Jost: "SP" and "A" Text
   Strong, "&" Shell dark blue, "One" Shell yellow — `#AD8700` (Primary Yellow darkened for light surfaces)
   in the light theme, Primary Yellow `#FFC600` in the dark theme. A new `LogoTile` (Shell dark blue with a
   Primary Yellow "1") stands for the product where the name does not fit — the top bar below 960 px — and
   is the favicon. Four new tokens carry the colours (research R2). The name is exposed as one text,
   "SP&A One" (R3).
2. **Every text that names the product says SP&A One** — tab titles, sign-in heading, footer, home-link
   names, descriptions, hints and errors in the web app; server messages, default help, the System actor's
   name and the downloaded file names in the API (R4, R5). Internal names stay (`mapp` module, `mapp.`
   storage keys, operator logs, code comments, the API's title).
3. **Constitution 4.5.0** (R1): X allows the product name's two colours and the tile; VIII exempts 035
   from the file-wide comment rule except in the files it redesigns; the opening states the product's
   on-screen name. Design system, glossary, decision log (D-332) and feature map follow.

## Technical Context

**Language/Version**: Go (pinned in `go.mod`); TypeScript 5.9 strict + React 19

**Primary Dependencies**: none new (Jost is already self-hosted from `@fontsource/jost`)

**Storage**: PostgreSQL; no schema change, no migration. The System actor's name is stored with the records
it writes (`*_by_name`, `actor_name`); new writes and a re-seed carry "SP&A One", older rows keep "MAPP"
(spec Edge Cases; R5)

**Testing**: Go handler and service tests on fakes and `apitest.DemoDB` (strings updated); Vitest + Testing
Library + vitest-axe for `Logo`, `LogoTile`, `TopBar`, `PublicLayout`, `SignInPage`, `AppLayout`,
`usePageTitle`; one Playwright file `frontend/e2e/product-name.spec.ts` for the two P1 stories (name,
tab titles, no "MAPP", computed colours in both themes, axe)

**Target Platform**: developers' machines (127.0.0.1); GitHub Actions

**Project Type**: Web application (Go API + React web app)

**Performance Goals**: unchanged

**Constraints**: constitution X as amended (R1); WCAG 2.2 AA — the wordmark is a logotype (SC 1.4.3
exemption) and still reaches 3:1 on Surface in both themes; the "1" is 6.7:1 on its tile

**Scale/Scope**: about 160 files with a string change (≈ 75 web-app sources and tests, ≈ 30 Go files,
8 e2e specs), 6 redesigned web-app files, the favicon, 1 contract file (descriptions only), 6 documents

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Check | Result |
|---|---|---|
| I Spec-first | Spec 035 with clarifications; IDs in test names | Pass |
| II Plain language | "SP&A One" in every product mention; Shell vocabulary unchanged | Pass |
| III Documented design | Design system §2, §3.1, §3.10, §5 `Logo`, §6 `PublicLayout`, glossary updated in the same change | Pass |
| IV Server is the source of truth | Names the server writes (actor, files, messages) change in the API; the web app shows them | Pass |
| V Simplicity | `Logo` stays one small component plus `LogoTile`; no config or theming layer for the name | Pass |
| VI Meaningful tests | Every acceptance scenario has a test; e2e for US1 and US2; existing tests only change their expected strings | Pass |
| VII Security | No new input, endpoint or dependency | Pass |
| VIII Comments | Budget kept in the redesigned files; other touched files exempt **only after** amendment 4.5.0 | Pass after R1 |
| IX Experience | The title stays the most prominent text; no yellow action added; the name is one link home | Pass |
| X Palette | `#AD8700`, `#003C88`/`#6F9BD1` text and a blue tile are outside today's rules | **Pass after R1** (amendment 4.5.0, product owner's instruction of 2026-10-28) |
| F6 Accessibility | Name read once as "SP&A One"; tile decorative inside a named link; axe on sign-in and home in both themes | Pass |

**Post-design re-check**: unchanged — every gate passes once the amendment (task T001) is merged in the
same pull request.

## Project Structure

### Documentation (this feature)

```text
specs/035-product-name-spa-one/
├── spec.md
├── plan.md              # this file
├── research.md          # R1–R8
├── data-model.md        # tokens and the System actor's name; no schema change
├── quickstart.md        # how to see and check the change
├── contracts/
│   ├── ui-brand.md      # Logo / LogoTile contract, tokens, where each appears
│   └── openapi.md       # descriptions and example names that change in backend/api/openapi.yaml
├── checklists/requirements.md
└── tasks.md             # /speckit-tasks
```

### Source Code (repository root)

```text
.specify/memory/constitution.md          # 4.5.0: X product name, VIII exemption, opening
frontend/
├── index.html                           # <title>SP&A One</title>
├── public/favicon.svg                   # the "1" tile
├── src/ui/tokens.css                    # --brand-name-amp, --brand-name-one, --brand-tile, --on-brand-tile; --brand-mark retired
├── src/ui/Logo/                         # Logo (name only) + LogoTile, CSS, tests
├── src/ui/usePageTitle.ts               # "· SP&A One"
├── src/ui/PublicLayout/                 # header shows the name
├── src/app/{TopBar,Sidebar,AppLayout}.tsx  # name ≥ 960 px, tile below; footer
├── src/features/**                      # strings that name the product
├── src/testing/fixtures/**              # System actor "SP&A One", messages
└── e2e/product-name.spec.ts             # new; other e2e specs: expected strings
backend/
├── api/openapi.yaml                     # example file names and "made by" wording
├── internal/platform/actor/actor.go     # System.Name = "SP&A One"
├── internal/platform/web/web.go, internal/settings/*.go   # messages, help defaults, lock reasons
├── internal/allwork/export.go, internal/purchaserequest/form_file.go, internal/financials/template.go  # file names
└── **/*_test.go                         # expected strings
docs/product/{design-system,glossary,demo-users-and-data}.md, docs/sdd/{decision-log,feature-map}.md
```

**Structure Decision**: existing web application layout; no new package or feature folder. `LogoTile`
lives beside `Logo` in `src/ui/Logo/` because both are the product's identity.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Text in a yellow shade and in a blue outside the palette (X) | The client's requirement for the product name | Primary Yellow text is 1.58:1 on white (the product owner rejected it); "One" on a yellow fill is not yellow letters |
| 035 exempt from the file-wide comment rule (VIII) outside its redesigned files | 44 of the ≈160 files it changes for one string are over budget today (≈ 450 comment lines) | Trimming them would bury the rename in unrelated deletions; the product owner chose the exemption (2026-10-28) |
