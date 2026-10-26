# Feature Specification: Web-App Structure

**Feature Branch**: `feature/026-web-app-structure`

**Created**: 2026-10-26

**Status**: Draft

**Input**: Feature map 026: "Web-app structure: constitution F2 folders, React Hook Form + Zod, MSW,
per-route splitting." Align `frontend/` with constitution 3.1.0 — F1 (stack), F2 (folders and
import boundaries), F5 (compiler options, no `any`, `!` or `as T`), F8 (route splitting and the
first page's code), F9 (file names), F10 (MSW, `fast-check`) and Principle V (files of at most 300
lines) — without changing what anyone sees or can do.

**References**: Constitution 3.1.0 — F1–F10, Principles V, VI, VIII, Governance → Alignment ·
[research.md](research.md) (measurements and decisions R1–R12) · ADR-0010 (gate 12's JavaScript
budget deferred; on 2026-10-26 the product owner decided no budget is checked)

## User Scenarios & Testing *(mandatory)*

The users of this feature are the developers and AI agents who change the web app. People using
MAPP notice nothing, except that the first page arrives with about half the JavaScript.

### User Story 1 - Find code where the constitution says it is (Priority: P1)

As a developer I open `frontend/src/` and find the same top-level folders the constitution lists:
`app`, `features`, `ui`, `api` (with `generated/`), `session`, `format`, `testing`. Each feature
has one public entry, `index.ts`, that says in one line what the feature does. When I import
another feature's internals, import a feature into a design-system component, or create an import
cycle, `npm run lint` tells me before review does.

Anup needs the date formatter. He looks in `src/format/`, finds `dates.ts`, and imports it with
`@/format/dates`. Later he imports `PlanForm` straight from `@/features/plans/PlanForm` into the
activities feature; lint fails with "import another feature through its index.ts".

**Why this priority**: every later change and every other story builds on the layout and its
rules; today 139 imports reach into other features' files and ten features depend on each other
in a cycle (research R1, R3).

**Independent Test**: list `src/`; open any feature's `index.ts`; make one deliberate boundary
violation, one cycle and one direct API call in a component, and check that `npm run lint` reports
each one.

**Acceptance Scenarios**:

1. **Given** the aligned web app, **When** a developer lists `frontend/src/`, **Then** they see
   only `app/`, `features/`, `ui/`, `api/`, `session/`, `format/`, `testing/`, `main.tsx` and
   `vite-env.d.ts`, and no folder named `utils`, `helpers`, `common`, `misc`, `base`, `shared` or
   `lib` anywhere.
2. **Given** a feature, **When** another feature or `app/` imports anything from it other than its
   `index.ts`, **Then** `npm run lint` fails and names the import.
3. **Given** a module in `ui/`, `api/`, `session/` or `format/`, **When** it imports from
   `features/` or `app/`, **Then** `npm run lint` fails.
4. **Given** two modules that import each other directly or through other modules, **When** lint
   runs, **Then** it reports the cycle.
5. **Given** a component (`.tsx`), **When** it imports the API client or calls `fetch`, **Then**
   lint fails; data access goes through a query hook in an `api.ts` module.

---

### User Story 2 - Forms and outside data are checked one way (Priority: P2)

As a developer I build or change a form the same way everywhere: a Zod schema in the feature's
`schemas.ts`, React Hook Form for the form state, and the shared field components. URL state and
stored preferences are parsed with Zod too, so a hand-edited address never crashes a screen.

Mei changes the plan form. She adds the rule to `features/plans/schemas.ts`; the form shows the
message in the error summary on Save, exactly as every other form does. A colleague pastes
`/all-work?view=nonsense&page=-3`; All work opens with its default view and first page.

**Why this priority**: 38 forms each hold their values in `useState` with their own checking code
(R1); one way to do it is easier to learn and to review.

**Independent Test**: for any one form, run its existing component tests unchanged (same
messages, same moment, values kept after a failed save); run the property-based tests of every
URL and storage parser.

**Acceptance Scenarios**:

1. **Given** any form with input fields, **When** the user saves with invalid values, **Then** the
   same messages as before appear at the same moment (on Save, all at once in the error summary and
   at each field), and the values typed are kept.
2. **Given** a save that the API refuses with field problems (400), **When** the answer arrives,
   **Then** each problem is shown at its field and in the summary, as before.
3. **Given** a form with unsaved changes, **When** the user leaves the page, **Then** "Discard
   changes?" appears as before.
4. **Given** any URL parameter or stored preference with an invalid value, **When** the screen
   opens, **Then** it uses the default value and does not crash.

---

### User Story 3 - The first page loads only its own code (Priority: P3)

