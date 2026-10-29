# Tasks: Product Name SP&A One

**Input**: Design documents from `specs/035-product-name-spa-one/` — [plan.md](plan.md), [spec.md](spec.md),
[research.md](research.md), [data-model.md](data-model.md), [contracts/](contracts/), [quickstart.md](quickstart.md)

**Tests**: required by constitution VI — a test for every acceptance scenario, an end-to-end test for each P1
story (US1, US2). Write each new test first and see it fail.

**Rename rule (all tasks)**: replace "MAPP" with "SP&A One" only in text people read (string literals, JSX
text, `aria-label`s, expected strings in tests). Never in code comments, identifiers, `mapp.` storage keys,
`slog` messages or the seed command's console messages (research R4). In files outside the redesign list
(R1) do not touch comments.

## Phase 1: Setup (governance)

- [X] T001 Amend `.specify/memory/constitution.md` to 4.5.0 per research R1: X "Product name" rule (values of
  R2; the only yellow or yellow-shade text; never "act here"), VIII exemption for 035 except its redesigned
  files, the opening's on-screen name, Sync Impact Report, Amendment History row, version line
- [X] T002 [P] Add D-332 to `docs/sdd/decision-log.md` (name, colours, tile, exemption, unchanged internal
  names) and row 035 to `docs/sdd/feature-map.md`; add the on-screen name to the Project section of `CLAUDE.md`

## Phase 2: Foundational (blocking)

- [X] T003 Add tokens to `frontend/src/ui/tokens.css` (light and dark blocks): `--brand-name-amp` `#003c88` /
  `#6f9bd1`, `--brand-name-one` `#ad8700` / `#ffc600`, `--brand-tile` `#003c88`, `--on-brand-tile` `#ffc600`;
  remove `--brand-mark`; trim the file's comments to its budget (5% of non-blank lines, ≈ 12; the design
  system §3.1 keeps the contrast notes)

**Checkpoint**: tokens exist — US1 and US2 can start.

## Phase 3: User Story 1 — SP&A One wherever the name appears (P1) 🎯 MVP

**Goal**: no screen, tab title or accessible name says "MAPP"; every product mention says "SP&A One".

**Independent test**: sign in, visit every navigation page, read texts and tab titles (quickstart steps 1–2).

### Tests (first)

- [X] T004 [P] [US1] Write `frontend/e2e/product-name.spec.ts` test "035 US1: …" (FR-001, FR-002): sign-in
  page heading "Sign in to SP&A One" and tab "Sign in · SP&A One"; after `signInAs`, tab "Home · SP&A One", footer
  "© SP&A One — demo data", home link "SP&A One home"; on each main navigation page the body text has no
  "MAPP"
- [X] T005 [P] [US1] Update expected strings in `frontend/src/ui/usePageTitle.test.ts`,
  `frontend/src/app/AppLayout.test.tsx`, `frontend/src/app/TopBar.test.tsx`,
  `frontend/src/app/NavDrawer.test.tsx`, `frontend/src/app/App.test.tsx`,
  `frontend/src/app/router.sign-in.test.tsx`, `frontend/src/features/sign-in/SignInPage.test.tsx`,
  `frontend/src/ui/PublicLayout/PublicLayout.test.tsx` and see them fail

### Implementation

- [X] T006 [US1] `frontend/src/ui/usePageTitle.ts` → `${page} · SP&A One`; `frontend/index.html` `<title>SP&amp;A One</title>`
- [X] T007 [US1] `frontend/src/features/sign-in/SignInPage.tsx` heading "Sign in to SP&A One" and the
  "Stop … API" hint; `frontend/src/app/AppLayout.tsx` footer "© SP&A One — demo data"; bring both files'
  comments within budget
- [X] T008 [US1] Rename the product in the remaining web-app sources: `frontend/src/api/errors.ts`,
  `frontend/src/features/all-work/api.ts` (fallback `SP&A One All work.${format}`),
  `frontend/src/features/financials/ImportDialog.tsx`, `frontend/src/features/home/home-text.ts`,
  `frontend/src/features/line-items/{gsap-financials,swbs-text}.ts`,
  `frontend/src/features/people/{PeoplePage,ProfilePage,settings-sections}.tsx|ts`,
  `frontend/src/features/people/access/RulesTab.tsx`, `frontend/src/features/placeholders/ComingSoonPage.tsx`,
  `frontend/src/features/purchase-requests/page-text.ts`,
  `frontend/src/features/settings/currencies/ImportRatesPage.tsx`,
  `frontend/src/features/settings/field-help/ResetHelpDialog.tsx`,
  `frontend/src/features/simulated-gsap/simulated-gsap-text.ts`, and any other non-comment match of
  `grep -rn MAPP frontend/src`
- [X] T009 [US1] Fixtures: System actor `{ id: 'system', name: 'SP&A One' }` and messages in
  `frontend/src/testing/fixtures/{access,dev,financial-imports,settings,sign-in,swbs-runs}.ts(x)`; update the
  expected strings of every other `*.test.ts(x)` that names the product (≈ 75 files)
- [X] T010 [US1] Update expected strings in the e2e specs `frontend/e2e/{activity-log,all-work,approval-restart-and-profile,brd-gaps,first-page,recycle-bin,simulated-gsap-swbs,smoke}.spec.ts`

**Checkpoint**: US1 tests green; the name reads "SP&A One" everywhere on screen.

## Phase 4: User Story 2 — The name is the logo, in Shell's colours (P1)

**Goal**: the star is gone; coloured name; "1" tile below 960 px and as favicon.

**Independent test**: quickstart steps 2–3 in both themes; screen reader reads "SP&A One".

### Tests (first)

