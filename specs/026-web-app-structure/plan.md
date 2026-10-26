# Implementation Plan: Web-App Structure

**Branch**: `feature/026-web-app-structure` | **Date**: 2026-10-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/026-web-app-structure/spec.md`

## Summary

Align `frontend/` with constitution F1–F10 and Principle V without a visible change. Folders move to
the F2 layout (`ui/`, `session/`, `format/`, `testing/`, `api/generated/`), each feature gets one
`index.ts`, and three merges plus a few moves make the feature graph acyclic. ESLint gains import
boundaries, cycles and accessibility; TypeScript gains the remaining F5 option. Forms move to React
Hook Form + Zod, URL and storage values to Zod with property tests, tests to typed MSW handlers.
Finally every page loads lazily, one chunk per feature, so the first page loads only its own
route's code; its size is measured once and recorded, not enforced (the product owner decided on
2026-10-26 that no JavaScript budget is checked; ADR-0010 keeps gate 12's budget deferred).
Measurements and decisions: [research.md](research.md) R1–R12.

## Technical Context

**Language/Version**: TypeScript 5.9 (strict), React 19, Node ≥ 22.22 (CI uses 24), npm ≥ 11.10

**Primary Dependencies**: React Router 8, TanStack Query 5, `openapi-fetch`, Radix; added
`react-hook-form` 7.91.0, `zod` 4.6.7, `@hookform/resolvers` 5.9.2 (R11)

**Storage**: none changed (browser-storage keys stay as they are)

**Testing**: Vitest 5 + Testing Library; added `msw` 3.0.3, `openapi-msw` 2.1.0, `fast-check`
4.10.2, `axe-core` 4.14.0 (declared); Playwright 1.64 unchanged. Lint adds `eslint-plugin-import-x`
4.17.1 and `eslint-plugin-jsx-a11y-x` 0.2.0

**Target Platform**: the browsers in `browserslist`

**Project Type**: web application; this feature changes `frontend/` and developer documentation only

**Performance Goals**: the first page loads only the shell and its own route's code (today one
374 KiB gzip chunk; 185 KiB in the R10 experiment); measured once after the split, not enforced

**Constraints**: no visible behaviour change (FR-001); every increment leaves `npm run check`,
`npm run build` and `npm run test:e2e` green (FR-017); tasks marked *after 020* / *after 022* start
only when those features are in `plan-module` (R12)

**Scale/Scope**: 1,012 files (288 test files), 38 forms, 68 files over 300 lines, 20 feature
folders becoming 17

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle / standard | Check | Result |
|---|---|---|
| I Spec-first | FR-001–FR-017 with acceptance scenarios; branch `feature/026-web-app-structure` | Pass |
| III Documented design | plan and research; no `data-model.md` or `contracts/`: no data, API or user-flow change. Each feature's interface is its `index.ts`; the validation below replaces `quickstart.md` | Pass |
| IV Two parts | no new runtime part; schemas hold only today's client checks, rules stay on the server (R7) | Pass |
| V Simplicity | each dependency named by F1 or justified (R11); no `config/` until a variable exists (R2); splitting per feature, not finer (R10); files ≤ 300 lines (FR-014) | Pass |
| VI Meaningful tests | today's tests are the reference and stay; new tests only for new parsers (`fast-check`, F10) and the first-page check (US3); lint and compiler settings are not unit-tested | Pass |
| VII Security | versions ≥ 24 h old, licences MIT or MPL-2.0, one install script noted (R11); test-only code banned from app code by lint (FR-011) | Pass |
| VIII Clean construction | new files meet the comment budget; pruning existing comments is 021's | Pass |
| IX, X Experience and palette | nothing visible changes; tokens move to `ui/` with 020's values | Pass |
| F1 Stack | RHF, Zod, MSW, `fast-check`, `jsx-a11y` added; `vitest-axe` left to 027 (R11) | Pass |
| F2 Folders and boundaries | FR-002–FR-005, enforced by lint (R5) | Pass |
| F3 / F4 | URL state parsed with Zod (FR-007); 400 field problems mapped onto fields (FR-006). Query-key factories and other F3/F4 work out of scope | Pass |
| F5 Types | all compiler options, no `any`/`!`/`as T`, MSW handlers typed (FR-008, FR-009, FR-011) | Pass |
| F6 Accessibility | `jsx-a11y` recommended with 0 findings (FR-010) | Pass |
| F8 Performance | routes in one module, split per feature, the first page loads only its route's code (FR-012, FR-013); no JavaScript budget is enforced — gate 12's budget stays deferred in ADR-0010 (product owner, 2026-10-26) | Pass |
| F9 / F10 | file names and import order; MSW, no module mocks, property tests | Pass |
| Gates 7–13 | green after every increment; gate 12's budget stays deferred (ADR-0010 unchanged) | Pass |
| Governance → Alignment | this feature aligns the web app; once merged, every F rule applies to every web-app change | Pass |

**Post-design re-check**: unchanged — Pass. No Complexity Tracking entries.

### Threat model (STRIDE)

No new trust boundary, origin, storage or network call: chunks come from the app's own origin (F1).

| Threat | Where | Response |
|---|---|---|
| Tampering: a compromised package version | the nine added packages | versions ≥ 24 h old (`min-release-age=1`), lockfile, `npm audit` gate; `unrs-resolver`'s postinstall only checks its native binding |
| Information disclosure: test code in the bundle | `msw`, `testing/` | lint forbids importing them outside tests; `tsconfig.app.json` excludes `testing/` |
| Denial of service: a chunk cannot be downloaded | lazy pages | the route error page offers a reload (F4) |

### Risk model

| Risk | Response | Owner |
|---|---|---|
| Conflicts with 020, 021, 022 (they touch most of `src/`) | start Phase 2 after 020 and 022 merge; do not run 021 in parallel; merge each phase into `plan-module` when done | Technical lead |
| A form behaves slightly differently | its component tests stay unchanged and must pass; validation on Save as today (R7) | Team |
| Lazy pages make page tests asynchronous | tests wait with `findBy…`; `renderApp` waits for the route | Team |
| MSW 3 misbehaves under jsdom | the first MSW task keeps `mockApi`'s signature; if MSW fails there, stop and record before migrating tests | Team |
| The `jsx-a11y` fork falls behind ESLint | it supports ESLint 9 and 10 today; the original plugin is the fallback (R5) | Technical lead |
| A large move hides a mistake | `git mv` plus a codemod; type check, lint, unit and e2e tests per increment; review with `git diff -M` | Reviewer |

## Project Structure

### Documentation (this feature)

```text
specs/026-web-app-structure/
├── spec.md
├── plan.md        # this file
├── research.md    # measurements and decisions R1–R12
└── tasks.md
```

### Source Code (target)

```text
frontend/
├── vite.config.ts          # + chunks named after their feature (R10)
├── tsconfig.base.json      # the F5 options, extended by app, test and node configs (R6)
├── eslint.config.js        # + import-x boundaries/cycles/order, jsx-a11y, type rules (R5)
└── src/
    ├── main.tsx            # mounts <App /> only
    ├── app/                # App (providers, styles), routes.tsx (every route), pages.tsx (lazy pages),
    │                       # layouts, navigation, settings sections, error pages
    ├── features/           # 17 features: index.ts · api.ts · schemas.ts · screens · tests
    │   ├── people/         #   + access/ (was features/access)
    │   ├── plans/          #   + campaigns/, phasing/ (were features/campaigns, budget-phasing)
    │   ├── settings/       #   + api.ts (was api/settingsQueries.ts), models/waterfall.ts
    │   └── …               #   activities, activity-log, all-work, approval-workflows, approvals,
    │                       #   dev, financials, global-budgets, home, line-items, notifications,
    │                       #   placeholders, recycle-bin, sign-in
    ├── ui/                 # <Component>/ (was components/), tokens.css, global.css, FormField,
    │                       # useFormProblems.ts, record-links.ts, list blocks from settings/shared
    ├── api/                # client.ts, errors.ts, download.ts, generated/schema.d.ts
    ├── session/            # SessionProvider, useSession, RequireSignIn, RequirePermission,
    │                       # SessionCheckPage, access.ts, gate-action.ts, safe-next.ts, api.ts
    ├── format/             # decimal.ts, intl.ts, text.ts, useFormatters.ts
    └── testing/            # setup.ts, mock-api.ts (MSW), replies.ts (typed answers), render helpers,
                            # fixtures/ (test data per feature), dom.ts, axe.ts