As a person opening MAPP, I download the application shell and the code of the page I opened, not
the code of every screen. As a developer, I find each feature's screens in a chunk named after the
feature.

**Why this priority**: the constitution requires routes split per feature (F1, F8), but nothing is
broken today; it comes last because it moves every page import. No JavaScript budget is checked:
the product owner decided on 2026-10-26 that it is not needed (gate 12's budget stays deferred,
ADR-0010); the first-page size is measured once and recorded in research.

**Independent Test**: build the app and check that each feature has its own chunk; open the
sign-in page in the browser and check that no other feature's code is downloaded.

**Acceptance Scenarios**:

1. **Given** the production build, **When** it finishes, **Then** each feature's screens are in
   their own chunk, named after the feature, and the first page's chunks hold only the shell; the
   sizes after the split are recorded once in [research.md](research.md) R10.
2. **Given** a visitor who opens `/sign-in`, **When** the page has loaded, **Then** the browser
   has downloaded the shell and the sign-in code, and no code of another feature.
3. **Given** a signed-in person who opens MAPP at `/plans`, **When** the plans code is still
   loading, **Then** the shell (sidebar and top bar) is already drawn and the page area shows the
   loading skeleton it shows today while data loads; a failed download shows the page error with a
   way to reload.

---

### User Story 4 - The type checker and the tests catch contract drift (Priority: P2)

As a developer I trust the types: the compiler options of F5 are on, there is no `any`, no
non-null assertion and no type assertion other than `as const` and the one in the API client.
Component tests answer requests through Mock Service Worker with handlers typed from the generated
API types, so a contract change that breaks a screen fails the type check, and an unexpected
request fails the test.

**Why this priority**: the API-v1 change (022) shows how far a contract change reaches; typed
handlers find every affected test.

**Independent Test**: change one response field's type in a copy of the contract, regenerate the
types, and check that `npm run typecheck` names the handlers and screens that use it.

**Acceptance Scenarios**:

1. **Given** `npm run typecheck`, **When** it runs, **Then** every F5 compiler option is on for the
   app, its tests, the end-to-end tests and the build configuration, with no error.
