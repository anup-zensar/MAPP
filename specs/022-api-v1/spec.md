# Feature Specification: API v1

**Feature Branch**: `feature/022-api-v1`

**Created**: 2026-10-26

**Status**: Draft

**Input**: User description: "API v1 (alignment roadmap). Align the existing API contract with constitution
3.1.0, keeping today's stack (`net/http`, JSON file store): every path under `/api/v1`; one error envelope
`{"error": {"code", "message", "fields"}}`; a record outside the person's access answers 404 like a missing
record, 403 only for an action the person's roles do not allow on a record they can see; `/healthz` and
`/readyz` probes outside `/api/v1`."

**References**: Constitution 3.1.0 — B6 API Contract, B7 Observability (health probes), Principle VII
Scope isolation, F4 Errors and Resilience, Governance → Alignment · Feature 014 (access control) ·
Feature 001 research R9 (error format), R11 (contract), R22 (seed refuses while the API runs)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - One versioned contract with one error shape (Priority: P1)

As a developer of the web app (or of any later client) I call the API under `/api/v1` and every failure
comes back in the same envelope, so one piece of client code handles every error, and a later breaking
change can be released as `/api/v2` without breaking today's clients.

**Why this priority**: B6 makes versioned paths and one envelope part of the contract; every later
alignment feature (PostgreSQL, Gin) builds on this contract.

**Independent Test**: call any operation under `/api/v1` and get its usual answer; call the same path
without `v1` and get 404; provoke a validation error, a conflict and a refusal and check that each body is
`{"error": {...}}` with the same code and message as before.

**Acceptance Scenarios**:

1. **Given** the seed, **When** Priya calls `GET /api/v1/plans`, **Then** she gets her plans as before.
2. **Given** the API, **When** anyone calls an unversioned path such as `GET /api/plans` or `GET /api/health`,
   **Then** the answer is 404 with `{"error": {"code": "not_found", "message": "This page or record doesn't
   exist.", "fields": []}}`.
3. **Given** a sign-in with an empty e-mail and password, **When** it is sent, **Then** the answer is 400 with
   `{"error": {"code": "validation_failed", "message": ..., "fields": [every field problem]}}`.
4. **Given** the web app, **When** any page works with records and an action fails, **Then** the person sees
   the same messages as before this feature.

---

### User Story 2 - Another market's records cannot be detected (Priority: P1)

As a local planner I must not be able to tell whether another market's plan, campaign, initiative, activity,
line item, recycle-bin entry, GSAP transaction or budget allocation exists: asking for it by its ID gives the
same answer as asking for an ID that was never used.

**Why this priority**: Principle VII (scope isolation) is a security rule; today the 403 answer tells a
person that a record they cannot see exists.

**Independent Test**: sign in as Mei (Mobility China) and call the plan, an activity and a line item of
*FY2026 Lubricants Europe* by ID, then the same operations with unused IDs; compare status, code and message.

**Acceptance Scenarios**:

1. **Given** the seed, **When** Mei reads, changes or deletes *FY2026 Lubricants Europe* or a record inside
   it by its ID, **Then** the answer is 404 `not_found` with "This <record type> no longer exists — someone
   may have deleted it.", exactly the answer for an unused ID, and nothing is changed.
2. **Given** the seed, **When** Sara (Viewer) changes a plan she can see, **Then** the answer stays 403
   `forbidden` with the role message.
3. **Given** a recycle-bin entry about a plan Mei cannot see, **When** she previews, restores, undoes or
   deletes it permanently, **Then** the answer is 404 "This item is not in the recycle bin.".

---

### User Story 3 - Probes tell whether the API is alive and ready (Priority: P2)

