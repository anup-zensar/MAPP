<!--
Sync Impact Report
- Version change: 4.2.0 → 4.3.0 (MINOR: the package manager changes; no principle removed)
- Modified principles: VII Security and Privacy by Design — release age enforced by pnpm's
  minimumReleaseAge instead of npm's min-release-age
- Modified sections: Frontend Standards (technology table, repository layout), quality gates 7–13
  and 16 (commands), Amendment History
- Added sections: none
- Removed sections: none
- Files changed with this amendment: frontend/package.json, frontend/pnpm-lock.yaml,
  frontend/pnpm-workspace.yaml (replaces .npmrc and package-lock.json), CI, frontend/Dockerfile,
  frontend/playwright.config.ts, READMEs, agent guidance, GO_STANDARDS, quality-gates, ADR-0010,
  ADR-0012, ADR-0015
- Templates: no change needed (plan-template.md reads its gates from this file)
- Follow-up TODOs:
  - TODO(DARK_TOKENS): dark-theme values for Disabled Text, Link Hover and an error-text red
-->


# MAPP Constitution

MAPP is Shell's marketing planning application that replaces Hive9. This constitution sets the
rules everyone — people and AI agents — follows when specifying, designing, building and testing
it, in the Go API (`backend/`) and the web app (`frontend/`) alike. It applies Shell's Software
Design & Construction Policy (the "Policy") and the compliance points of the Software Engineering
Management System (SEMS), which are mandatory. It may be stricter than either, never weaker.

Normative keywords: **MUST** / **MUST NOT** are mandatory; **SHOULD** / **SHOULD NOT** are
expected, and a deviation needs a recorded justification (Governance → Exceptions). When another
document disagrees with this one, this one wins.

## Core Principles

These principles apply to every part of the system: backend, frontend, data and tooling.

### I. Spec-First and Traceability (NON-NEGOTIABLE)

- No spec, no code. Every change that alters behaviour MUST trace to a requirement in a feature
  specification, `specs/NNN-short-name/spec.md`. A missing requirement is written first, with
  `/speckit-specify`.
- Every feature follows the Spec Kit flow: **specify → clarify → plan → tasks → analyze →
  implement**. `clarify` may be skipped only when the spec has no `[NEEDS CLARIFICATION]` marker.
- A spec says **what** the user can do and **why**, in business language. **How** (technology,
  code, storage) belongs in `plan.md`.
- Requirement IDs (`FR-###`, cited with the feature number where the context does not give it,
  e.g. `004 FR-012`) MUST appear in the names of the tests that verify them and in the pull request
  description. Every functional requirement is covered by at least one task.
- A change in behaviour starts with a change in the spec, in the same pull request as the code.
- Features MUST be designed from independent domain knowledge. Hive9, which MAPP replaces, is
  a source of requirements (Principle II), but the project MUST NOT copy Hive9's or any other
  third party's code, assets, content, copy, visual design, layout, icons, fonts or imagery.

**Rationale:** the spec is the agreement between Shell and the team, and requirements are the
cheapest place to fix a defect. Traceability lets a maintainer find why any behaviour exists.

### II. Shell Business, Plain Language

- MAPP uses Shell's planning vocabulary on every screen: **Plan → Campaign → Initiative →
  Activity → Line item**. Hive9 product words (Program, Tactic, Inspection Window, Honeycomb, HUD)
  are never shown to users. `docs/product/glossary.md` is the authoritative list of terms.
- **Requirements and features come from the available documentation and the Hive9 application
  itself; Shell's process is the functional reference; MAPP's design system is the layout and
  interaction reference; Shell's enterprise design language is the visual reference (Principles IX
  and X).** When sources disagree on behaviour, follow the precedence in
  `docs/reference/README.md` (Shell's own documents first). Every simplification of Hive9 behaviour
  is recorded in `docs/sdd/decision-log.md`.
- The data each level holds, its statuses and its calculations are defined once in
  `docs/product/domain-model.md` and `docs/product/field-catalogue.md`; specs reference them
  instead of redefining them.
- Business configuration is data, not code: models, activity types, custom attributes and their
  option lists, currencies and exchange rates, finance budget items and roles are managed on
  Settings screens, never hard-coded.
- **MAPP owns plans; GSAP owns the money.** MAPP never creates SWBS codes, purchase
  orders or invoices, and shows GSAP's financial transactions read-only. Live GSAP integration is
  out of scope for the current phase: GSAP data (daily exchange rates; purchase-order lines,
  invoices, payments and accruals) is loaded through **simulated feeds** behind adapter interfaces
  (Principle IV, ADR-0009).

**Rationale:** users already know Shell's words and process; they must not have to learn Hive9's
screens.

### III. Documented Design

Each feature MUST deliver written design documentation — the Spec Kit `plan.md`, `research.md`,
`data-model.md`, `contracts/` and `quickstart.md`, plus ADRs — that explains the structure of the
code and why it looks the way it does. In the frontend, `data-model.md` describes the client-side
view models and state, and `contracts/` records the API operations the feature consumes and the
route and component interfaces it exposes. The design MUST:

- Satisfy the documented requirements and be consistent with the solution architecture in
  `docs/architecture/`.
- Contain a threat model (STRIDE or equivalent) for every new trust boundary, data flow or
  externally reachable interface, with a response to each threat, including how cryptographic
  requirements are met. In the frontend this covers each new third-party origin, embedded content,
  use of browser storage and cross-origin call.
- Contain a risk model stating how each identified business risk is addressed or explicitly
  accepted, and by whom.
- Identify each level of intermediate architecture by role and rationale, and record every
  cross-cutting or hard-to-reverse decision as an ADR in `docs/architecture/adr/NNNN-title.md`.
- Map high-level design elements one-to-one onto source modules (a Go package, a frontend feature
  folder), and document each element's interface (names, parameters, return values, types and
  contract) in the design documents well enough that a consumer never needs to read its
  implementation.
- Represent a cost-effective solution for the expected service life.

Model-based diagrams use the appropriate UML dialect (C4 is acceptable for context and container
views).

**Rationale:** about 80% of software cost is maintenance; intent and rationale cannot be
recovered from code alone, so they are written down where maintainers look.

### IV. Two Parts, Server as Source of Truth

- Exactly two runtime parts: a **React + TypeScript web app** (`frontend/`) and a **Go API**
  (`backend/`), storing their data in **PostgreSQL**. Adding a third part (worker, queue, cache,
  separate service) requires an ADR.
- The Go API is one program with one package per business area (plans, activities, line items,
  settings, people, approvals, audit). Each package has the same three layers: **handler (HTTP)
  → service (business rules) → store (storage)**, described in B2.
- **The server is the source of truth.** Every business rule, validation and calculation
  (roll-ups, remaining budget, Sys_Gen_Balance, spreads, currency conversion, statuses) runs in the
  Go API. The web app may repeat a check only to give faster feedback, and never recomputes a
  figure the server provides.
- The API is **REST + JSON**, described in an **OpenAPI 3.1** file in the repository. The web
  app's API types are generated from it, so the two sides cannot drift apart.
- All storage goes through store interfaces defined by the services, implemented on
  **PostgreSQL** in every environment: a developer's machine, CI and, later, Azure Database for
  PostgreSQL. One engine everywhere means no storage migration later. The data model is relational
  (B5).
- **Shell's systems are reached only through adapter interfaces** owned by the package that needs
  them (for example the transaction feed and the rate feed). **Now:** simulated adapters — a
  file-import endpoint (check, then import) and demo seed data. **Later:** the real feed implements
  the same interface, with no change to services or the web app (ADR-0009).

**Rationale:** a small, predictable structure is easy for new developers and AI agents to follow,
and keeps the moves to Azure and Shell's live systems cheap.

### V. Simplicity and Controlled Complexity

- Design and code MUST follow the fundamental design principles: abstract essential (problem
  space) complexity; encapsulate accidental (solution space) complexity; maximise cohesion;
  minimise coupling; anticipate the most likely changes (design to invariants); minimise
  redundancy.
- Build only what the approved spec needs (YAGNI): no speculative options, layers or
  abstractions. Code is written for a beginner to read: clear names, small functions, the same
  folder layout in every package and feature.
- Optimisation MUST NOT be premature. A non-obvious optimisation (in the frontend, for example
  `memo`, `useMemo`, `useCallback`, list virtualisation or code splitting finer than one chunk per
  route) MUST be backed by a benchmark or profile and justified by a non-functional requirement or
  measured value, and its comment states what changed and why.
- Simple, clean code takes precedence over complex, faster code unless a non-functional
  requirement demands otherwise.
- Prefer the standard library and the approved stack (B1, F1). Every new dependency is justified
  (B1, F1).
- Every file, function and expression MUST fall within the quality metrics below. A value in the
  "Needs justification" band MUST carry a written justification (Governance → Exceptions); a value
  in the "Unacceptable" band MUST NOT be merged. In React code the metrics apply to every function,
  including components, hooks and callbacks: a component's props object is one parameter, and
  decisions inside JSX (`&&`, `||`, `??`, `?:`) count toward complexity.

