# Research: Web-App Structure (026)

Measured on `feature/019-alignment-guardrails` at `28993f6` on 2026-10-26 (Node 24.18, npm 11.16,
TypeScript 5.9.3, ESLint 10.13, Vite 8): file counts and import graphs with small scripts, compiler
errors with `tsc --noEmit` and one extra option at a time, lint findings with the candidate plugins
installed without saving, bundle sizes from `vite build` (gzip level 9, KiB).

## R1 — What exists today

| Measure | Today |
|---|---|
| Files under `src/` | 1,012: `features/` 754 (20 folders), `components/` 165, `app/` 49, `lib/` 26, `test/` 8, `api/` 6, `styles/` 2; 288 test files |
| Files over 300 non-blank lines (Principle V) | 68 (24 tests); 22 over 400. Largest: `SampleRecordsPage.test.tsx` 729, `ShellAttributeEditorPage.tsx` 682, `PlanForm.tsx` 666, `AllWorkPage.test.tsx` 659, `all-work/testData.ts` 632, `all-work/viewState.ts` 595 |
| Barrel files | 3: `components/index.ts` (imported by 285 files), `features/people/index.ts`, `features/settings/field-help/index.ts` |
| Imports into another feature's files | 139 (110 app code, 29 tests) in 76 files, across 35 feature pairs; only 10 go through a barrel. `app/` imports 52 feature files directly |
| Lower folders importing `app/` | 70 imports (`useSession` 49, `SessionProvider` 8, `navigation` 5, `router` 4, other 4) |
| Import cycles | 1 at runtime (`FormPanel` → `lib/useUnsavedChanges` → `components/index.ts` → `FormPanel`); between features, one cycle through 10 of the 20 features (8 pairs import each other, R3) |
| `as T` (not `as const`) | 181 in 108 files (85 in tests); 1 in `e2e/fixtures.ts` |
| Non-null `!` / `any` / `@ts-ignore` / `enum` / `console.*` | 13 in 8 files (6 in tests) / 0 / 0 / 0 / 0 |
| F5 compiler options missing | `noUncheckedIndexedAccess`: 238 errors in 83 files (79 in 35 app files, 159 in 48 test files) plus 9 in 4 `e2e/` files; `noImplicitOverride`, `noImplicitReturns`, `isolatedModules`: 0 errors each |
| Forms | 39 files draw a form (`<form>`, `FormDialog`, `FormPanel`, `PanelFrame`) in 15 features, 38 with input fields (`PublishBlockedDialog` only shows a message); none uses a form library; values in `useState`, problems from hand-written checks or the API |
| Components reaching the API | no `fetch` outside `api/client.ts`; 5 components call the client directly (`SessionProvider`, `DevOnly`, `LineItemEditPage`, `SignInPage`, `ExportMenu`); 4 use `useQuery`/`useMutation` inline; 3 forms call fetch functions in handlers (`CurrencyForm`, `FinanceBudgetItemForm`, `RenameBudgetDialog`) |
| Test doubles | `mockApi` replaces `fetch` in 121 test files; 18 `vi.mock` calls on project modules in 10 files |
| Outside data | 40 reads of URL search parameters; 3 browser-storage reads (`mapp.theme`, plan-list preferences, line-item grid period); no `VITE_` variable |
| Route splitting | none: `app/router.tsx` imports 41 pages of 18 features; route arrays also in `features/access/accessRoutes.tsx` and `features/settings/settingsRoutes.tsx`. One JS chunk: 1,295 KiB, gzip 374 KiB |
| Lint with `jsx-a11y` recommended | 48 findings: `no-redundant-roles` 30, `no-noninteractive-tabindex` 13, `click-events-have-key-events` 2, three others 1 each |
| Lint with `import-x/order` (F9 grouping) | 33 findings in 26 files (all auto-fixable) |
| File names (F9) | 125 modules that are neither components nor hooks use camelCase (`recordLinks.ts`, `plansApi.ts`) |
| Undeclared dependency | `axe-core`, imported by `features/settings/shared/axeCheck.ts`, arrives only through `@axe-core/playwright` |