```

**Structure Decision**: the F2 layout with three feature merges (R3); `ui/` without a barrel (R4).

## Increments

Each row is one or more commits that leave every frontend gate and the e2e suite green.

| Phase | Increment | Exit | Waits for |
|---|---|---|---|
| 1 Setup | dependencies; `tsconfig.base.json` with the three zero-error options | `npm run check`, build | — |
| 2 Folders (US1) | generated types; `ui/`; `lib/` split; `session/`; `testing/`; settings blocks; cycle breaks and merges; barrels; API calls into hooks; names | 0 forbidden folders, 0 deep cross-feature imports | 020, 022 |
| 3 Lint (US1, US4) | import-x rules; restricted imports; `jsx-a11y`; no `!`; no `as T` | `npm run lint` clean with every rule | 020, 022 |
| 4 Compiler (US4) | `noUncheckedIndexedAccess` fixed in batches, then on | `npm run typecheck` with all F5 options | 022 |
| 5 Forms and outside data (US2) | form helpers in `ui/`; then per feature: forms, URL and storage parsing with property tests, files ≤ 300 lines | 38/38 forms on RHF + Zod | 020, 022 |
| 6 MSW (US4) | MSW behind `mockApi`; typed handlers; per feature: tests off `vi.mock` and onto typed handlers; `mockApi` removed | 0 `vi.mock` of project modules | 022 |
| 7 Splitting (US3) | one route module; lazy pages per feature; chunks named after their feature; sizes measured once; first-page e2e check | the first page loads only its route's code | 020, 022 |
| 8 Documentation | README, repository layout, CLAUDE.md, feature map | FR-016 | — |

## First page (F8)

No JavaScript budget is checked: the product owner decided on 2026-10-26 that it is not needed, and
ADR-0010 keeps gate 12's budget deferred. The first page loads only the shell and its own route's
code (FR-013). Today it loads one 374 KiB gzip chunk; the R10 experiment gave 185 KiB. T054
measures the first page and each feature chunk once after the split and records them in research
R10.

## Validation

- **US1**: `ls frontend/src`; one deliberate deep import, one lower-layer import of `app/`, one
  cycle and one `fetch` in a component each fail `npm run lint` (checked once in review, not kept).
- **US2**: every form's component tests pass unchanged; the property tests of every URL and storage
  parser pass; hand-edit a URL parameter of All work to nonsense and the page opens with defaults.
- **US3**: `npm run build` lists one chunk per feature, named after it; the e2e check proves
  `/sign-in` loads no other feature's chunk; the sizes are recorded in research R10.
- **US4**: `npm run typecheck` with all options; change a response type in a scratch copy of the
  contract, run `npm run api:types` and `npm run typecheck`: handlers and screens are named.

## Complexity Tracking

No violations to justify.