2. **Given** any source file, **When** it uses `any`, `!` or `as T` (other than `as const` and the
   API client's one response line), **Then** lint fails.
3. **Given** a component test, **When** the screen makes a request that the test did not answer,
   **Then** the test fails and names the request.
4. **Given** the test suite, **When** it is searched, **Then** no test replaces one of the
   project's own modules (`vi.mock`); tests mock the network only.

### Edge Cases

- A file that must move is also changed by 020 (Shell palette) or 022 (API v1): the move waits until
  that feature has merged, so no change is lost (tasks marked *after 020* / *after 022*).
- Two features that genuinely need each other (people and roles; plans and campaigns) become one
  feature with sub-folders, rather than a hidden cycle (R3).
- A shell element needs a feature (the Approvals badge, the notification bell): that feature's
  code is part of the first page.
- A chunk fails to download (a new release replaced it): the route's error page offers a reload;
  it never shows a blank page.
- A test that drew a page synchronously now waits for its code: tests wait for what the user would
  see (`findBy…`), never for a fixed time.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The web app MUST behave exactly as before for its users: the same screens, words,
  validation messages and their timing, navigation, focus behaviour and API requests. Every existing
  component and end-to-end test MUST keep passing, changed only where the test harness changes.
- **FR-002**: `frontend/src/` MUST follow the constitution's F2 layout: `app/`, `features/`, `ui/`
  (components and design tokens), `api/` (client; generated types in `api/generated/`), `session/`,
  `format/`, `testing/`. The folders `components/`, `lib/`, `styles/`, `test/` and every folder named
  `utils`, `helpers`, `common`, `misc`, `base`, `shared` or `lib` MUST be gone. `config/` is created
  when the first build-time variable exists (R2).
- **FR-003**: Each feature MUST have exactly one `index.ts`, opening with one or two lines of
  purpose, that exports what other code uses; it MUST be the only barrel file in `src/`.
- **FR-004**: Dependencies MUST point inward (`app` → `features` → `ui`, `api`, `session`,
  `format`), no feature may import another feature's internals, and there MUST be no import cycle;
  `npm run lint` MUST enforce all three.
- **FR-005**: Components MUST NOT import the API client or call `fetch`; every request goes through
  a query hook in an `api.ts` module of a feature or of `session/`. Lint MUST enforce it.
- **FR-006**: Every form with input fields MUST use React Hook Form with a Zod schema kept in its
  feature's `schemas.ts`. Client checks MUST be exactly today's (same rules, same messages, checked
  on Save and shown all at once); server field problems MUST map onto the fields; typed values MUST
  survive a failed save.
- **FR-007**: URL state and browser-storage values MUST be parsed with Zod schemas; an invalid value
  falls back to the default. Each such parser MUST have a property-based test with `fast-check`.
- **FR-008**: The compiler options of F5 MUST be enabled for app code, tests, end-to-end tests and
  build configuration.
- **FR-009**: `any`, non-null assertions and type assertions MUST NOT appear, except `as const` and
  the single response-typing line of the API client (with a lint directive and reason). Lint MUST
  enforce it.
- **FR-010**: Lint MUST include the `eslint-plugin-jsx-a11y` recommended rules, with no finding.
- **FR-011**: Component tests MUST mock the network with MSW, through handlers typed with the
  generated API types; a request without a handler MUST fail the test; no test may mock a project
  module. Test-only code (`testing/`, `msw`) MUST NOT be imported by application code.
- **FR-012**: All routes MUST be declared in one module of `app/`, and each feature's screens MUST
  load as their own chunk when first needed.
- **FR-013**: The first page MUST load only the shell and the code of the route it opens. Its size
  is measured once after the split and recorded in [research.md](research.md); the build does not
  enforce a size. *The JavaScript budget (the build failing over a recorded size) was withdrawn on
  2026-10-26: the product owner decided no budget is checked, and ADR-0010 keeps gate 12's budget
  deferred.*
- **FR-014**: Every hand-written source file MUST have at most 300 non-blank lines (Principle V),
  and file names MUST follow F9 (components in PascalCase, hooks `useX.ts`, other modules in
  kebab-case).
- **FR-015**: Each added dependency MUST be justified in [research.md](research.md) (purpose,
  maintenance, licence, size, install scripts) and published at least 24 hours before it is added.
- **FR-016**: The developer documentation (`frontend/README.md`,
  `docs/architecture/repository-layout.md`, `CLAUDE.md`) MUST describe the new layout and rules.
- **FR-017**: The work MUST land in increments that each leave `npm run check`, `npm run build` and
  `npm run test:e2e` green.

### Key Entities

- **Feature**: a folder under `features/` for one business area, with its public `index.ts`,
  `api.ts` (query hooks) and `schemas.ts` (Zod schemas).
- **Layer**: a top-level folder of `src/` and the folders it may import.
- **Feature chunk**: the JavaScript of one feature's screens, loaded when one of them is first
  opened.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 0 top-level folders under `src/` outside the F2 list (today 4: `components`, `lib`,
  `styles`, `test`), and 0 forbidden folder names (today 1: `features/settings/shared`).
- **SC-002**: `npm run lint` reports 0 findings with the boundary, cycle, accessibility and
  type-safety rules on (today: 139 deep cross-feature imports, 70 imports of `app/` from lower
  folders, 1 runtime cycle and a 10-feature dependency cycle, 48 accessibility findings, 181 type
  assertions, 13 non-null assertions).
- **SC-003**: `npm run typecheck` passes with all F5 options (today 247 errors with
  `noUncheckedIndexedAccess`).
- **SC-004**: The first page loads only the shell and its own route's code (today it loads every
  screen: one chunk, 374 KiB gzip); the size after the split is measured once and recorded in
  research R10 (an experiment gave 185 KiB). *The enforced limit of 220 KiB was withdrawn on
  2026-10-26 with the JavaScript budget (FR-013).*
- **SC-005**: 38 of 38 forms with input fields use React Hook Form with Zod (39 files draw a form;
  `PublishBlockedDialog` only shows a message); 0 form values held in `useState`.
- **SC-006**: 0 `vi.mock` calls on project modules (today 18 in 10 files) and 0 untyped network
  mocks.
- **SC-007**: 0 hand-written files over 300 non-blank lines (today 68, of which 22 over 400).
- **SC-008**: After every increment, all unit, component and end-to-end tests pass.

## Assumptions

- 020 (Shell palette) and 022 (API v1) merge into `plan-module` before the tasks marked *after 020*
  or *after 022* start; their changes reach almost every folder of `src/` (R12). 021 (comment
  budget) does not run at the same time as this feature's moves.
- Decisions that a clarify step would have asked are recorded in research instead: merging
  `access` into `people` and `campaigns` and `budget-phasing` into `plans` (R3); no barrel in
  `ui/` (R4); `React.lazy` per feature (R10); validation on Save as today (R7); Zod's classic API
  (R7); no `config/` yet (R2).
- Out of scope: the comment budget (021); deleting tests that prove nothing, axe checks in every
  component test and the coverage threshold (027); query-key factories (F3), error-state handling
  beyond field problems (F4), browser-storage key versions (F7), telemetry (F8); the backend.