- [X] T011 [P] [US2] Rewrite `frontend/src/ui/Logo/Logo.test.tsx` (FR-003–FR-005): `Logo` is one image named
  "SP&A One", draws no svg; a link around it is named "SP&A One"; `LogoTile` is hidden from assistive
  technology; axe passes
- [X] T012 [P] [US2] Add "035 US2: …" to `frontend/e2e/product-name.spec.ts`: computed colour of "&" and
  "One" in the light theme (`rgb(0, 60, 136)`, `rgb(173, 135, 0)`) and dark theme (`rgb(111, 155, 209)`,
  `rgb(255, 198, 0)`); at 800 px wide the top bar shows the tile, not the name; axe on sign-in and home in
  both themes

### Implementation

- [X] T013 [US2] Rewrite `frontend/src/ui/Logo/Logo.tsx` and `Logo.module.css` per
  `contracts/ui-brand.md` (name only, `size`, `role="img"` + `aria-label`, no `withWordmark`); add
  `frontend/src/ui/Logo/LogoTile.tsx` (28 px `--brand-tile` tile, "1" path in `--on-brand-tile`,
  `aria-hidden`), comments within budget
- [X] T014 [US2] `frontend/src/app/TopBar.tsx` + `TopBar.module.css`: `aria-label="SP&A One home"`, `Logo` at
  ≥ 960 px and `LogoTile` below; `frontend/src/app/Sidebar.tsx` drawer brand → `<Logo />` and "SP&A One home";
  `frontend/src/ui/PublicLayout/PublicLayout.tsx` → `<Logo />`; bring the three files' comments within budget
- [X] T015 [P] [US2] `frontend/public/favicon.svg` → the "1" tile (`#003C88`, `#FFC600`, same path as `LogoTile`)
- [X] T016 [P] [US2] `frontend/src/features/dev/ComponentGalleryPage.tsx` Logo card: `<Logo />`,
  `<Logo size="lg" />`, `<LogoTile />`

- [X] T026 [US2] Sign-in title: `ProductName` (the coloured name in the surrounding font) exported from
  `frontend/src/ui/Logo/Logo.tsx` and used in `frontend/src/features/sign-in/SignInPage.tsx`; test in
  `SignInPage.test.tsx` and `e2e/product-name.spec.ts` (US2 scenario 3)

**Checkpoint**: US1 and US2 green — the MVP the client asked for.

## Phase 5: User Story 3 — Names the server writes (P2)

**Goal**: file names, System actor and server messages say "SP&A One".

**Independent test**: quickstart steps 4–6.

- [X] T017 [US3] Update expected strings and file names first in the Go tests that name the product
  (`backend/**/*_test.go`, 19 files incl. `internal/allwork/export_test.go`,
  `internal/purchaserequest/model_test.go`, `internal/settings/service_currencies_test.go`,
  `internal/platform/web/web_test.go`, `internal/financials/handler_test.go`) and see them fail
- [X] T018 [US3] `backend/internal/platform/actor/actor.go` `System.Name = "SP&A One"`;
  `backend/internal/platform/web/web.go` and `backend/internal/settings/handler.go` unreadable message
- [X] T019 [US3] `backend/internal/settings/{purchase_levels,importfile,service_currencies,fieldhelp_catalogue}.go`
  messages and the `plan.name` default help
- [X] T020 [US3] File names: `backend/internal/allwork/export.go` ("SP&A One All work – …"),
  `backend/internal/purchaserequest/form_file.go` ("SP&A One PR form – …", "Routed by SP&A One — …"),
  `backend/internal/financials/template.go` (`spa-one-gsap-extract-sample.csv`)
- [X] T021 [US3] `backend/api/openapi.yaml` descriptions per `contracts/openapi.md`; run `pnpm run api:types`
  (regenerates `frontend/src/api/generated/schema.d.ts`)

**Checkpoint**: all three stories green.

## Phase 6: Polish & cross-cutting

- [X] T022 [P] `docs/product/design-system.md` §2 top bar, §3.1 brand tokens, §3.10, §5 `Logo`/`LogoTile`,
  §6 `PublicLayout`, change log; `docs/product/glossary.md` "SP&A One" entry;
  `docs/product/demo-users-and-data.md` System name
- [X] T023 Backend gates: `go vet ./...`, `golangci-lint run ./...`, `go test -short ./...` (vet, gofmt and tests green
  locally; golangci-lint is not installed on this machine and runs in CI)
- [X] T024 Frontend gates: `pnpm run check`, `pnpm run test:e2e` (product-name and the changed specs) — check green
  locally (api types, typecheck, lint, format, 2348 unit tests); end-to-end left to CI (product owner, 2026-10-28).
  2026-10-29 on develop: CI green, and locally `pnpm run check` (2,536 tests) and the full e2e suite (96/96)
- [X] T025 Check: `grep -rn MAPP frontend/src backend/internal` shows only comments, identifiers, logs and the
  contract title; comment budget of the redesigned files (R1) within 5%

## Dependencies

- T001 → everything (the amendment allows X's new colours and VIII's exemption). T003 → US1, US2.
- US1 and US2 are independent after T003 (they share `TopBar.test.tsx`: do T005 before T014). US3 is
  independent of US1/US2 except the shared fixture name (T009 matches T018).
- Polish after all stories.

## Parallel examples

- After T003: T004, T005, T011, T012 together (different files).
- In US2: T015 and T016 alongside T013.
- In US3: T019 and T020 touch different files once T017 is written.

## Implementation strategy

MVP = US1 + US2 (both P1, the client's visible request). US3 completes the rename in names the server writes.
One pull request against `develop` carrying the amendment, code, tests and documents.