As whoever runs MAPP (a developer's script, the end-to-end test runner, later a container platform) I ask
`/healthz` whether the process answers and `/readyz` whether it can use its data, without signing in.

**Why this priority**: B7; the end-to-end runner and the seed command need a probe, and a deployment needs
both later.

**Independent Test**: start the API and call both probes; remove its data folder and call `/readyz` again.

**Acceptance Scenarios**:

1. **Given** a running API, **When** `GET /healthz` is called, **Then** it answers 204 without a body.
2. **Given** a running API with a usable data folder, **When** `GET /readyz` is called, **Then** it answers 204.
3. **Given** a running API whose data folder has gone, or whose store refused a half-finished save, **When**
   `GET /readyz` is called, **Then** it answers 503 with the error envelope (`internal_error`).
4. **Given** an API running on `APP_PORT`, **When** the seed command runs, **Then** it refuses as before.

### Edge Cases

- A person without the permission at all (Sara deleting a plan) gets 403 with the role message whether the
  record exists, is outside her access or was never there: the role check comes first (014 D-228) and names
  nothing of the record.
- A record outside the person's access that is also missing still answers 404.
- An activity the person owns or must approve, in a plan they cannot see, stays visible (014 FR-006).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001** (B6): Every operation of the API MUST be served under `/api/v1/…`; the unversioned `/api/…`
  paths MUST answer 404 `not_found`.
- **FR-002** (B6): The contract (`backend/api/openapi.yaml`) MUST describe every operation with its
  `/api/v1` path, and the web app MUST call only those paths.
- **FR-003** (B6): Every error answer MUST be `{"error": {"code", "message", "fields", …}}`, the inner object
  being today's error object (`conflict`, `parent`, `resolution`, `missing` stay where they apply); codes,
  messages and statuses stay as they are, except FR-005.
- **FR-004** (F4): The web app MUST read the envelope in its one client module; no other code reads an error
  body.
- **FR-005** (VII): A request whose path names a plan-owned record outside the person's access (a plan,
  campaign, initiative, activity, line item, recycle-bin entry, GSAP transaction or budget allocation) MUST
  answer 404 `not_found` with the not-found message of that record, and change nothing.
- **FR-006** (VII): 403 `forbidden` MUST answer only an action the person's roles do not allow (no
  permission, a permission for other teams than the record's plan, segregation of duties), and MUST name
  nothing of a record they cannot see.
- **FR-007** (B7): `GET /healthz` MUST answer 204 while the process runs, with no dependency and no session.
- **FR-008** (B7): `GET /readyz` MUST answer 204 when the store is usable and 503 with the error envelope
  when it is not, with no session.
- **FR-009**: `GET /api/health` MUST be removed. The web app's question "is demo sign-in on" MUST be answered
  by `GET /api/v1/environment` (`{environment, demoSignIn}`, no session); the seed command and the
  end-to-end runner MUST use `/healthz`.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of the contract's operations are under `/api/v1`, except `/healthz` and `/readyz`; the
  contract test serves exactly the contract's operations.
- **SC-002**: Every error answer checked by the contract tests validates against the envelope schema.
- **SC-003**: For a plan, an activity and a line item outside Mei's access, the status, code and message are
  identical to those for an unused ID.
- **SC-004**: Every backend and frontend gate and the end-to-end suite pass with no change in what people see.

## Decisions

- **`fields` stays a list** of `{field, message}`: B6 asks for per-field detail; the list keeps the order
  of the form and is what every screen reads today. Turning it into a map is left to the product owner
  (follow-up).
- **Not-found message**: an out-of-scope planning record answers "This <type> no longer exists — someone may
  have deleted it.", the message its own operations give for a missing record; a bin entry answers "This
  item is not in the recycle bin."; a GSAP transaction "This GSAP transaction doesn't exist."; an allocation
  "This allocation no longer exists — someone may have removed it.".
- **`/readyz` 503** uses the existing code `internal_error`, so no code is added.
- **Health information**: `environment` and `demoSignIn` move from `/api/health` to
  `GET /api/v1/environment`; `/healthz` and `/readyz` are in the contract, tag System, as public operations.

## Conflict with feature 014

014 FR-008, User Story 1 scenario 3, the Edge case "Notifications and links" and the Assumption
"Refusals: 403 for a record outside the person's access" say that a record the person cannot see answers 403
and shows "You don't have access to this <record type>". Constitution 3.1.0 (Principle VII, F4) and this
feature replace that with 404 and the not-found page. 014's 403 for role refusals (FR-010, FR-012) stays.

## Assumptions

- Only records named in the request path are covered. Records named in a body — move and copy destinations
  (005 campaign moves answer 403 for a destination plan outside the person's teams), GSAP link targets — keep
  today's answers (follow-up).
- A few operations word a missing parent differently from its own operations (007: "This activity was
  deleted — restore it from the recycle bin …"); their out-of-scope answer uses the record's own message
  (follow-up to align the wording).
- Out of scope (later features): a pagination envelope for lists, `Idempotency-Key`, Gin, PostgreSQL,
  OpenTelemetry, package renames, comment pruning, splitting long files.