| Metric | Scope | Acceptable | Needs justification | Unacceptable |
|---|---|---|---|---|
| Number of parameters | Function | 0–4 | 5–6 | 7+ |
| Cyclomatic complexity | Function | 1–8 | 9–14 | 15+ |
| Depth of decision nesting | Function | 1–4 | 5–6 | 7+ |
| Fan-out (distinct functions called) | Function | 0–7 | 8–10 | 11+ |
| Length (statements, as counted by `funlen`) | Go function, excluding tests | 1–40 | 41–60 | 61+ |
| Number of terms | Expression | 1–5 | 6–8 | 9+ |
| Length (non-blank lines) | Hand-written source file, tests included | 1–300 | 301–400 | 401+ |

File length counts the non-blank lines of a hand-written source file (Go, TypeScript, TSX, CSS,
SQL); generated files are excluded. A file that grows past the limit is split by responsibility,
not by cutting lines. Fan-out counts the distinct functions and methods a function calls, including hooks and calls
inside its callbacks; language operators and JSX elements do not count. Terms are the operands in
one expression: identifiers, literals and calls.

**Rationale:** uncontrolled complexity is the largest technical cause of project failure after
requirements and project management, and the product owner asked for an application that is
industry-standard but not complex.

### VI. Meaningful Tests (NON-NEGOTIABLE)

- A test exists to catch a defect that could really happen. Every test proves a behaviour someone
  relies on: a business rule or calculation, an access rule, a validation, a status transition, an
  acceptance scenario, an API contract, an error the user sees, a state a screen renders, or a
  reproduced bug.
- These tests are required:
  - unit tests for every business rule and calculation, with edge and negative cases (money,
    roll-ups, spreads, budgets, status transitions, required fields);
  - at least one automated test for every acceptance scenario in a spec, and an end-to-end browser
    test for every P1 user story;
  - a contract test that checks every API endpoint against the OpenAPI file;
  - for every store method that reads scoped records, a test proving records outside the actor's
    scope are not returned;
  - for every view bound to data, a test of each state it renders: loading, empty, error,
    forbidden and success;
  - for every bug fix, a test that reproduces the bug and fails before the fix.
- Tests MUST NOT be written for the sake of having them. Forbidden: tests that restate the
  implementation line by line; tests of the language, a framework, a library or generated code
  (that React renders a prop, that a getter returns its field, that sqlc maps a column); database
  tests of queries, constraints or migrations other than the scope-isolation tests (sqlc checks
  every query against the schema, and end-to-end tests run the migrations); tests
  whose only assertion is that a mock was called; a second test that proves what another test at
  the same level already proves; snapshots of a rendered DOM tree; tests that only execute code to
  raise coverage.
- **Coverage (SEMS):** new and changed code MUST reach at least **80% decision coverage**. The
  frontend measures it as Vitest branch coverage; Go reports statement coverage, so the backend
  measures at least 80% statement coverage on new code **and** tests each outcome of every
  decision that carries a rule, a validation, an access check or a user-visible state. Plain error
  propagation (`if err != nil { return fmt.Errorf("...: %w", err) }`) needs no test of its own.
  Coverage is a floor that the required tests reach; a gap is closed by testing behaviour that
  matters or by deleting code nobody needs, never by a test without a meaningful assertion.
- Tests MUST be committed in the same pull request as the code they cover.
- Tests MUST be deterministic: fixed dates, an injected clock and seed data; no dependence on the
  real clock, execution order, network services outside the test's control, or `sleep`-based
  synchronisation.
- Only two kinds of test touch a real database: end-to-end tests, and the scope-isolation test of
  each store read of scoped records (the scope filter is a security boundary written in SQL, which
  a fake cannot check). Both run against PostgreSQL, the engine and major version of every other
  environment. Every other test (unit, service, handler, contract) uses hand-written fakes and
  needs no database. No test requires Docker.

**Rationale:** planning money must be trustworthy and calculations cannot be checked by eye; but
tests that prove nothing cost time to write, read and maintain, and erode trust in the suite.

### VII. Security and Privacy by Design

- **Demo sign-in** stays until Shell's single sign-on (Microsoft Entra ID) is available (D-004): a
  user picks or types a seeded demo account. Demo sign-in is enabled only in local and development
  environments, by an explicit setting.
- **Role- and team-based access control** (feature 014, ADR-0008): permissions are a fixed list in
  code; roles are data that administrators change in Settings; every role assignment has a scope
  (All markets, My teams or Chosen teams — a market is a team).
- **Authorisation** MUST be enforced by the server on every request, with least privilege, and
  fails closed: every API operation has a declared access rule in one route table, and an
  operation without a rule is refused. Scoped checks run in the service. The web app hides or
  explains actions from the server's decisions, but that is a convenience, never the control.
- **Scope isolation.** MAPP serves one organisation, Shell; the NorthStar rules for tenant
  isolation apply here to access scopes. The scope MUST be taken only from the signed-in actor,
  never from a request body, query string or header. Every read and write of plan-owned data MUST
  be filtered by the actor's scope. A record outside the actor's scope MUST be indistinguishable
  from a missing record (404 `not_found`); 403 `forbidden` answers only an action the actor's
  roles do not allow on a record they can see, and names nothing they cannot see.
- Every request carries the signed-in user (the "actor"); every service operation receives it;
  every change is written to the activity log with who, what and when. Deletes of business records
  go to the recycle bin and can be restored (parents first).
- **Input** MUST be validated at the trust boundary (type, range, length, format, allowlist).
  Output MUST be encoded for its context.
- **Secrets** MUST come from the environment, never from source control, the browser bundle,
  logs, error messages or API responses.
- **Cryptography** MUST use vetted standard libraries; home-grown cryptography is forbidden.
  Randomness for security purposes MUST come from a CSPRNG. Stored passwords MUST be hashed with
  argon2id or bcrypt; only hashes of tokens and other credentials are persisted. Transport outside
  a developer's own machine MUST use TLS 1.2 or later.
- **Personal data** is kept to name, e-mail and job title, MUST NOT be logged, and MUST have a
  defined retention. In the frontend it MUST NOT appear in URLs, console output, telemetry or
  browser storage.
- Money is an exact decimal with a currency code — never a binary floating-point number. Dates
  are calendar dates; timestamps are stored in UTC.
- OWASP ASVS Level 2 and the OWASP API Security Top 10 are the verification baseline.
  Dependencies MUST be scanned for known vulnerabilities on every build; a reachable high or
  critical vulnerability blocks merge.
- **Release age:** a package version — an npm package or a Go module — MUST have been published at
  least 24 hours before it is installed or added. `frontend/pnpm-workspace.yaml` sets pnpm's built-in
  `minimumReleaseAge: 1440` (minutes) and the web app requires that pnpm version; Go has no such setting, so
  the reviewer checks the publication date of every Go module a pull request adds or upgrades.

**Rationale:** Shell's markets must see and change only their own plans, and adding real sign-in
and a database server later must not require rewriting features.

### VIII. Clean Construction

The delivered source code MUST be consistent with the documented design and MUST include
everything needed to build and run it (source, build files, configuration, migrations, lockfile).
In addition:

- **Comment budget.** Comments are rare. In every hand-written source file (Go, TypeScript, TSX,
  CSS, SQL; tests included), the lines that hold a comment MUST NOT exceed **5% of the file's
  non-blank lines**, rounded up: a 20-line file may have one, a 100-line file five. Generated
  files are excluded, and so are tool directives (`//go:`, `//nolint`, `eslint-disable`,
  `prettier-ignore`, `@ts-expect-error` with its reason, `/// <reference>`, goose's `-- +goose`
  and sqlc's `-- name:` annotations).
- **Files over the budget today** are brought within it one at a time: a change that touches a file
  brings the whole file's comments within the budget, not only the lines it changed. Features 025
  (backend layout), 027 (test hygiene) and 028 (query efficiency) are exempt: they change many files
  for another purpose and leave the comments of those files to the next change of each file.
- **What a comment says.** Names, types and small functions carry the meaning. A comment is
  written only for what the code cannot say: why a choice was made, a constraint, a non-obvious
  behaviour, or the measurement behind an optimisation. A comment never restates the code and
  never contradicts it. A Go package or a frontend feature's `index.ts` opens with one or two lines
  stating its purpose; any other doc comment exists only where the name and signature do not
  already state the contract, in one to three lines. Requirement IDs go in test names and pull
  requests, not in source comments. No commented-out code. No `TODO`/`FIXME`/`XXX`/`HACK` markers
  on `main`, `release` or `develop`. History lives in Git, not in source files.
- **Files:** one abstract concept (a type, a component, a hook or a cohesive unit) per file;
  consistent ordering and formatting in every file, enforced by the formatter and the linter; no
  compiler-, platform- or browser-specific extensions without an ADR.
- **Declarations:** every declaration is used; scope is the smallest reasonable one; an inner
  declaration MUST NOT shadow an outer one; built-in and language identifiers MUST NOT be
  redefined; exported package- or module-level state MUST be read-only.
- **Literals:** a literal used more than once MUST be a named constant.
- **Variables:** one meaning per variable; named by meaning, not representation; singular names
  for scalars, plural names for collections; initialised before use.
- **Functions:** the name reflects everything the function does; boolean functions are named
  `IsX`/`isX`, `HasX`/`hasX`, `CanX`/`canX` or similar; preconditions are checked defensively at
  trust boundaries; constructors, factories and hooks return only semantically valid values; input
  parameters — props and state included — MUST NOT be modified unless the contract says so; every
  parameter is used.
- **Statements:** one logical effect per statement; each statement is necessary, and together
  they are sufficient; an intentionally empty block carries a comment; jumps (`goto`, labelled
  breaks) are used only for fatal-error handling in Go and never in TypeScript.
- **Types:** a type represents a higher-level concept than a single function; implementations of
  an interface MUST honour its full contract (Liskov Substitution) or be explicitly documented as
  not substitutable.

**Rationale:** code is read far more often than it is written. Consistency lets readers spend
their attention on the problem, and comments that repeat the code are noise that drifts out of
date.

### IX. Tasteful, Intuitive Experience

A first-time user MUST understand every screen without training, and the interface MUST look like
a modern, well-made product. `docs/product/design-system.md` (version 2) shows how; where it
disagrees with this principle or with Principle X, this constitution wins and the document is
corrected.

**Modern and tasteful**

- The interface follows established design principles: clear hierarchy; alignment to the 4 px
  grid; proximity (related things grouped, unrelated things separated by space before lines);
  repetition (the same component looks and behaves the same everywhere); restraint (every element
  earns its place, nothing is decoration without purpose).
- Shell's enterprise look: a light grey page with white cards and panels, a left sidebar, a
  compact page header, summary cards and plain words. Jost (open licence, self-hosted) for
  headings and Arial for body text; no Shell pecten, logo or typeface until Shell consents
  (D-310–D-314).
- Type sizes, weights, spacing, radii and shadows come only from the design tokens. Cards are flat
  with a subtle border; only floating layers (menus, popovers, dialogs, toasts) cast a shadow. No
  ornamental gradients, glows or illustrations.
- Motion is short (200 ms or less), shows where something came from or went, and honours
  `prefers-reduced-motion`.

**Visual hierarchy**

- Every screen has one purpose, stated by its title, which is the most prominent text on it. The
  most important information and the primary action are where the eye lands first.
- At most **one primary (yellow) action** per page or dialog. Secondary actions are visibly
  quieter; destructive actions are set apart from the others.
- Hierarchy comes from size, weight, spacing and position first; colour is reserved for meaning
  (Principle X).

**No surprises**

- A control does exactly what its label or icon promises, and nothing else. Buttons act, links
  navigate, and selecting a row in a list opens that record. No hidden side effects: closing a
  dialog never saves, changing a filter never changes data, Cancel never deletes.
- The same gesture does the same thing on every screen: Enter submits the focused form, Escape
  closes the top layer without saving, and Back returns to the previous screen with its filters,
  sort and scroll position intact.
- Consequences are shown before they happen: a destructive or consequential action (publish,
  archive, close, delete, remove) states what will change and asks for confirmation; a bulk action
  names how many records it touches; a change that affects other records (roll-ups, approvals)
  says so before saving. Deletes can be undone.
- Feedback is immediate and in place: a pressed control responds within 100 ms, work that takes
  longer than one second shows progress, and a save reports its success or failure where the user
  is looking. Nothing fails silently.
- Layout does not move under the pointer: content loads into reserved space, and nothing appears
  or reorders while the user aims at it.
- A link that opens a new tab says so.

**Obvious and honest**

- **Obvious CRUD.** Every record is created, viewed, edited and deleted with the same pattern: a
  clearly named create button, a searchable list, a details view with an **Edit** action, and a
  delete that says what will be removed and sends it to the recycle bin.
- **Status is always visible.** Every plan, campaign, initiative, activity and line item shows its
  status as a word plus a colour, and a completeness indicator ("3 required fields missing") that
  links to the missing fields.
- Validation shows **all** problems at once, next to the fields, in plain language. Nothing is
  greyed out without saying why and what to do instead. Unsaved changes are protected.
- Screens meet **WCAG 2.2 AA** (detail in F6).

**Rationale:** Shell users describe Hive9 as too complex; MAPP's value is that anyone can plan
in it with confidence, in a tool that looks like the rest of Shell and never does something the
user did not expect.

### X. Shell Colour Palette (NON-NEGOTIABLE)

Every colour in the interface comes from a design token, and the tokens below carry Shell's
values exactly. No other value may fill these roles: no substitutes, tints, shades or opacity
changes.

**Brand colours (both themes)**

| Token | Value | Hex |
|---|---|---|
| Primary Yellow | `rgba(255, 198, 0, 1)` | `#FFC600` |
| Primary Yellow — Hover | `rgba(255, 203, 20, 1)` | `#FFCB14` |
| Primary Yellow — Press | `rgba(235, 182, 0, 1)` | `#EBB600` |
| Secondary Red | `rgba(221, 29, 33, 1)` | `#DD1D21` |

**Theme colours**

| Token | Light | Dark |
|---|---|---|
| Foundation Base (page) | `#F5F5F5` | `#1C1C1C` |
| Surface (cards, panels) | `#FFFFFF` | `#292929` |
| Raised (menus, dialogs) | `#FFFFFF` | `#343434` |
| Text Strong | `#292929` | `#F5F5F5` |
| Text Subtle | `#757575` | `#919191` |
| Link | `#4B78AA` | `#54B8C8` |
| Link Visited | `#9A60A4` | `#AF81B7` |
| Link Hover | `#6E94C0` | TODO(DARK_TOKENS) |
| Link Press | `#4B78AA` | `#54B8C8` (as Link, following the light theme) |
| Border Strong | `#292929` | `#F5F5F5` (as Text Strong, following the light theme) |
| Border Medium | `#757575` | `#919191` (as Text Subtle, following the light theme) |
| Border Subtle | `rgba(28, 28, 28, 0.135)` | `rgba(255, 255, 255, 0.09)` |
| Hover Background | `#EBEBEB` | `#262626` |
| Press Background | `#E0E0E0` | `#121212` |
| Disabled Background | `rgba(28, 28, 28, 0.04)` | `rgba(255, 255, 255, 0.032)` |
| Disabled Text | `rgba(28, 28, 28, 0.38)` | TODO(DARK_TOKENS) |
| Disabled Border | `rgba(28, 28, 28, 0.135)` | `rgba(255, 255, 255, 0.09)` (as Border Subtle) |

**How the palette is used** (contrast ratios measured against WCAG 2.2):

- **Yellow** means "act here" or "you are here": the one primary action per page or dialog, and
  the current-place marker. It is a fill only. Text and icons on yellow use Text Strong of the
  light theme (`#292929`, 7.8:1 or more at rest, hover and press). Yellow is never text, an icon
  or a thin line on a light surface (1.45:1).
- **Red** means danger or error, nothing else. A destructive button is a red fill with white text
  (4.9:1). In the light theme, error text may be red (4.9:1 on Surface, 4.5:1 on Foundation Base).
  In the dark theme red is a fill carrying white text or a white icon only, never text or a thin
  line (3.0:1 on Surface), until a dark error-text token is added (TODO(DARK_TOKENS)).
- **Body-size Text Subtle, Link and Link Visited** sit on Surface (4.6:1) and, in the dark theme,
  on Foundation Base and Surface (4.6:1 or more). Where such text would sit on a background below
  4.5:1 — light Foundation Base (4.2:1), Hover Background (3.9:1) or Press Background (3.5:1),
  dark Raised (3.9:1) — it uses Text Strong instead, unless it is large text (24 px, or 18.7 px
  bold).
- **Link Hover** (3.2:1 on light Surface) colours only large text and icon-only links; a hovered
  body-size link keeps the Link colour and shows hover through its underline.
- **Borders:** Border Subtle for card edges and dividers; Border Medium for the edges of controls
  (inputs, check boxes, outlined buttons; 4.2:1 or more); Border Strong for a hovered or emphasised
  control.
- **Focus** indicators use Text Strong of the active theme, drawn with a 2 px offset (9:1 or more
  against every palette surface and against yellow).
- **Disabled** controls use the Disabled tokens and still say why they are disabled
  (Principle IX).
- **Colours outside the palette** exist only for meanings the palette has no role for: statuses
  and feedback (success, warning, information), planning-level tags, team dots and chart series.
  Each is defined once as a token in `docs/product/design-system.md`, MUST NOT resemble Primary
  Yellow or Secondary Red, MUST meet the WCAG contrast for its use, and MUST NOT be the only
  carrier of meaning.

**Rationale:** the palette is Shell's; using it exactly is what makes MAPP recognisably part
of Shell's estate, and stating the contrast limits next to it lets the palette and accessibility
both be non-negotiable without conflict.

## Current Phase & Scope

- **Scope:** the **Plan module** and what it needs to work — demo sign-in, application shell, home
  dashboard, settings lists and custom attributes, people and teams, plans, campaigns,
  initiatives, activities, line items, activity approvals, planning views, activity log and
  recycle bin — and the features that close the MVP gaps of Shell's requirements: roles, access
  and global budgets (014), read-only financial transactions from a simulated GSAP feed (015),
  exchange-rate history with a simulated GSAP daily-rate import, plan currency and planning
  extensions (016), and approval rules (017). The feature list and build order are in
  `docs/sdd/feature-map.md`.
- **Out of scope for now:** the Finance budgets module, purchase requests, accrual calculation,
  Measure reporting, live GSAP integration, real sign-in (Microsoft Entra ID single sign-on),
  e-mail, AI features. Specs may mention these only as future integration points.
- **Running locally:** the application runs on a developer laptop: the Go API, a local PostgreSQL
  database, and the web app's development server. Servers listen on `127.0.0.1` only. **Docker is
  optional:** a developer installs PostgreSQL natively or starts it with the repository's
  `compose.yaml`. That file can also build and run the whole application from the checkout, for
  reviewers who don't develop it (ADR-0012); every port it publishes is on `127.0.0.1`, nothing is
  pushed to a registry, and developing MAPP never requires Docker or Podman. No cloud services at run time; CI and the
  scans in the quality gates run on GitHub.
- **Demo data:** a clearly labelled, fictitious seed dataset is loaded by an explicit command; it
  never contains real Shell people or figures.

| Area | Now | Later (when Azure is available) |
|---|---|---|
| Web app | React + TypeScript (strict), built with Vite (F1) | Same, served as static files |
| API | Go with Gin, REST + JSON, OpenAPI 3.1 contract (B1, B6) | Same |
| Storage | PostgreSQL behind store interfaces (B5): installed natively or through the optional `compose.yaml` | Azure Database for PostgreSQL, same major version |
| Sign-in and access | Demo sign-in with seeded users; roles, permissions and team scopes kept in MAPP (ADR-0008) | Microsoft Entra ID single sign-on; roles may come from Entra app roles or groups |
| Shell integrations | Simulated GSAP feeds behind adapters (ADR-0009) | Live GSAP feeds implementing the same adapters |
| Visual design | Shell palette (Principle X), Jost and Arial, MAPP's own mark | Shell's typeface and pecten once Shell supplies them and consents |
| Containers | Optional: `compose.yaml` runs PostgreSQL, or builds and runs the whole app locally for reviewers (ADR-0012), with Docker or Podman | Production images, registry and scanning, chosen by ADR before hosting |
| Hosting | Local laptop | Azure Container Apps behind Azure Front Door (see `docs/reference/README.md`) |

## Backend (Go) Standards

These rules apply to everything under `backend/` and refine the Core Principles for Go.
`backend/docs/GO_STANDARDS.md` explains them with code samples.

### B1. Approved Stack and Toolchain

| Concern | Standard |
|---|---|
| Language | Go, pinned by the `go` and `toolchain` directives in `go.mod`; a release the Go team still supports |
| HTTP transport | Gin, confined to the handler layer |
| Database | PostgreSQL in every environment (Principle IV), one major version pinned for all of them |
| Data access | `pgx/v5` connection pool + `sqlc`-generated, type-safe queries |
| Schema migrations | `goose`, SQL migrations embedded in the binary |
| Logging | `log/slog` with the JSON handler |
| Telemetry | OpenTelemetry (traces and metrics) |
| Contract tests | `github.com/pb33f/libopenapi-validator` against `backend/api/openapi.yaml` |
| Testing | standard `testing`, `net/http/httptest`, `github.com/google/go-cmp`; scope-isolation store tests on PostgreSQL databases created per test from `TEST_DATABASE_URL`, every other test on fakes; no test requires Docker (no testcontainers) |
| Static analysis | `golangci-lint` v2 with `backend/.golangci.yml`, `govulncheck`, SonarQube |

- ORMs (e.g. GORM, ent) and runtime auto-migration MUST NOT be used.
- Adding a direct dependency MUST be justified in the feature's `research.md` and the pull request
  (purpose, maintenance activity, licence, why the standard library is insufficient); a dependency
  that shapes the architecture requires an ADR.

### B2. Architecture and Packages

- Layout: `cmd/<binary>/main.go` only wires dependencies and calls `run`; all other code lives
  under `internal/`. Cross-cutting infrastructure lives in `internal/platform/<concern>/`. Each
  business area is a package `internal/<area>/` with the same files: `model.go`, `store.go` (the
  store interface the service needs), `store_postgres.go` (its sqlc-backed implementation),
  `service.go` and `handler.go`, plus tests.
- Dependencies MUST point inward: handler → service → store. Services MUST NOT import Gin or
  `net/http`; stores MUST NOT contain business rules. Import cycles are forbidden.
- Interfaces MUST be defined by the consumer, kept small, and used only where a second
  implementation or a test fake exists. Functions accept interfaces and return concrete types.
- Dependencies MUST be passed through constructors. Package-level mutable state and `init()`
  functions with side effects MUST NOT be used.
- Package names are short, lowercase, singular and descriptive. `util`, `common`, `helpers`,
  `misc` and `base` are forbidden. Exported names MUST NOT stutter (`plan.Service`, not
  `plan.PlanService`).

### B3. Context, Concurrency, and Lifecycle

- Every function that performs I/O, may block, or calls something that does MUST take
  `ctx context.Context` as its first parameter and pass it on. A context MUST NOT be stored in a
  struct or be `nil`. Every outbound call (database, HTTP, feed) MUST be bounded by a deadline.
- Every goroutine MUST have an owner that waits for it, a defined termination condition and a
  path for its error (`errgroup.Group` or `sync.WaitGroup.Go`). Fire-and-forget goroutines and
  unbounded per-request goroutine creation are forbidden.
- Shared mutable state MUST be confined to one goroutine or guarded by a mutex. Only the sender
  closes a channel.
- The server MUST use `http.Server` with `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout` and
  `IdleTimeout` set, MUST shut down gracefully on `SIGINT`/`SIGTERM` within a bounded deadline,
  and MUST close the database on exit.
- `log.Fatal`, `os.Exit` and `panic` MUST NOT appear outside `main`, except `panic` for an
  impossible state that indicates a programming error.

### B4. Error Handling

- Every returned error MUST be handled. Errors MUST be wrapped with context using
  `fmt.Errorf("doing x: %w", err)` and inspected with `errors.Is` / `errors.As`, never by string
  comparison.
- Expected failures are values: sentinel errors (`ErrNotFound`) or typed errors
  (`*ValidationError`) declared by the package that produces them.
- An error is handled exactly once: either logged or returned, never both.
- Domain errors MUST be mapped to HTTP status codes in one place in the transport layer. Internal
  details (SQL, driver, stack) MUST NOT reach the client.
- A recovery middleware MUST convert panics into a logged 500 response.
- Error strings are lowercase and carry no trailing punctuation.

### B5. Data and Persistence

- All SQL MUST live in `.sql` files compiled by sqlc. Building SQL by string concatenation or
  formatting is forbidden; dynamic sorting and filtering MUST map client input through an
  allowlist to fixed SQL fragments.
- **Relational schema:** ids, foreign keys, `NOT NULL`, `CHECK` and unique constraints; no nested
  ownership tricks. The database enforces every invariant it can check.
- **Types:** money is `bigint` minor units, or `numeric` where more scale is needed (exchange
  rates); never `real` or `double precision`. Calendar dates are `date`; timestamps are
  `timestamptz`, handled in UTC.
- **Connections:** one `pgx` pool per process, opened at startup, sized per environment, closed on
  shutdown. Connection strings come from the environment and never appear in logs or errors.
- Every query on plan-owned data MUST filter by the actor's scope in the query itself, not after
  loading (Principle VII).
- Schema changes MUST be goose migrations: reviewed, applied in CI to an empty database, and never
  edited once merged. Migrations are forward-only.
- Transaction boundaries are owned by the service layer and MUST be explicit; a change and its
  activity-log entry are written in the same transaction.
- List endpoints MUST paginate with an enforced maximum page size and MUST NOT issue a query per
  row (N+1).

### B6. API Contract

- An OpenAPI 3.1 document at `backend/api/openapi.yaml` is the contract and MUST be updated in the
  same pull request as any API change.
- Paths are versioned (`/api/v1/...`). Breaking changes require a new version; additive changes
  MUST remain backward compatible.
- JSON field names are `camelCase`. Errors use one envelope:
  `{"error": {"code": "...", "message": "...", "fields": [{"field": "...", "message": "..."}]}}` with
  stable, documented codes; `fields` is a list, so it keeps the order and allows several messages
  for one field.
  List responses use one pagination envelope.
- Request bodies MUST have a size limit. Validation failures return 400 with per-field detail,
  all problems at once.
- Create operations that clients may retry SHOULD accept an `Idempotency-Key` header.

### B7. Observability and Configuration

- Logs MUST be structured JSON via `log/slog` on stdout. Every log record emitted while serving a
  request MUST carry `request_id`, `trace_id` and, once signed in, the actor's id. Tokens,
  passwords, secrets and personal data MUST NOT be logged.
- Inbound HTTP, database calls and outbound calls MUST be traced with OpenTelemetry. Request rate,
  error rate and duration MUST be exported as metrics. In the local phase no exporter sends data
  off the machine.
- The service MUST expose `/healthz` (liveness, no dependencies) and `/readyz` (readiness, checks
  the database).
- Configuration MUST come from environment variables, be parsed into a typed struct and be
  validated at startup; invalid configuration MUST stop the process with a clear message.
  Development-only behaviour (demo sign-in, seeding) MUST require an explicit setting.

### B8. Go Code Conventions

- Code MUST be formatted with `gofmt` and `goimports`. *Effective Go*, *Go Code Review Comments*
  and the *Google Go Style Guide* are the reference where this document is silent.
- Comments follow the comment budget (Principle VIII): a package opens with a one- or two-line
  doc comment; an exported identifier carries a doc comment only where its name and signature do
  not already state its contract, and that comment begins with the identifier's name.
- One primary type per file, named after the type. Within a file: imports, constants, variables,
  types, constructors, methods, then functions.
- Names use MixedCaps with consistent initialisms (`ID`, `URL`, `HTTP`); receivers are short and
  consistent (never `this` or `self`); getters omit `Get`.
- Use concrete types over `any`. Generics are used only to remove real duplication across two or
  more types. Dot imports, naked returns in functions longer than five lines, `unsafe`,
  `//go:linkname` and cgo MUST NOT be used without an ADR.
- Slices and maps received as parameters MUST NOT be mutated unless the contract says so.

### B9. Go Testing

- Tests use the standard `testing` package with `go-cmp` for comparisons; a test with several
  input cases is table-driven with a descriptive name per case, and tests call `t.Parallel()`
  where they share no state.
- Scope-isolation tests of store reads are the only Go tests on a real database: each gets its own
  PostgreSQL database, created from a migrated template on the server named by `TEST_DATABASE_URL`
  (a native install or the Compose service) and dropped afterwards, and skips under `-short`.
  Service, handler and contract tests use fakes. Handler tests use `net/http/httptest`; handler tests are external test packages
  (`package <name>_test`).
- Time-dependent code MUST receive an injected clock or be tested with `testing/synctest`;
  `time.Sleep` in tests is forbidden.
- Parsers and importers (the GSAP file imports) MUST have fuzz tests. Every optimisation MUST be
  backed by a benchmark.
- Test doubles SHOULD be small hand-written fakes of consumer-defined interfaces.
- `go test -race` MUST pass.

## Frontend (TypeScript and React) Standards

These rules apply to everything under `frontend/` and refine the Core Principles for the browser.

### F1. Approved Stack and Toolchain

| Concern | Standard |
|---|---|
| Language | TypeScript in strict mode with the compiler options in F5; no JavaScript source files under `src/` |
| UI library | React with function components and hooks; the major version declared in `package.json` |
| Build and dev server | Vite; `pnpm run build` type-checks with `tsc -b` before bundling |
| Tooling runtime | An Active or Maintenance LTS release of Node.js, declared in `engines.node` |
| Package manager | pnpm (version pinned by `packageManager` in `package.json`, through Corepack) with a committed `pnpm-lock.yaml`; CI installs with `pnpm install --frozen-lockfile`; dependency install scripts run only when allowed in `pnpm-workspace.yaml` |
| Routing | React Router; all routes declared in one module and code-split per feature |
| Server state | TanStack Query for every read and write of API data |
| Forms and validation | React Hook Form with Zod schemas |
| API access | `fetch`, directly or through the thin typed wrapper `openapi-fetch`, used only inside the client module in `src/api/`; types generated from `backend/api/openapi.yaml` with `openapi-typescript` |
| Component primitives, icons, styling, data grid, charts | One library or approach for each, chosen by ADR that records accessibility conformance, maintenance activity, licence and bundle size. Current: Radix primitives, lucide-react icons, CSS Modules with the design tokens |
| Testing | Vitest, React Testing Library with `user-event`, Mock Service Worker (`msw`), Playwright, `axe-core` (through `vitest-axe` and `@axe-core/playwright`), `fast-check` |
| Static analysis | ESLint with a flat config in `frontend/eslint.config.js`, using `typescript-eslint`, `eslint-plugin-react-hooks` and `eslint-plugin-jsx-a11y`; Prettier; `pnpm audit`; SonarQube |

- `package.json` MUST define the scripts `dev`, `build`, `typecheck`, `lint`, `format:check`,
  `test`, `test:coverage`, `test:e2e` and `api:types`. Each quality gate below runs one of them.
- Supported browsers are declared once, in `browserslist` in `package.json`: the two most recent
  major versions of Chrome, Edge, Firefox and Safari, desktop and mobile. No polyfills for
  browsers outside that list.
- Adding a direct dependency MUST be justified in the feature's `research.md` and the pull request
  (purpose, maintenance activity, licence, bundle-size effect, whether it runs install scripts,
  and why the platform or an existing dependency is insufficient); a dependency that shapes the
  architecture requires an ADR.
- Every font, icon set, image and component library MUST have its licence recorded in the pull
  request that adds it, and the licence MUST permit the project's use.
- Scripts, stylesheets, fonts and images MUST be bundled and served from the application's own
  origin. Loading any of them from another origin at runtime requires an ADR.
- `React.StrictMode` MUST remain enabled.

### F2. Architecture and Folder Structure

- Layout:

```text
frontend/
├── index.html
├── package.json              # scripts, engines, browserslist
├── pnpm-lock.yaml
├── tsconfig.json
├── vite.config.ts
├── eslint.config.js
├── e2e/                      # Playwright journeys (F10)
└── src/
    ├── main.tsx              # mounts <App /> and nothing else
    ├── app/                  # providers, router, application shell
    ├── features/
    │   └── plans/            # one folder per business area: plans, activities, settings, ...
    │       ├── index.ts      # the feature's public API; the only import point for other code
    │       ├── PlanEditor.tsx
    │       ├── PlanEditor.test.tsx
    │       ├── api.ts        # TanStack Query hooks for this feature
    │       └── schemas.ts    # Zod schemas for this feature's forms and URL state
    ├── ui/                   # design-system components and the design tokens (Principle X)
    ├── api/                  # the API client; generated/ holds the generated OpenAPI types
    ├── session/              # the signed-in person and route guards
    ├── format/               # Intl-based money, date and number formatting
    ├── config/               # typed, validated environment configuration
    └── testing/              # test setup, MSW handlers, fixtures
```

- Each feature folder is one design element (Principle III). Every screen, dialog and tab traces
  to a feature spec under `specs/` and follows the screen patterns in
  `docs/product/design-system.md`; one that is not described there is specified before it is
  built.
- Dependencies MUST point inward: `app` → `features` → `ui`, `api`, `session`, `format`,
  `config`. `ui` and the modules below it MUST NOT import from `features` or `app`. A feature MUST
  NOT import another feature's internals; it uses the other feature's `index.ts`. Import cycles
  are forbidden. Import-boundary rules in `eslint.config.js` enforce all three.
- Components render and handle events; they MUST NOT call `fetch` or build request URLs. Data
  access goes through the TanStack Query hooks in a feature's `api.ts`, which call the client in
  `src/api/`.
- Business rules live on the server (Principle IV). Interface logic is limited to presentation,
  input validation for fast feedback, and navigation.
- Directory names are short, lowercase and describe what the directory provides. `utils`,
  `helpers`, `common`, `misc`, `base`, `shared` and `lib` are forbidden. Names do not stutter
  (`plans/PlanEditor.tsx`, not `plans/PlansPlanEditor.tsx`).
- A feature's `index.ts` is the only barrel file. Modules use named exports, except where a tool
  requires a default export (for example `React.lazy`).
- An import that crosses a top-level directory of `src/` uses the `@/` alias; an import within one
  top-level directory, or within one feature, is relative.
- Tests sit next to the code they test. End-to-end tests live in `e2e/`.

### F3. State, Data Fetching, and Side Effects

- Server state is owned by TanStack Query. It MUST NOT be copied into component state, a context
  or another store. Each feature builds its query keys with one key factory.
- Local state uses `useState` or `useReducer`. Context is for application-wide concerns only
  (session, theme). A global client-side store requires an ADR.
- The URL is the source of truth for state a user can share or bookmark: filters, sort, page, and
  the selected view or tab. Saved views serialise exactly that state. URL state is parsed with a
  Zod schema before use; an invalid value falls back to the default and never crashes the screen.
- Derived values are computed during render and not stored in state. `useEffect` is only for
  synchronising with something outside React (subscriptions, timers, browser APIs). An effect
  MUST NOT fetch data, derive state or react to a user event, and an effect that subscribes or
  starts a timer MUST return its cleanup.
- Every outbound request MUST be cancellable and bounded by a deadline: the client passes an
  `AbortSignal`, applies `AbortSignal.timeout`, and cancels a request whose result is no longer
  needed.
- Automatic retries are allowed only for idempotent reads, with a bounded count and backoff. A
  write is never retried automatically unless it carries an `Idempotency-Key`, generated once per
  user intent.
- A mutation blocks resubmission while it is pending. Optimistic updates are allowed only where
  the rollback path is implemented and tested.
- A stale response MUST NOT overwrite newer state. A 409 reloads the affected server state and
  shows it.
- The query cache MUST be cleared on sign-out, on session expiry and whenever the signed-in person
  changes, so that one person's data never survives into another session.

### F4. Errors and Resilience

- One client module converts every non-2xx response and every network failure into a typed
  `ApiError` carrying `status`, `code`, `message`, per-field `fields`, and the request ID from the
  `X-Request-ID` response header. No other code parses an error body.
- Status handling lives in one place: 401 goes to sign-in and returns the user to the page they
  asked for; 403 shows a no-permission state; 404 shows one not-found state for a missing record
  and an out-of-scope record alike; 409 shows an actionable message with the current state; 400
  maps `fields` onto the form's fields; 5xx and network failures show a plain message with the
  request ID. A network failure is never shown as an empty result.
- An error is handled exactly once: shown, reported or rethrown, never two of these. A `catch`
  clause declares `error: unknown` and narrows it. An empty `catch` is forbidden.
- The application has a top-level error boundary and an error boundary per route. A fallback
  offers recovery, shows the request ID when it has one, and never shows a stack trace, SQL or
  other internals.
- Every view bound to data renders all of its states: loading, empty, error, forbidden and
  success. Stale data is never presented as a successful result.
- A form keeps what the user typed when a submission fails.

### F5. Types, Data Contracts, and Formats

- `tsconfig.json` MUST enable `strict`, `noUncheckedIndexedAccess`, `noImplicitOverride`,
  `noImplicitReturns`, `noFallthroughCasesInSwitch`, `noUnusedLocals`, `noUnusedParameters`,
  `isolatedModules` and `verbatimModuleSyntax`.
- `any`, non-null assertions (`!`) and type assertions (`as T`) MUST NOT be used. The only
  exceptions are `as const` and the single place in the client module that types a response body.
  The compiler is overridden only with `@ts-expect-error` and its reason; `@ts-ignore` is
  forbidden.
- Data from outside the code is `unknown` until it is parsed: form input, URL state, browser
  storage, file contents and `postMessage` data are parsed with Zod schemas. API response bodies
  are typed by the generated types.
- API types are generated from the OpenAPI contract, committed under `src/api/generated/`, and
  never edited by hand. MSW handlers are typed with the generated types, so a contract change that
  breaks the interface fails the type check.
- Use union types and `as const` objects instead of `enum`. A `switch` over a union is exhaustive
  and ends with a `never` check in `default`.
- Money: the interface displays amounts as the server returns them and MUST NOT add, subtract,
  convert or round amounts to produce a business figure. Amounts are formatted with
  `Intl.NumberFormat` using the record's currency code.
- Dates: a calendar date (`YYYY-MM-DD`) has no time zone and MUST NOT be passed through
  `new Date(string)`, which reads it as UTC and shifts it in other time zones. A timestamp is ISO
  8601 in UTC and is displayed in the user's time zone through `Intl`, using the user's
  date-format preference. Code that needs the current time receives it from an injectable clock.

### F6. Accessibility, Responsiveness, and Consistency

- Every screen MUST meet WCAG 2.2 Level AA. Where a legal or regulatory accessibility requirement
  applies, it applies in addition, and the stricter requirement prevails.
- Every user gets the same experience: navigation, components and behaviour are consistent across
  screens, so a screen is built from the shared components in `src/ui/`, not restyled.
- Semantic HTML first: landmarks, headings in order, lists, tables, real buttons and links. ARIA
  only fills a gap native elements leave. A `div` or `span` with a click handler is forbidden.
- Every control has an accessible name and a visible label (a placeholder is not a label). A form
  error is associated with its field and announced to assistive technology. Identity and password
  fields carry the correct `autocomplete` value, and the interface MUST NOT block paste or password
  managers.
- Everything is operable by keyboard, in a logical order, with a visible focus indicator that
  sticky bars and messages never hide. On a route change, focus moves to the page heading and the
  document title updates. A dialog, drawer or menu moves focus in, keeps it inside, closes on
  Escape and returns focus to the control that opened it; it uses the native `<dialog>` element or
  the component primitives chosen in F1.
- Text contrast is at least 4.5:1 (3:1 for large text); interface components and focus indicators
  at least 3:1 (Principle X lists the palette's limits). Colour is never the only carrier of
  meaning: a status or an over-allocation also appears as text or an icon with a text alternative.
- Interactive targets are at least 24 × 24 CSS pixels. The interface honours
  `prefers-reduced-motion`, works at 200% text size, and reflows without two-dimensional scrolling
  down to 320 CSS pixels; a data table may scroll inside its own container.
- A grid uses the semantic table or the ARIA grid pattern correctly and announces its sort state
  (`aria-sort`). Every chart has an equivalent data table or text summary. Loading outcomes, saves
  and errors are announced to assistive technology.
- Every colour, spacing, type, radius, shadow and breakpoint value comes from one set of design
  tokens, defined as CSS custom properties in `src/ui/`. Components contain no hard-coded values
  for them.
- User-visible text is never assembled by concatenating translatable fragments. Until a
  localisation requirement exists, strings are plain English, and locale-sensitive values use
  `Intl` (F5).

### F7. Frontend Security

- Untrusted text is rendered only through React's escaping. `dangerouslySetInnerHTML`,
  `innerHTML`, `document.write`, `eval`, `new Function`, and string arguments to `setTimeout` or
  `setInterval` MUST NOT be used. The one exception is rich text, rendered only after a sanitiser
  approved by ADR has cleaned it.
- A URL that comes from data is checked against an allowlist of schemes (`https:`, `mailto:` and
  relative URLs) before it reaches `href`, `src`, `window.open` or navigation; `javascript:` and
  `data:` URLs are rejected. A link that opens a new browsing context uses
  `rel="noopener noreferrer"`.
- The production build MUST run under a Content-Security-Policy that forbids inline script and
  `eval`, and MUST NOT need `unsafe-inline` or `unsafe-eval` for scripts. Each third-party origin
  the policy admits is recorded in an ADR.
- Tokens and other credentials MUST NOT be written to `localStorage`, `sessionStorage`, IndexedDB
  or any cookie that script can read. The session is an `HttpOnly`, `SameSite` cookie set by the
  API (`Secure` outside local development); an access token, if the API ever issues one, is held
  in memory only. A credential MUST NOT appear in a URL.
- Browser storage holds only non-sensitive interface preferences. Each key is prefixed
  `mapp.`, carries a version, and is parsed with a Zod schema on read, an invalid value
  falling back to the default. Per-user preferences are cleared on sign-out.
- Only the Web Crypto API is used for cryptography. Client-side encryption is never a substitute
  for a server-side control.
- Any externally hosted script that an ADR admits carries a Subresource Integrity hash.

### F8. Configuration, Observability, and Performance

- Configuration comes from build-time environment variables, and only `VITE_`-prefixed values
  reach the bundle. Everything in the bundle is public, so no `VITE_` variable holds a secret. A
  Zod schema parses the variables into one typed `config` object at startup, and invalid
  configuration stops the application with a clear message.
- The API base is the application's own origin: `/api/v1` through the Vite proxy in development
  and the reverse proxy in production. Development-only behaviour MUST require an explicit build
  flag, and a test MUST prove it is absent from the production build.
- Errors caught by a boundary, unexpected failures, API failures and Core Web Vitals are reported
  through one telemetry module, to a destination chosen by ADR (none off the machine in the local
  phase). A report carries the route and the API request ID, never a token, a password or personal
  data. `console.*` MUST NOT appear in committed code outside that module.
- Performance targets, at the 75th percentile, are the Core Web Vitals "good" thresholds: Largest
  Contentful Paint at most 2.5 s, Interaction to Next Paint at most 200 ms, Cumulative Layout Shift
  at most 0.1. A feature's plan may record a different target with its justification.
- Each plan records a JavaScript budget for the initial route and for each route chunk, and the
  build MUST fail when the initial route exceeds the budget recorded in the repository.
- Routes are code-split, and heavy libraries (grid, charts, editors) load with the screens that
  use them. A list that can render more than 200 rows MUST be virtualised, and grids page on the
  server.

### F9. TypeScript and React Code Conventions

- Prettier formats the code, and ESLint carries no formatting rules. Imports are grouped as
  external, then `@/`, then relative.
- Comments follow the comment budget (Principle VIII). A feature's `index.ts` opens with one or two
  lines stating the feature's purpose; TSDoc appears only where a name and its types do not already
  state the contract.
- Within a file: imports, constants, types, the primary exported item, then private helpers.
- Component files are named in PascalCase after the component (`PlanEditor.tsx`), hooks `useX.ts`,
  other modules in kebab-case. Variables and functions use camelCase, types and components
  PascalCase, named constants UPPER_SNAKE_CASE. Boolean names start with `is`, `has`, `can` or
  `should`. Event-handler props are `onX`, and the functions passed to them are `handleX`.
- Components are plain functions that take one typed props object. Props, state and arguments are
  never mutated; updates build new values. A `key` is a stable identifier, never an array index for
  a list that can change. `const` is the default, and `var` is forbidden.
- No `console.*`, `debugger`, or `TODO`/`FIXME`/`XXX`/`HACK` markers on `main`, `release` or
  `develop`; lint rules enforce this.
- A lint rule is suppressed only with `// eslint-disable-next-line <rule> -- <reason>`, and only
  where Governance → Exceptions allows a justification at the point of deviation. The complexity
  rule in the "Unacceptable" band and every security rule are never suppressed.
- Generics are used only to remove real duplication across two or more types.

### F10. Frontend Testing

- Unit tests cover pure logic. Component tests use React Testing Library and `user-event` and find
  elements the way a user does: by role, label or text. A test ID is a last resort.
- Tests are named by behaviour and cite the requirement IDs they verify, for example
  `describe("004 FR-012: copy a plan", ...)`. A test with several input cases is table-driven with
  `it.each`.
- Component tests mock the network with MSW, and an unhandled request fails the test. A test mocks
  the network, not the project's own modules.
- Automated accessibility checks run in every component test of a screen or dialog and in every
  end-to-end journey, and report no violations. An `axe` rule is disabled only for a named element
  and with a written justification in the test.
- End-to-end tests use Playwright against the real Go API and PostgreSQL (Principle VI).
  They cover sign-in and at least one journey for every P1 user story, and use web-first
  assertions, never fixed waits.
- Tests are deterministic: fake timers or an injected clock, seeded randomness, no `sleep` or
  fixed timeout. Tests are independent and run in parallel.
- Parsers of untrusted input (URL state, saved-view state, file contents) have property-based tests
  using `fast-check`.
- A focused (`.only`) or skipped test MUST NOT be merged. Snapshot tests are limited to small,
  stable values and never cover a rendered DOM tree.
- Vitest exports coverage as LCOV for SonarQube, and its results in a machine-readable report for
  the pipeline. Coverage counts unit and component tests; end-to-end tests do not count toward it.

## Development Workflow and Quality Gates

**Branches:** `main` (production), `release` (stabilisation), `develop` (integration). A module
branch (now `plan-module`) collects one module's features and merges into `develop` through a pull
request when the module is ready. Feature branches are named `feature/NNN-short-name`, matching
the spec folder `specs/NNN-short-name`, start from their module branch (or `develop`) and merge
back through a pull request. `release` merges into `main` only after its quality gates pass.
Nobody pushes directly to `main`, `release` or `develop`: every change reaches them through a
reviewed pull request from a working branch — `feature/NNN-short-name` for a specified feature,
`docs/short-name` or `fix/short-name` for work without a spec, a module branch, or `develop` into
`release` and `release` into `main`. Branch protection on GitHub MUST enforce this.

**Spec Kit flow:** run from the repository root, where `.specify/` lives.

| Step | Spec Kit command | Output | Done when |
|---|---|---|---|
| 1. Specify | `/speckit-specify` | `spec.md`, `checklists/requirements.md` | Product owner agrees the stories, rules and acceptance scenarios |
| 2. Clarify | `/speckit-clarify` | Answers recorded in `spec.md` | No open `[NEEDS CLARIFICATION]` that affects scope |
| 3. Plan | `/speckit-plan` | `plan.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md` | Constitution Check passes, or each violation is justified in Complexity Tracking |
| 4. Tasks | `/speckit-tasks` | `tasks.md` | Every FR has a task, and every required test (Principle VI) has a task |
| 5. Analyze | `/speckit-analyze` | Consistency report | No CRITICAL findings |
| 6. Implement | `/speckit-implement` | Code and tests | All tasks done, quality gates green, demo accepted |

**Commits** follow Conventional Commits and mention the feature number and requirement IDs, e.g.
`feat(005): copy plan (FR-012)`.

**Source control and CI/CD:** all source code, including build and pipeline definitions, MUST be
managed in the approved version control system (TOE default: GitHub). The quality gates below
MUST run in the approved CI/CD system (TOE default: GitHub Actions) and be required checks on
`main`, `release` and `develop`. Each run publishes its test results and coverage reports.

**Quality gates (every pull request; all MUST pass before merge):**

*Backend*

1. `gofmt` and `goimports` report no changes.
2. `go vet ./...` and `golangci-lint run` (with `backend/.golangci.yml`) report no findings.
3. `sqlc generate` produces no diff, and `sqlc vet` passes.
4. All goose migrations apply cleanly to an empty PostgreSQL database.
5. `go test -race -coverprofile ./...` passes, contract tests included, and new-code coverage
   meets Principle VI.
6. `govulncheck ./...` reports no reachable vulnerability.

*Frontend*

7. `pnpm run format:check` reports no changes.
8. `pnpm run typecheck` reports no errors.
9. `pnpm run lint` reports no findings, warnings included.
10. `pnpm run api:types` produces no diff against the committed generated types.
11. `pnpm run test:coverage` passes, accessibility checks included; its results and coverage report
    are published, and new-code coverage meets Principle VI.
12. `pnpm run build` succeeds and stays within the JavaScript budget (F8).
13. `pnpm run test:e2e` passes against the real API and PostgreSQL, accessibility checks included.

*Whole repository*

14. No hand-written source file exceeds the comment budget (Principle VIII) or the file-length
    limit (Principle V); the reviewer checks both.
15. Secret scanning (GitHub Advanced Security and Apiiro) reports no results.
16. Static application security testing (GitHub Advanced Security code scanning, or the SonarQube
    scanning policy) reveals no high or critical vulnerability.
17. Software composition analysis (GitHub Advanced Security, Apiiro or Mend) reveals no high or
    critical vulnerability in any dependency, and `pnpm audit --prod --audit-level high`
    reports none. The reachability exception in Principle VII does not apply to this gate.
18. The SonarQube quality gate passes, and its findings are remediated in line with the quality
    scanning policy.

**Release gates (before every deployment; all MUST pass).** The local phase deploys nothing; these
apply from the first deployment.

1. Every quality gate passed for the commit being deployed, and gates 15, 16 and 17 report clean
   again at deployment time.
2. Dynamic application security testing (TOE default: Invicti) is run against the deployed web
   app and API in a pre-production environment and reveals no high or critical vulnerability.
3. If the application runs on cloud infrastructure or as a container image, cloud security
   posture management scanning (TOE defaults: Wiz for cloud, Rapid7 on premises), including the
   container image, reveals no high or critical vulnerability or weakness, enforced before
   deployment and monitored continuously after it.
4. The change to production is registered in the approved ITIL system (TOE default: ServiceNow)
   before it is deployed, and the change reference is recorded in the release pull request.

**Code review:** every pull request needs at least one approving reviewer other than the author.
The reviewer checks what tools cannot: fan-out and expression-term metrics, file length, naming,
comment value and budget, design-document consistency, threat-model coverage, the tests' worth (Principle VI),
the keyboard and screen-reader behaviour of new interactions, the experience against Principles IX
and X, and that every metric exception is justified. The pull request description lists the
requirement IDs addressed and each metric exception with its justification; for interface changes
it includes before and after screenshots, light and dark, at 320 and 1280 CSS pixels.

**Definition of Done:** tasks complete; every quality gate green; OpenAPI, specs and docs updated;
screens checked against Principles IX and X and `docs/product/design-system.md`; the feature
demonstrated with the seed data.

**Vulnerability and secret remediation:** a critical vulnerability found in the operational
environment MUST be remediated within 30 days and a high-severity one within 60 days. A secret
identified after deployment MUST be revoked and replaced within 30 days. This holds whether or not
the application is in active development, so its build, tests and pipeline are kept runnable.

**Work management:** each feature's work breakdown (the spec's user stories and the tasks derived
from them) MUST be captured on the approved board (TOE default: Azure Boards; alternative: GitHub
Projects), which also records the evaluation and approval of each proposed change. A defect MUST
be tracked there as a work item, sized in the same unit as user stories, with the effort spent
remediating it recorded.

**Asset and export metadata:** the location of the repository MUST be registered in software asset
management and updated whenever it changes. The export control classification number (ECCN) of
the software MUST be recorded in the repository's metadata and kept current, and is reviewed
whenever contributor access or the software's content changes in a way that could alter it.

**Agent assets:** agent skills, instructions and custom agents are treated as code: `.claude/`,
`CLAUDE.md` and everything under `.specify/` (this constitution, templates, scripts and
workflows). They MUST be stored in the approved source control, change only through pull requests
that pass the same review and gates as code, and be scanned for secrets and vulnerabilities with
GitHub Advanced Security. No secret appears in any of them.

## SEMS Compliance

SEMS is mandatory. This table lists the SEMS 2026 early release (30 April 2026) compliance points
and where this constitution meets each one. It adds no rule of its own. Where this constitution is
silent or weaker than a compliance point, the compliance point applies and this constitution is
amended to match.

| SEMS policy | Compliance point | Met by |
|---|---|---|
| Experience Design | Adhere to legal and regulatory requirements, including accessibility, and give all users a consistent experience | Principle IX; F6 |
| Construction | Static code analysis (TOE default: SonarQube) is run and findings are remediated per the quality scanning policy | Quality gate 18 |
| Developer Testing | Unit testing runs in the CI/CD pipeline, with test result reporting | Source control and CI/CD; quality gates 5 and 11 |
| Developer Testing | Decision coverage of changed or new code is at least 80% | Principle VI; quality gates 5 and 11 |
| Security and Privacy | No secrets in source code; Apiiro and GitHub Advanced Security show no results before deployment | Principle VII; quality gate 15; release gate 1 |
| Security and Privacy | SAST on all source code reveals no high or critical vulnerability | Quality gate 16; release gate 1 |
| Security and Privacy | SCA on all software reveals no high or critical vulnerability | Quality gate 17; release gate 1 |
| Security and Privacy | Critical vulnerabilities fixed within 30 days, high within 60, secrets within 30 | Vulnerability and secret remediation |
| Security and Privacy | DAST, if applicable, reveals no high or critical vulnerability | Release gate 2 |
| Security and Privacy | CSPM scanning of cloud infrastructure, if applicable, reveals no high or critical weakness | Release gate 3 |
| Configuration Management | Source code is in an approved version control system (TOE default: GitHub) | Source control and CI/CD |
| Configuration Management | CI in an approved system (TOE default: GitHub Actions) with unit testing, static analysis and composition analysis | Source control and CI/CD; quality gates 5, 11, 16, 17, 18 |
| Configuration Management | Every production change is registered in the approved ITIL system (TOE default: ServiceNow) | Release gate 4 |
| Configuration Management | Asset management is updated for the code repository location | Asset and export metadata |
| Configuration and Work Management | An approved board (TOE default: Azure Boards; alternative: GitHub Projects) captures evaluation and approval of changes and the work breakdown | Work management |
| Work Management | Defects are managed, including effort spent remediating them | Work management |
| Export classification | A current ECCN is recorded for everything that may be exported | Asset and export metadata |
| Third party development | Third parties developing Shell IP adhere fully to SEMS | Governance |
| Agentic Engineering | Agent skills, instructions and custom agents are treated as code in approved source control | Agent assets |
| Agentic Engineering | Agent skills, instructions and custom agents are scanned for vulnerabilities and secrets | Agent assets |

## Governance

- This constitution supersedes every other practice. Where it conflicts with the Policy or with
  SEMS, the stricter rule applies. `/speckit-plan` runs a Constitution Check against it; any
  exception is written in the plan's Complexity Tracking table with a reason.
- **Amendments** are made by a pull request that edits this file and its Sync Impact Report,
  states the rationale and the impact on existing code, includes a migration plan when existing
  code becomes non-compliant, and is agreed by the product owner and the technical lead.
  Versioning: MAJOR when a principle is removed or redefined, MINOR when one is added or materially
  expanded, PATCH for wording.
- **Exceptions:** a value in a "Needs justification" band, or any deviation from a SHOULD, MUST be
  justified at the point of deviation — a `//nolint:<linter> // <reason>` or
  `// eslint-disable-next-line <rule> -- <reason>` directive for a linter finding, or a note in the
  pull request description for a review-only metric. A deviation from a MUST requires an approved
  ADR. A SEMS compliance point may instead be met by an agreed alternative, and the ADR records
  who agreed it and what it covers. A value in an "Unacceptable" band is never merged.
- **Compliance:** `/speckit-plan` and `/speckit-analyze` check every feature against this
  document, and every code review verifies it. SEMS compliance is also assessed by automation and
  by audit; a point reported as not met is treated as a failed gate. Code that predates 2.0.0 is
  brought into compliance as Alignment describes; remaining non-compliance is tracked as backlog
  work.
- **Alignment of existing code:** code written before 2.0.0 is aligned to it by dedicated features
  that the product owner schedules (decided 2026-10-26), not as a side effect of other work. Until
  a package or screen is aligned, a change to it keeps that package's current stack and structure
  (for example `net/http`, the JSON file store and today's folders), and the gates it cannot meet
  yet — the B1 and F1 stack, sqlc and goose, linter findings on lines the change did not touch,
  and the file-length limit in files that existed before 2.0.0 — apply to it from its alignment
  on. The comment budget is aligned file by file instead, by every change that touches a file
  (Principle VIII).
  Everything else in this constitution applies to every change now, and new packages, screens and
  files meet all of it from the start.
- **Third parties:** a third party that develops code in this repository MUST follow this
  constitution and SEMS in full.
- **Enforcement and guidance files:** `backend/.golangci.yml`, `frontend/eslint.config.js` and the
  compiler options in `frontend/tsconfig*.json` enforce the machine-checkable rules;
  `backend/docs/GO_STANDARDS.md` explains the Go rules with examples and is loaded for AI agents by
  `backend/CLAUDE.md`; the root `CLAUDE.md` is the runtime guidance for AI agents. Where any of
  them disagrees with this constitution, this constitution wins and the file is corrected.

### Amendment History

| Version | Date | Change | Decisions |
|---|---|---|---|
| 1.0.0 | 2026-10-18 | Ratified: principles I–VII, Current Phase & Scope, Technology Choices, Development Workflow, Governance | D-001–D-008 |
| 1.1.0 | 2026-10-20 | II: MAPP's design system is the layout and interaction reference; GSAP data from simulated feeds, live integration still out. III: Shell's enterprise design language instead of "white, one blue primary action"; WCAG 2.2 AA instead of 2.1 AA. IV: Shell's systems behind adapter interfaces. VI: role- and team-based access control enforced by the server instead of "every signed-in user can do everything"; demo sign-in stays until single sign-on. Current Phase & Scope and Technology Choices follow (features 014–017) | D-223, D-239, D-242, D-259, D-308, D-310–D-315; ADR-0008, ADR-0009 |
| 1.1.1 | 2026-10-22 | II (wording): requirements and features come from the available documentation and the Hive9 application itself; MAPP's design system is the layout and interaction reference | D-318 |
| 1.1.2 | 2026-10-22 | The product is named MAPP throughout (on `develop`) | D-319 |
| 2.0.0 | 2026-10-26 | The NorthStar backend (1.1.0) and frontend (1.0.0) constitutions merged into this one, NorthStar winning where they disagreed (SEMS): Documented Design, Clean Construction, quality metrics, Backend Standards B1–B9 (Gin, sqlc, goose, OpenTelemetry), Frontend Standards F1–F10, quality and release gates, SEMS Compliance, `main`/`release`/`develop` with `feature/NNN-short-name` branches. Product owner's additions: SQLite now and PostgreSQL later, no Docker; comments at most 5% of a file's lines; meaningful tests only; IX modern, tasteful UI with visual hierarchy and no surprises; X Shell colour palette, non-negotiable (primary yellow `#FFC600`). Scope isolation answers 404 for out-of-scope records | Product owner instruction of 2026-10-26 |
| 2.1.0 | 2026-10-26 | Governance: existing code is aligned by dedicated features, and until then a change keeps its package's stack. Workflow: no direct pushes to `main`, `release` or `develop`; every change arrives by pull request from a working branch, enforced by branch protection. VIII: the comment budget ignores goose, sqlc and other tool directives. Backend Standards point to `backend/docs/GO_STANDARDS.md`; `backend/.golangci.yml` and `backend/CLAUDE.md` added | Product owner instruction of 2026-10-26 |
| 2.1.1 | 2026-10-26 | Merge of 1.1.2 from `develop`: the product is named MAPP in every rule; the Go module is `mapp` and browser-storage keys start `mapp.` | D-319 |
| 3.0.0 | 2026-10-26 | IV: PostgreSQL in every environment from the start (pgx, sqlc, goose), replacing "SQLite now, PostgreSQL later"; Docker optional — `compose.yaml` runs PostgreSQL for those who want it, a native install works as well, and no test requires Docker. V: file length limit of 300 non-blank lines (401+ never merged) | Product owner instruction of 2026-10-26 |
| 3.1.0 | 2026-10-26 | VII: release age — a package version (npm package or Go module) must have been published at least 24 hours before it is installed or added; npm's built-in `min-release-age=1` in `frontend/.npmrc`, reviewers for Go modules | Product owner instruction of 2026-10-26 |
| 3.1.1 | 2026-10-26 | B6: the error envelope's `fields` is a list of `{field, message}` — it keeps the order and allows several messages for one field (feature 022) | Product owner decision of 2026-10-26 |
| 4.0.0 | 2026-10-26 | VI, B9: only end-to-end tests and the scope-isolation tests of scoped store reads use a real database; every other test uses fakes, and database tests of queries, constraints or migrations are forbidden (they were required store tests before) | Product owner decision of 2026-10-26 |
| 4.1.0 | 2026-10-26 | Current Phase & Scope: `compose.yaml` may also build and run the whole application locally for reviewers such as business analysts, published on `127.0.0.1` only and pushed nowhere; Docker and Podman stay optional for developers (ADR-0012) | Product owner instruction of 2026-10-26 |
| 4.2.0 | 2026-10-26 | VIII, Governance: the comment budget is aligned file by file — a change that touches a file brings the whole file's comments within the budget, except in features 025, 027 and 028; feature 021 (a dedicated comment pass) is dropped | Product owner instruction of 2026-10-26 |
| 4.3.0 | 2026-10-27 | F1 and VII: pnpm replaces npm as the web app's package manager (pinned through Corepack, `pnpm-lock.yaml`, `pnpm install --frozen-lockfile`); the 24-hour release age is pnpm's `minimumReleaseAge`, and dependency install scripts run only when allowed (ADR-0015) | Product owner instruction of 2026-10-27 |

**Version**: 4.3.0 | **Ratified**: 2026-10-18 | **Last Amended**: 2026-10-27