## R2 — Folder mapping

| Today | Target (F2) | Notes |
|---|---|---|
| `components/<Name>/` | `ui/<Name>/` | one folder per component, imported as `@/ui/<Name>/<Name>` (no barrel, R4) |
| `styles/tokens.css`, `styles/global.css` | `ui/tokens.css`, `ui/global.css` | F6: tokens live in `ui/`; imported once by `app/App.tsx`, so `main.tsx` only mounts `<App />` |
| `lib/decimal.ts`, `lib/format.ts`, `lib/text.ts`, `lib/useFormatters.ts` | `format/decimal.ts`, `format/intl.ts`, `format/text.ts`, `format/useFormatters.ts` | `format/format.ts` would stutter; `intl.ts` formats dates, times and money with `Intl`, `decimal.ts` reads and writes exact decimals |
| `lib/cx.ts`, `lib/usePageTitle.ts`, `lib/useUndoableDelete.ts`, `lib/useUnsavedChanges.tsx`, `lib/recordLinks.ts` | `ui/` (`record-links.ts`) | interface helpers used by every feature; a record link holds no data access |
| `lib/access.ts`, `lib/safeNext.ts`, `features/plans/gateAction.ts` | `session/access.ts`, `session/safe-next.ts`, `session/gate-action.ts` | the signed-in person's permissions and return address |
| `app/SessionProvider.tsx`, `useSession.ts`, `RequireSignIn.tsx`, `RequirePermission.tsx`, `SessionCheckPage.tsx` | `session/` (with `api.ts` for the session's requests and `page-handle.ts`) | F2: the signed-in person and route guards; `RequireSignIn` now draws the layout it is given, so it needs nothing from `app/` |
| `app/theme.ts`, `useTheme.ts`, `ThemeToggle.tsx` | `ui/` | the session applies the theme, and `session/` may not import `app/` |
| `app/PublicLayout.tsx` | `ui/PublicLayout/` | the sign-in page uses it; a feature may not import `app/` |
| `lib/download.ts` | `api/download.ts` | wraps the client for file answers |
| `lib/waterfall.ts` | `features/settings/models/waterfall.ts` | the models' goal formula; plans and activities use it through `@/features/settings` |
| `api/schema.d.ts` | `api/generated/schema.d.ts` | scripts `api:types`/`api:check`, ESLint ignore, coverage exclude follow |
| `api/settingsQueries.ts` | `features/settings/api.ts` | query hooks belong to a feature |
| `test/*`, `features/settings/shared/axeCheck.ts`, every `testData.ts` | `testing/`, fixtures in `testing/fixtures/<feature>.ts` | `testing/` may import anything (it renders the whole app); nothing outside tests may import it. Fixtures of one feature are used by others' tests too, which may not reach into another feature |
| `features/settings/shared/` | `ui/` (`ListToolbar`, `SettingsCount`, `useListFilters`, `useStatusFlow`, `useCloseAfterSave`) and `features/settings/` (the rest) | `shared` is a forbidden name; the five are used by four features |
| `features/<name>/<x>Api.ts` | `features/<name>/api.ts` (a second resource: `<resource>-api.ts`) | F2 names; F9 kebab-case |

- **Decision**: no `config/` folder yet: there is no `VITE_` variable and no configuration to parse
  (Principle V, YAGNI). The first variable creates `config/` with its Zod schema (F8).
- **Decision**: `features/<name>/schemas.ts` holds the feature's Zod schemas for forms, URL state and
  storage; a sub-area with its own forms (for example `settings/models/`) has its own `schemas.ts`.

## R3 — Breaking the feature cycle

With one public `index.ts` per feature, any two features that import each other form a module
cycle through their barrels. Today's mutual pairs and the decisions:

| Pair | What crosses | Decision |
|---|---|---|
| people ↔ settings, access ↔ settings, approval-workflows ↔ settings | list building blocks from `settings/shared`; the Settings hub's section lists | blocks move to `ui/` (R2); `app/` assembles the hub's sections and passes them to `SettingsPage` |
| access ↔ people | `useTeams`, `firstName` one way; `YourAccessCard`, `PersonRolesFact` the other | **merge**: `features/people/` (people, teams, roles and access) with `access/` as a sub-folder |
| campaigns ↔ plans, budget-phasing ↔ campaigns | plan currency, status words, `RecordFinancials`, the plan tree, `refreshPlanning`, Budget by month | **merge**: `features/plans/` with `campaigns/` and `phasing/` sub-folders — one screen (the campaign panels are child routes of the plan page) |
| activities ↔ approvals | `ActivityStatusChip` and its words | move to `ui/` (also used by approval-workflows and home) |
| activities ↔ line-items | `FormRail`, `countFilled` | move to `ui/` (the form template of design system §2.4) |
| activities → plans, line-items → plans, campaigns → plans | `gateItem`, `allows`, `AmountPair` | `session/gate-action.ts`; `ui/AmountPair/` |

Result: an acyclic order — settings, activity-log, people, approvals, line-items, activities,
plans, then the consumers (home, all-work, financials, global-budgets, approval-workflows,
notifications, recycle-bin, dev, placeholders, sign-in). 20 feature folders become 17.

- **Alternatives**: slot props or callbacks threaded from `app/` (rejected: harder to follow than
  one folder per business area); allowing cycles through type-only imports (rejected: F2 forbids
  cycles; the graph stays simple).

## R4 — Barrels, names and imports

- **Decision**: `features/<name>/index.ts` is the only barrel. `ui/` has none: 285 files change
  `from '@/components'` to `from '@/ui/<Name>/<Name>'`. This also removes today's runtime cycle and
  stops the shell from depending on every component (R10).
- **Decision**: moves use `git mv` (history kept) and a one-off codemod script that is not committed;
  TypeScript and ESLint prove every import resolves.
- **Decision**: renames to kebab-case (F9) happen in the same commit as a file's move, so files move
  once.

## R5 — Lint rules

- **Decision**: `eslint-plugin-import-x` 4.17.1 with its built-in resolver (`createNodeResolver`
  with the `@` alias) — measured identical to `eslint-import-resolver-typescript`, so that package is
  not added:
  - `import-x/no-restricted-paths`: `ui`, `api`, `session`, `format` may not import `features` or
    `app`; `features` may not import `app`; and, from zones built in `eslint.config.js` from the
    folder list, a feature may import another feature only through its `index.ts` (resolved paths,
    so relative imports are caught too);
  - `import-x/no-cycle` (type imports included); `import-x/order` for F9's grouping.
  Measured on today's code: 5 s for the whole of `src/`.
- **Decision**: built-in rules for the rest: `no-restricted-imports` (`@/api/client` in `*.tsx`;
  `msw`, `openapi-msw` and `@/testing/*` outside test files), `no-restricted-globals` (`fetch`
  outside `src/api/`), `@typescript-eslint/no-non-null-assertion`,
  `@typescript-eslint/consistent-type-assertions` (`assertionStyle: never`; `as const` stays
  allowed), plus the existing `no-explicit-any` and `ban-ts-comment` of the recommended set.
- **Decision** (technical lead, 2026-10-26): `eslint-plugin-jsx-a11y-x` 0.2.0, the maintained fork
  that supports ESLint 10, recommended config. The original 6.10.2 stops at ESLint 9 and needs an
  npm `overrides` entry; both give the same 48 findings on today's code. Three rule options keep
  deliberate patterns: `role="list"` on lists (Safari drops the role of a list without bullets),
  focusable scrolling regions and groups (WCAG 2.1.1), and labels with their text three levels
  down; the few other exceptions carry an `eslint-disable-next-line` with the reason.
- **Alternatives**: `eslint-plugin-boundaries` 7.3.0 (no cycle rule, more configuration);
  `no-restricted-imports` alone (matches import text, not resolved paths; no cycles).

## R6 — Compiler options without a big bang

- **Decision**: a shared `tsconfig.base.json` with every F5 option, extended by `tsconfig.app.json`,
  `tsconfig.test.json` and `tsconfig.node.json`. `noImplicitOverride`, `noImplicitReturns` and
  `isolatedModules` are switched on at once (0 errors). `noUncheckedIndexedAccess` is switched on
  last, after its errors are fixed batch by batch while `npx tsc -p tsconfig.test.json --noEmit
  --noUncheckedIndexedAccess` counts what is left; `npm run check` stays green throughout.
- **Fix patterns**: app code narrows (`const first = rows[0]; if (!first) …`) or uses
  `Array.prototype.find`; tests use one helper in `testing/` that returns the element or fails with
  a clear message; no `!`.

## R7 — Forms with React Hook Form and Zod

- **Decision**: field components stay controlled (`value`, `onChange`, `error`); one generic
  `ui/FormField` wraps React Hook Form's `Controller` and passes those props. The error summary
  reads `formState.errors` through one adapter; a 400's `fields` go to `setError` through another.
  "Discard changes?" uses `formState.isDirty`.
- **Decision**: `mode: 'onSubmit'`, `reValidateMode: 'onSubmit'`: today every form checks on Save
  and keeps its problems until the next Save. Schemas contain exactly today's client checks and
  messages; rules only the server knows stay on the server (Principle IV). The form's existing
  component tests are the reference and stay unchanged.
- **Decision**: Zod's classic API (`z.string().min(1, …)`) for readability. Measured: React Hook
  Form 11.7 KiB, Zod classic 23.5 KiB (`zod/mini` 5.5 KiB), resolver 1.8 KiB gzip. They load with the
  first form screen; Zod also reaches the first page through the session's URL parsing.

## R8 — URL state and stored preferences

- **Decision**: each feature's `schemas.ts` defines the schema for its URL parameters and stored
  values; readers use `safeParse` and fall back to the default. Storage keys stay as they are (a
  versioned key would reset everyone's preferences — F7 work, out of scope).
- **Decision**: every such parser gets one `fast-check` property test: any string input never
  throws and yields a valid state, and a valid state survives write-then-read unchanged (F10).

## R9 — Mock Service Worker

- **Decision**: `testing/mock-api.ts` runs MSW's `setupServer`; `mockApi([...handlers])` installs a
  test's answers (a later one wins), records every request, and a last-resort handler answers 501
  and fails the test afterwards, as before. Answers are written `reply('GET /api/v1/plans/{planId}',
  { body: plan }, { planId: 'P-1004' })` (`testing/replies.ts`): the key is an operation of the
  contract and the body is typed with `openapi-msw`'s `ResponseBodyFor`, so a body that no longer
  matches the contract fails the type check (F5). A codemod turned the 1,198 answers of the old
  string-keyed maps into `reply` calls; files whose bodies did not fit the contract yet used a
  temporary `looseReply` until their feature was aligned.
- **Decision**: MSW answers a request a few ticks after `fetch`, when the next test may have begun;
  each request is marked with the test that sent it (a header) and a leftover request of a
  finished test fails as a network error instead of being answered or recorded by the next one.
- **Decision**: router state (the sign-in notice) is outside data too and is parsed with Zod.
- **Decision**: the 18 `vi.mock` calls are replaced by network answers for the data those modules
  load (F10).
- **Alternatives**: typing `mockApi`'s string keys by hand from `paths` (complex template-literal
  types a beginner cannot read); untyped MSW handlers (F5 forbids).

## R10 — Per-route splitting

- **Decision**: `React.lazy` per page, importing the feature's `index.ts`
  (`lazy(() => import('@/features/plans').then((m) => ({ default: m.PlansPage })))` through one
  small helper in `app/`), so each feature is one chunk (F1) and no feature needs a default export.
  A `<Suspense>` around the routed page in each layout shows the page's loading skeleton while its
  code loads, and the shell is drawn at once; a failed download reaches the route's error page
  (F4). All routes, including today's `accessRoutes` and `settingsRoutes`, are declared in `app/`
  (F1).
- **Measured** (experiment: pages imported lazily, everything else unchanged): first-page JS
  185 KiB gzip (from 374 KiB); the largest page chunk 21 KiB. The shell keeps approvals and
  notifications (badge and bell) in the first page.
- **No JavaScript budget** (product owner, 2026-10-26): gate 12's budget stays deferred (ADR-0010)
  and nothing in the build checks sizes. Chunks are named after their feature
  (`build.rolldownOptions.output.chunkFileNames` in `vite.config.ts`), so the first-page end-to-end
  check can tell them apart. After the split, T054 measures the first page (entry chunk and its
  static imports) and each feature chunk once, gzip, and records them here:

  | Measure | Today | After the split (T054) |
  |---|---|---|
  | First page | 374 KiB (one chunk) | to be recorded (experiment: 185 KiB) |
  | Largest feature chunk | — | to be recorded (experiment: largest page 21 KiB) |
- **Alternatives**: React Router's `lazy` route property (on the first load the router draws
  nothing until the page's code arrives — a blank screen where the shell appears today); splitting
  per page (finer than F1's per feature, which Principle V forbids without a measurement).

## R11 — Dependencies added

Every version was published more than 24 hours before 2026-10-26 (constitution VII; npm's
`min-release-age=1` enforces it). Sizes are gzip in a production bundle.

| Package | Version (published) | Licence | Kind | Purpose and notes |
|---|---|---|---|---|
| `react-hook-form` | 7.91.0 (2026-10-23) | MIT | runtime | form state (F1); no dependencies; 11.7 KiB |
| `zod` | 4.6.7 (2026-10-12) | MIT | runtime | schemas for forms, URL and storage (F1, F5); no dependencies; 23.5 KiB |
| `@hookform/resolvers` | 5.9.2 (2026-10-17) | MIT | runtime | `zodResolver`; 1.8 KiB. **6.0.0 (2026-10-19) rejected**: npm fails with ERESOLVE on its optional peers (`effect` 3 vs 4) |
| `msw` | 3.0.3 (2026-10-10) | MIT | dev | network mocking in tests (F1, F10); Node ≥ 22.12, TypeScript ≥ 5.9; no install script |
| `openapi-msw` | 2.1.0 (2026-09-30) | MIT | dev | MSW handlers typed from `paths` (F5); no dependencies |
| `fast-check` | 4.10.2 (2026-09-19) | MIT | dev | property tests of parsers (F10) |
| `eslint-plugin-jsx-a11y-x` | 0.2.0 (2026-05-10) | MIT | dev | accessibility lint (F1, F6); the maintained fork of `eslint-plugin-jsx-a11y` with ESLint 9 and 10 support (R5) |
| `eslint-plugin-import-x` | 4.17.1 (2026-06-28) | MIT | dev | boundaries, cycles, import order (F2, F9); its `unrs-resolver` runs a postinstall check for its native binding and also works with `--ignore-scripts` (verified) |
| `axe-core` | 4.14.0 (2026-10-05) | MPL-2.0 | dev | declares what the test helper already imports; same version `@axe-core/playwright` pins |

Not added: `eslint-plugin-jsx-a11y` 6.10.2 (R5), `eslint-import-resolver-typescript` (not needed, R5), `eslint-plugin-boundaries` (R5),
`vitest-axe` 0.1.0 (last release 2022; component axe checks are 027's).

## R12 — Working alongside 020, 021, 022 and 027

- Their work in progress on 2026-10-26: 020 (Shell palette) changes about 100 files — `styles/`, 31
  component files (mostly CSS modules), 5 in `app/`, 62 in `features/`. 022 (API v1) changes about
  200 — `api/` (client, errors, generated types), every feature's API module and many tests, 7 in
  `lib/`, 6 in `test/`, 13 in `app/`, `e2e/` and `playwright.config.ts`. Neither touches
  `package.json`, `tsconfig*.json`, `eslint.config.js` or `vite.config.ts`.
- **Decision**: only Phase 1 (dependencies, compiler options with 0 errors) starts
  before both merge. Every later task starts once both are in `plan-module`; its *after 020* /
  *after 022* marker names whose files it touches, for the reviewer of a conflict. Each phase merges
  into `plan-module` when it is done, to keep the window for conflicts short.
- 021 (comment budget) must not run during this feature's moves; it can follow it. Files this
  feature creates meet the comment budget already. 027 (test hygiene) follows the MSW phase.
