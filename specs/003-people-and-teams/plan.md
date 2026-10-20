# Implementation Plan: People and Teams

**Branch**: `003-people-and-teams` | **Date**: 2026-10-19 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/003-people-and-teams/spec.md`

**Design artifacts**: [data-model.md](data-model.md) · [contracts/openapi.yaml](contracts/openapi.yaml) (a
fragment merged into `backend/api/openapi.yaml`) · [tasks.md](tasks.md). Conventions are feature 001's
([plan](../001-app-shell-and-sign-in/plan.md), [research R1–R24](../001-app-shell-and-sign-in/research.md),
[data model](../001-app-shell-and-sign-in/data-model.md)) plus the Settings-record conventions feature 002
designs at the same time (`POST …/status` with `StatusUpdate`, `POST …/impact` answering `ChangeImpact`,
`409 in_use`); this plan states only what is new. Research decisions are in *Decisions* (no research.md).

## Summary

Feature 003 turns 001's six seeded demo people into a managed directory and adds teams:

- **Go API** — the existing package `internal/people` grows (one business area, constitution IV): people
  are created, edited, deactivated and reactivated (never deleted); **teams** (table `teams`) and **team
  memberships** (table `team_memberships`) are created, edited, set inactive, reactivated and deleted to the
  recycle bin with Undo; an **impact check** warns before a membership removal leaves a plan's owner outside
  the plan's teams (R1); My profile (`PATCH /api/me`); two new preferences, **display currency** and **date
  format**; the person and team references (`PersonRef`, `TeamRef`) and the lookup functions later features
  use (including the display rate their APIs convert amounts with); **hooks** through which features 002,
  004, 005, 006 and 008 give people and teams their currencies, plans and usage counts without import
  cycles; seed teams and memberships.
- **React web app** — Settings › **People** and Settings › **Teams** (standard CRUD pattern with side
  panels), **My profile** replacing 001's placeholder, the **person picker** (single, multiple and "only
  members of the selected teams") and team picker every later form uses, person and team labels that always
  show "(inactive)", and `useFormatters()`, through which every page shows and reads dates in the person's
  date format.

Approach: copy `internal/samples` and `features/dev` (constitution VII), reuse 001's history core for the log
and the recycle bin, keep stored records relational (people, teams, memberships, preferences), and let the
server decide every rule — including the R1 owner warning and the display-currency conversion.

## Technical Context

**Language/Version**: Go 1.27 (API); TypeScript 5.9 strict + React 19.3 (web app) — as feature 001

**Primary Dependencies**: none new. API: standard library (+ 001's test-only libopenapi-validator). Web
app: 001's react-router, TanStack Query, radix-ui, lucide-react, openapi-fetch

**Storage**: 001's JSON file store; new tables `teams.json`, `team_memberships.json`; `preferences.json`
gains two fields (001's records stay valid). New readable-ID counters `T` and `TM`

**Testing**: Go `testing` + `apitest` (contract-validated); Vitest + Testing Library; Playwright + axe

**Target Platform**: as feature 001 (developer laptops, Chromium/Edge)

**Project Type**: web application (React SPA + Go REST API)

**Performance Goals**: SC-002 — with 1,000 people the picker narrows while typing in well under 1 second:
`GET /api/people` returns everyone once (cached by TanStack Query) and the picker filters in memory

**Constraints**: 127.0.0.1 only; personal data limited to name, e-mail and job title, never in server logs;
WCAG 2.2 AA and keyboard-only use (SC-008); 390 px wide without horizontal page scroll

**Scale/Scope**: demo: 6 people, 3 teams; designed for Hive9's scale (≈250 people, a team of 250, a person
in 85 teams). 15 new API operations (+3 with larger answers), 3 routes, 12 new or changed shared web parts

## Constitution Check

*GATE: checked before design and again after it.*

| Principle | How this plan complies | Result |
|---|---|---|
| I. Spec first | Every element traces to an FR, scenario or edge case (*Requirement traceability*). The spec has no `[NEEDS CLARIFICATION]`. Two additions beyond the spec's text are proposed decisions for the product owner: the team colour (already decided as D-025) and gender-neutral wording | Pass |
| II. Shell business, plain language | Plan, team, owner, approver; no Hive9 terms; every text in *User-visible messages* | Pass |
| III. Simple, friendly experience | Standard CRUD pattern (list, search, details side panel, **Edit**, delete with Undo or set inactive); status chips Active/Inactive as word + colour; every problem at once; "(inactive)" everywhere; nothing disabled without its reason (self-deactivation, picker without teams) | Pass |
| IV. Simple, layered architecture | One package `people` (handler → service → repository); all storage through `people.Repository`; every rule in the service, including the R1 impact check; display conversion stays in the API (004–012 convert with `DisplayRateOf`) | Pass |
| V. Tested business rules | A Go test per rule named with its FR; a contract test per operation; a component test per page and shared part; Playwright journeys for US1–US3 (P1) and US4 | Pass |
| VI. Safe by default | Actor on every service call; every change logged in the same transaction; personal data = name, e-mail, job title only; inactive people cannot sign in (001's `Authenticate` ends their sessions); no secrets | Pass |
| VII. Keep it small | No new dependency; people, teams and memberships in one package; client-side filtering instead of a search API; hooks are plain function fields | Pass |

**Post-design re-check**: Pass — no new runtime part, dependency or exception; the extra moving parts are
in *Complexity Tracking*.

## Decisions

The spec has no `[NEEDS CLARIFICATION]` marker. Open questions use the decision-log defaults: **Q-006** — no
plan or record is hidden from anyone (FR-026); roles are not enforced (D-004, FR-027).

Proposed decisions (numbered at consolidation, 2026-10-19):

| # | Proposed decision | Why |
|---|---|---|
| D-056 (P1) | Teams and memberships live in the existing Go package `people`; readable-ID prefixes **`T`** (team, `T-1001`) and **`TM`** (membership, internal) | One business area (constitution IV); every stored record gets its id from `ids.Generator` |
| D-057 (P2) | **A membership change is a change of both records**: each person and each team whose membership set changes gets a new version and one `changed` log entry ("Teams" on the person, "Members" on the team) | FR-012 (two edit points) + 001 FR-029: an edit form opened before the change gets `version_conflict` instead of silently undoing it; feature 011's History of either record shows it |
| D-058 (P3) | **R1 owner warning (FR-013)** follows feature 002's confirm-before-save pattern: `POST /api/teams/{teamId}/impact` and `POST /api/people/{personId}/impact` take the update body, save nothing and answer `ChangeImpact` with one `ownerNeedsAttention` warning per affected owner. The web app calls it only when the form removed members or teams; warnings → **Keep editing** / **Save changes**. The save itself is always allowed. Plans in the recycle bin are not part of the warning | Server decides R1 (constitution IV); one pattern for every "confirm before saving" in Settings (002 models, currencies; 003) |
| D-059 (P4) | Error codes: **`in_use`** (feature 002's code, also answered by a team used by plans) and a new **`not_allowed`** (409: deactivating yourself) | 001's codes do not express these refusals; a code lets the web app pick the right message |
| D-060 (P5) | Usage comes through **hooks**, not imports: `UsePlans(PlanSource)` (004), `AddPersonUsage(PersonUsage)` (005, 006, 008), `UseCurrencies(CurrencySource)` (002, default `OnlyUSD`) | 004–008 import `people` for references; `people` importing them would be a cycle. Same idea as 001's `history.Kind` |
| D-061 (P6) | **Display conversion happens in the API**: features 004–012 convert USD amounts in their answers with `people.DisplayRateOf(ctx, actorID)` and 002's `money.FromUSD`, labelled with the currency; amounts typed in another currency are answered as they are. The web app formats the `{amount, currency}` it receives and never converts | Constitution IV (currency conversion on the server); matches 002's `money.FromUSD` and 004's plan |
| D-062 (P7) | Display currency and date format are **preferences** (as D-038): stored with theme and sidebar, no version, not logged | They change how a person sees data, not business data; FR-028 lists no preference change |
| D-063 (P8) | A display currency set inactive in Settings **stays in use** with its current rate (spec edge case); feature 009's edge case "falls back to USD" conflicts with it and should follow this one | The owning feature's spec wins; consolidation aligns 009 |
| D-064 (P9) | Spec sentences with "she/her" are written **gender-neutrally** with the first name or "them" ("Priya won't be able to sign in…", "reactivate them instead") | NorthStar stores no gender (constitution VI: personal data limited) |
| D-065 (P10) | **Team colour** (D-025): 8-colour palette (Blue, Violet, Teal, Magenta, Orange, Green, Slate, Gold), assigned automatically (the colour used by the fewest teams) and changeable in the team form; shown as a dot in team pills. Spec FR-011 should gain this sentence | D-025 and domain model §6 already decided it; 004 and 009 show coloured team pills |
| D-066 (P11) | The person and team lists filter **All / Active / Inactive** with **All** as default | US1-5: "afterwards she is listed as Inactive" — the person must stay visible after deactivation |
| D-067 (P12) | "What they own or approve" counts records **not** in the recycle bin; a team's "Used by" **includes** plans in the recycle bin (marked) | FR-004 counts working records; FR-014 includes the bin for the delete guard, as 002 FR-003 does for "Used by" |
| D-068 (P13) | My profile shows two cards, **Your details** (name, job title, e-mail) and **Display preferences**, each with its own **Edit** side panel and one request | One save = one request with one meaning (logged person change vs. unlogged preferences) |
| D-069 (P14) | Pickers filter **in the browser** over `GET /api/people` (everyone, with teams); no search endpoint | SC-002 at 1,000 people is instant in memory; one cached list serves the People page and every picker |
| D-070 (P15) | Memberships follow one rule from both sides: only **active** people can be added to a team and only **active** teams to a person; existing memberships with an inactive person or team may stay | Domain model §6: inactive records are not offered for new choices but are kept |
| D-071 (P16) | People and teams use 002's status action `POST …/status` with `StatusUpdate {active, version}` instead of separate deactivate/reactivate operations | One status action for every Settings record |

## Project Structure

### Documentation (this feature)

```text
specs/003-people-and-teams/
├── spec.md · checklists/requirements.md
├── plan.md                 # this file (decisions included)
├── data-model.md           # stored records, rules, transitions, kinds, seed
├── contracts/openapi.yaml  # fragment merged into backend/api/openapi.yaml
└── tasks.md
```

### Source code (new `+`, changed `~`)

```text
backend/
├── api/openapi.yaml                    ~ merged fragment (paths, schemas, Error code not_allowed, ImpactWarning kind)
├── internal/
│   ├── platform/errs/errs.go           ~ not_allowed (+ in_use if 002 has not added it) + constructors (+ errs_test.go)
│   ├── app/app.go                      ~ people service after history; registers people kinds
│   └── people/
│       ├── model.go                    ~ Team, Membership, Preferences fields, palette, date formats, API views and inputs
│       ├── messages.go                 + every user-visible text of this package
│       ├── repository.go · store_json.go   ~ teams, memberships, ListPreferences
│       ├── service.go                  ~ NewService(+ids, history, clock), hook setters, CurrentUser, preferences, profile, display currencies
│       ├── people.go                   + person operations and checks
│       ├── teams.go                    + team operations and checks
│       ├── memberships.go              + applying membership changes on both sides; owner impact (FR-013)
│       ├── usage.go                    + hook types, OnlyUSD, usage counts, lookup functions for other features
│       ├── kinds.go                    + history kinds person, team, teamMembership; Describe; PrepareRestore
│       ├── handler.go                  ~ Routes + /api/me operations
│       ├── people_handler.go · teams_handler.go   + HTTP for people and teams
│       └── *_test.go                   ~/+ model, service, people, teams, memberships, usage, kinds,
│                                         handler (me), people_handler, teams_handler
├── seed/seed.go · seed_test.go         ~ teams and memberships
└── cmd/seed/main.go · main_test.go     ~ summary line "… 6 people, 3 teams."

frontend/
├── src/
│   ├── api/schema.d.ts                 ~ regenerated
│   ├── api/errors.ts (+ test)          ~ kinds inUse, notAllowed; optional dateFormat in conflictMessage/resolvedMessage
│   ├── app/router.tsx · navigation.ts  ~ routes; My profile leaves COMING_SOON_SECTIONS
│   ├── styles/tokens.css               ~ --team-<colour> tokens
│   ├── lib/format.ts (+ test)          ~ date formats: formatDate/formatDateTime(…, format), parseDate
│   ├── lib/useFormatters.ts (+ test)   + date, dateTime, parseDate, money, moneyCompact bound to the person
│   ├── lib/useUndoableDelete.ts        ~ resolvedMessage with the person's date format
│   ├── components/                     + Combobox, PersonPicker, TeamPicker, PersonLabel, TeamPill (+ TeamPills),
│   │                                     SelectField, TextAreaField, FilterPills, SearchField, FormPanel (moved from dev)
│   │                                   ~ PageHeader (breadcrumbs), ActionsMenu (item hint), index.ts
│   ├── features/people/                + peopleApi.ts, texts.ts, PeoplePage, PersonPanels, TeamsPage, TeamPanels,
│   │                                     OwnerImpactDialog, ProfilePage, ProfilePanels, PersonField, TeamField,
│   │                                     settingsSections.ts, index.ts (+ .module.css, .test.tsx each)
│   ├── features/dev/                   ~ SamplePanels uses FormPanel; dates through useFormatters
│   └── test/users.ts                   + testUser(overrides): a complete CurrentUser for tests
└── e2e/
    ├── fixtures.ts                     ~ preference reset incl. displayCurrency/dateFormat; createPerson(), createTeam()
    ├── people-and-teams.spec.ts        + US1–US3 journeys + axe
    └── profile.spec.ts                 + US4 journey + axe
```

## Backend design

### Stored types (`people/model.go`; fields and rules in data-model §1–§4)

```go
// Person is unchanged (ID, Name, Email, JobTitle, Active, record.Meta).

type Team struct {
    ID          string `json:"id"`          // "T-1001"
    Name        string `json:"name"`
    Description string `json:"description"`
    Colour      string `json:"colour"`      // a palette key: "blue", …
    Active      bool   `json:"active"`
    record.Meta
}

type Membership struct {
    ID       string `json:"id"`       // "TM-1001"
    TeamID   string `json:"teamId"`
    PersonID string `json:"personId"`
    record.Meta
}

type Preferences struct {
    PersonID         string `json:"personId"`
    Theme            string `json:"theme"`
    SidebarCollapsed bool   `json:"sidebarCollapsed"`
    DisplayCurrency  string `json:"displayCurrency"` // "" in 001 records = "USD"
    DateFormat       string `json:"dateFormat"`      // "" in 001 records = "DD MMM YYYY"
}

var palette = []struct{ Key, Label string }{{"blue", "Blue"}, {"violet", "Violet"}, {"teal", "Teal"},
    {"magenta", "Magenta"}, {"orange", "Orange"}, {"green", "Green"}, {"slate", "Slate"}, {"gold", "Gold"}}
var dateFormats = []string{"DD MMM YYYY", "DD/MM/YYYY", "MM/DD/YYYY", "YYYY-MM-DD"}
```

API types (separate from stored records, research R10; lists never nil; JSON names as the contract):
`PersonRef{ID, Name, Initials, JobTitle, Active}`, `TeamRef{ID, Name, Colour, Active}`, `PersonView{ID,
Name, Email, JobTitle, Initials, Active, Teams []TeamRef, record.Meta}`, `PersonDetails{PersonView; Usage
[]UsageCount}`, `UsageCount{Role, RecordType, Label, Count}`, `PersonPage{Items, Total}`, `TeamView{ID,
Name, Description, Colour, Active, Members []PersonRef, Plans []TeamPlanUse, record.Meta}`,
`TeamPlanUse{ID, Name, InRecycleBin}`, `TeamPage{Items, Total}`, `DisplayCurrencyOption{Code, Name,
Active}`, `DisplayCurrencyPage{Items, Total}`, `ChangeImpact{Warnings []ImpactWarning}` and
`ImpactWarning{Kind, Message string; Count int}` (the JSON of 002's schemas; `people` defines its own Go
types so it never imports `settings`); `CurrentUser` gains `Version int`, `Teams []TeamRef`;
`PreferencesView` gains `DisplayCurrency`, `DateFormat`. Inputs: `PersonInput{Name, Email, JobTitle
string; TeamIDs []string}`, `PersonUpdate{Name, Email, JobTitle *string; TeamIDs *[]string; Version *int}`,
`TeamInput{Name, Description, Colour string; MemberIDs []string}`, `TeamUpdate{Name, Description, Colour
*string; MemberIDs *[]string; Version *int}` (only the fields to change, as D-203 says; a field left out
keeps its value, D-210), `StatusUpdate{Active *bool; Version *int}`,
`ProfileUpdate{Name, JobTitle string; Version *int}`; `PreferencesUpdate` gains `DisplayCurrency *string`,
`DateFormat *string`. Plus `DisplayRate{Code, Rate string}` (Go only).

### Repository (`repository.go`, JSON in `store_json.go`)

001's methods stay. Added: `GetTeam(ctx, id) (Team, bool, error)`, `ListTeams(ctx) ([]Team, error)`,
`PutTeam(ctx, Team) error`, `DeleteTeam(ctx, id) error` (table `teams`); `GetMembership`,
`ListMemberships(ctx) ([]Membership, error)`, `PutMembership`, `DeleteMembership` (table
`team_memberships`); `ListPreferences(ctx) ([]Preferences, error)`. Joins (a team's members, a person's
teams) are done by the service over `ListMemberships` — quick at demo scale; PostgreSQL will add indexed
queries behind the same interface. There is **no** `DeletePerson` (FR-008).

### Constructor, hooks and wiring

```go
func NewService(tx store.Transactor, repo Repository, generator *ids.Generator,
    hist *history.Service, now clock.Clock) *Service           // currencies default to OnlyUSD

// usage.go — set in internal/app before the API serves requests (like history.Register)
type PlanUse struct { ID, Name, OwnerID string; TeamIDs []string; InRecycleBin bool }
type PlanSource func(ctx context.Context) ([]PlanUse, error)   // every plan, stored and in the bin (004)
type PersonUsage struct {
    Role             string // "owner" or "approver"
    RecordType       string // "activity", "approvalWorkflow"
    Singular, Plural string // "activity"/"activities", "approval workflow"/"approval workflows"
    Count func(ctx context.Context, personID string) (int, error) // records not in the recycle bin
}
type Currency struct { Code, Name, Rate string; Active bool }  // Rate: units per 1 USD, decimal text
type CurrencySource func(ctx context.Context) ([]Currency, error)
func OnlyUSD(ctx context.Context) ([]Currency, error)          // [{USD, "US dollar", "1", true}]

func (s *Service) UsePlans(src PlanSource)
func (s *Service) AddPersonUsage(u PersonUsage)
func (s *Service) UseCurrencies(src CurrencySource)
```

Hooks run inside the people service's transaction (the context carries it), like history's kind functions.
`app.New`: generator → people repository → history service → `people.NewService(db, peopleRepo, generator,
historyService, now)` → register `peopleService.Kinds()` with history (person, team, teamMembership; an
error stops start-up) → auth (unchanged, still on `peopleRepo`) → samples (demo mode) → routes. Feature 002
adds `peopleService.UseCurrencies(...)`, 004 `UsePlans(...)`, 005/006/008 `AddPersonUsage(...)`.

### Service operations (`people.go`, `teams.go`, `service.go`; every one takes the actor)

Every operation runs in one `Write` (or `Read`), reports **every field problem at once** (`errs.Problems`;
format, length, uniqueness, membership choices and a missing version are all collected in one pass inside
the transaction), then checks the version (`CheckVersion`, nouns "person" / "team"), then saves, touches and
logs in the same `Write`. A save that changes nothing writes nothing.

| Operation | Rules (FR) |
|---|---|
| `ListPeople(ctx, a) ([]PersonView, error)` | Everyone, active or not, with teams (TeamRef, by name); sorted by name ignoring case, then e-mail (FR-001, FR-012) |
| `GetPerson(ctx, a, id) (PersonDetails, error)` | 404 "This person doesn't exist."; `usage` = owner of plans (PlanSource, not in the bin) then each `PersonUsage` in registration order, counts ≥ 1 only, owners before approvers (FR-004, P12) |
| `CreatePerson(ctx, a, PersonInput) (PersonDetails, error)` | Trim; e-mail lower case; name 1–120; e-mail valid, ≤ 254, unique among **all** people ignoring case (message names the holder; inactive holder → "reactivate them instead") (FR-002); job title ≤ 120; each team exists and is active (P15, FR-011); id `U-`; `Active: true`; memberships `TM-`; logs (data-model §6); the person can sign in at once — 001's `auth` finds them (FR-003) |
| `UpdatePerson(ctx, a, id, PersonUpdate) (PersonDetails, error)` | As create; version required; teams already held may be inactive (kept); membership changes on both sides (P2, FR-012); a new e-mail is the next sign-in name (US1-4) |
| `PersonImpact(ctx, a, id, PersonUpdate) (ChangeImpact, error)` | 404; the same field checks as `UpdatePerson` (version not checked); warnings for the removed teams (P3, FR-013); saves nothing |
| `SetPersonStatus(ctx, a, id, StatusUpdate) (PersonDetails, error)` | `active` and `version` required (missing → field problems); 404; deactivating **yourself** (`a.ID == id`, `active: false`) → `409 not_allowed` "You can't deactivate yourself — ask someone else to do it." (FR-006); version check; already in that status → unchanged; otherwise Touch and log `statusChanged` "deactivated" / "reactivated"; memberships kept (FR-005, FR-007). Sessions of a deactivated person end at their next request (001 `auth.Authenticate`) |
| `ListTeams(ctx, a) ([]TeamView, error)` | Every team with members (PersonRef by name, inactive included) and plans using it (PlanSource, **including** the bin, by name) (FR-010, FR-014) |
| `GetTeam(ctx, a, id) (TeamView, error)` | 404 "This team no longer exists — someone may have deleted it." |
| `CreateTeam(ctx, a, TeamInput) (TeamView, error)` | Name 1–80, unique among stored teams ignoring case and spaces ("A team called '<stored name>' already exists."); description ≤ 300; colour in the palette, or empty → least-used colour, ties in palette order (P10); each member exists and is active (FR-011); id `T-`; memberships; logs both sides (FR-012, FR-028) |
| `UpdateTeam(ctx, a, id, TeamUpdate) (TeamView, error)` | As create; colour required; version required; existing inactive members may stay; a rename shows on every plan at once because plans store the team id (US2-4) |
| `TeamImpact(ctx, a, id, TeamUpdate) (ChangeImpact, error)` | 404; the same field checks as `UpdateTeam` (version not checked); warnings for the removed members, "this team" wording (P3, FR-013); saves nothing |
| `SetTeamStatus(ctx, a, id, StatusUpdate) (TeamView, error)` | `active` and `version` required; 404; version check; no change when already so; log `statusChanged` "deactivated" / "reactivated"; members and plans keep it; inactive members still count for R1 (FR-014) |
| `DeleteTeam(ctx, a, id) (history.BinEntry, error)` | 404; any plan using it, **including plans in the recycle bin** → `409 in_use` "<name> is used by 1 plan, so it can't be deleted. Set it inactive instead." (FR-014); otherwise `history.MoveToBin(ctx, a, "team", id)` — team + memberships in one entry (FR-015) |
| `CurrentUser(ctx, a) (CurrentUser, error)` | 001 + `version`, `teams` (FR-026), preferences with defaults filled (FR-023, FR-024) |
| `UpdateProfile(ctx, a, ProfileUpdate) (CurrentUser, error)` | The signed-in person (gone → `sign_in_required`); name and job title rules of FR-002 (FR-022); version required, noun "person"; logs person `changed` (US4-2); the e-mail cannot change here (FR-021) |
| `UpdatePreferences(ctx, a, PreferencesUpdate) (PreferencesView, error)` | 001 + `displayCurrency` must be an **active** currency or the stored current one (edge case, FR-023); `dateFormat` one of four (FR-024); saved per person, so every browser shows them (FR-025); not logged (P7) |
| `DisplayCurrencies(ctx, a) ([]DisplayCurrencyOption, error)` | Active currencies + the current one when inactive, by code (FR-023) |
| `ActivePeople(ctx)` | Unchanged (001, no actor): new people appear on the sign-in page (FR-003) |
| `Kinds() []history.Kind` | data-model §6 |

### Membership changes and owner impact (`memberships.go`)

- `applyMemberships(ctx, a, add, remove []pair)` (inside the caller's Write): creates `TM-` memberships for
  `add` (never a second one for a pair), deletes those of `remove`, then for **every other record** whose
  membership set changed — the teams of a person edit, the people of a team edit — reads it, `Touch`es it,
  saves it and logs one `changed` entry built with `history.Diff` over its log view (data-model §6). The
  edited record itself is saved and logged once by its operation, together with its other changes.
- `ownerImpact(ctx, removals map[personID][]teamID, after map[personID][]teamID) []ImpactWarning`: for each
  stored plan (not `InRecycleBin`) of the PlanSource whose `OwnerID` loses a membership in one of the plan's
  `TeamIDs` and whose teams `after` share none with the plan's `TeamIDs` (inactive teams count), grouped by
  owner: one warning `{kind: "ownerNeedsAttention", message, count: <plans>}` per owner, owners by name,
  plans by name (texts in *User-visible messages*). No PlanSource → no warnings.

### Lookup functions for other features (`usage.go`)

They join the caller's transaction and take **no actor** — an addition to 001's list of exceptions: they
serve another service that already holds the actor.

| Function | Returns | Used by |
|---|---|---|
| `PersonRefs(ctx, ids []string) (map[string]PersonRef, error)` | Refs of the known ids (unknown ids left out) | 004–008 views and `Describe` hooks ("Owner: Priya Raman") |
| `TeamRefs(ctx, ids []string) (map[string]TeamRef, error)` | Refs of the known ids | 004 plan teams, 009, 010 |
| `MemberIDs(ctx, teamIDs []string) (map[string][]string, error)` | Each team's member ids (active and inactive people) | 004 `plans.Directory` |
| `TeamIDsOf(ctx, personID string) ([]string, error)` | The person's team ids (active and inactive teams) | 004 default plan teams and "My teams" (FR-026) |
| `IsMemberOfAny(ctx, personID string, teamIDs []string) (bool, error)` | Whether the person belongs to one of the teams (inactive teams count) | 004 R1 check (004 FR-021) |
| `DisplayRateOf(ctx, personID string) (DisplayRate, error)` | The person's display currency and rate (fallback USD/"1") | 004–012 API conversion (P6), 010 export |
| `PeopleByDisplayCurrency(ctx) (map[string][]string, error)` | Currency code → ids of people showing amounts in it (empty stored value = USD) | 002 "Display currency of 2 people" (`settings.ReferenceSource` adapter in `internal/app`) |

### History kinds (`kinds.go`) and log entries

Kinds `person` (its `Remove` returns an error: people are never deleted, FR-008 — it exists because
`history.Register` requires it), `team`, `teamMembership` (parent `team`, `ChildIDs` = the team's
memberships by id). `Describe` and `PrepareRestore` exactly as data-model §6; the team's `PrepareRestore`
copies `samples.prepareFolderRestore` (Undo + taken name → `name_taken` "Can't undo — there is now another
team called '<name>'. Restore it from the recycle bin instead."; Restore → " (restored)", " (restored 2)", …
with the note "Renamed to '<new name>' because the name was taken."). The entries each action writes are in
data-model §6 (FR-028).

### Errors

`errs` gains `CodeNotAllowed = "not_allowed"` (409) with `NotAllowed(message)`, and uses feature 002's
`CodeInUse = "in_use"` / `InUse(message)` — if 003 is built before 002 is merged, 003 adds them with exactly
these names. Missing records answer `404 not_found` with this package's messages.

## API operations

All need a signed-in person; requests and answers exactly as [contracts/openapi.yaml](contracts/openapi.yaml).

| Method and path | Request | Answer | Errors |
|---|---|---|---|
| `GET /api/people` | — | 200 `PersonPage` | 401 |
| `POST /api/people` | `PersonCreate` | 201 `PersonDetails` | 400 validation_failed / bad_request, 401 |
| `GET /api/people/{personId}` | — | 200 `PersonDetails` | 401, 404 |
| `PATCH /api/people/{personId}` | `PersonUpdate` | 200 `PersonDetails` | 400, 401, 404, 409 version_conflict |
| `POST /api/people/{personId}/impact` | `PersonUpdate` | 200 `ChangeImpact` | 400, 401, 404 |
| `POST /api/people/{personId}/status` | `StatusUpdate` | 200 `PersonDetails` | 400, 401, 404, 409 version_conflict / not_allowed |
| `GET /api/teams` | — | 200 `TeamPage` | 401 |
| `POST /api/teams` | `TeamCreate` | 201 `Team` | 400, 401 |
| `GET /api/teams/{teamId}` | — | 200 `Team` | 401, 404 |
| `PATCH /api/teams/{teamId}` | `TeamUpdate` | 200 `Team` | 400, 401, 404, 409 version_conflict |
| `DELETE /api/teams/{teamId}` | — | 200 `DeleteResult` | 401, 404, 409 in_use |
| `POST /api/teams/{teamId}/impact` | `TeamUpdate` | 200 `ChangeImpact` | 400, 401, 404 |
| `POST /api/teams/{teamId}/status` | `StatusUpdate` | 200 `Team` | 400, 401, 404, 409 version_conflict |
| `PATCH /api/me` | `ProfileUpdate` | 200 `CurrentUser` | 400, 401, 409 version_conflict |
| `GET /api/me/display-currencies` | — | 200 `DisplayCurrencyPage` | 401 |
| changed: `GET /api/me`, `POST /api/auth/sign-in` | — | `CurrentUser` with `version` and `teams` | as 001 |
| changed: `PATCH /api/me/preferences` | `PreferencesUpdate` (+ `displayCurrency`, `dateFormat`) | `Preferences` (4 fields) | as 001 |

There is no `DELETE /api/people/{personId}`: it answers 001's JSON `404 not_found` (FR-008).

## Demo data (`backend/seed/seed.go`)

After the people: teams Lubricants Europe (T-1001, blue), Mobility China (T-1002, violet) and Global Marketing
Ops (T-1003, teal) with descriptions and the 8 memberships TM-1001 … TM-1008 of data-model §7, created
through the repositories with `actor.System` and the counters `T` and `TM`. `cmd/seed` prints "Loaded the
demo data into <folder>: 6 people, 3 teams." (002 and later features extend the list; the consolidation step
merges the wording).

## Frontend design

### Routes and navigation

| Path | Page | Tab title |
|---|---|---|
| `/settings/people/:personId?` | `PeoplePage`; with `personId` the details side panel is open | People · NorthStar |
| `/settings/teams/:teamId?` | `TeamsPage`; with `teamId` the details side panel is open | Teams · NorthStar |
| `/profile` | `ProfilePage` (replaces 001's Coming soon) | My profile · NorthStar |

- `navigation.ts`: `MY_PROFILE` becomes a plain `NavItem` and leaves `COMING_SOON_SECTIONS` (the account menu
  keeps linking to `/profile`). Settings stays in the sidebar; its `/settings` page belongs to feature 002,
  whose section cards list `PEOPLE_SETTINGS_SECTIONS` from `features/people/settingsSections.ts` (title,
  description, path, and a hook giving the count of active records). The sidebar's Settings item is current
  on `/settings/people` and `/settings/teams` (NavLink prefix match).
- One optional-segment route per list keeps the page (with its search and filter) mounted while the panel
  opens and closes. New and Edit panels are page state, not addresses.
- Breadcrumb on both pages: **Settings** (→ `/settings`) › People / Teams.

### Shared building blocks (`src/components/`, each with a test and an export in `index.ts`)

| Part | Props (essentials) | Behaviour | Why it is shared |
|---|---|---|---|
| `Combobox` | `label`, `name`, `options: {id, label, searchText, content?}[]`, `value: string[]`, `onChange(ids)`, `multiple?`, `renderChip(id) → {label, content, tip?}`, `placeholder?`, `countText(n)`, `noMatchText(query)`, `disabled?`, `disabledReason?`, `error?`, `hint?`, `marking?` | ARIA 1.2 combobox: input `role="combobox"` with `aria-expanded`, `aria-controls`, `aria-activedescendant`; listbox below (`aria-multiselectable` when multiple) showing **every** match in a scrolling list (max 320 px); case-insensitive "contains" on `searchText`; ↓/↑ move, Enter chooses (toggles in multiple mode; single mode closes), Escape closes, Backspace in an empty input removes the last chip; chips with a "Remove <label>" button; a visible count line and a polite `role="status"` region with the same count (FR-020); disabled = `aria-disabled` input with the reason shown below | Design system §5 "Choice / multi-choice"; one keyboard and screen-reader implementation for every searchable choice (002, 004–008) |
| `PersonPicker` | `label`, `name`, `people: Person[]`, `currentUserId`, `value`, `onChange`, `multiple?`, `teamIds?: string[]`, `disabledReason?`, `notMemberText?`, `error?`, `hint?`, `marking?` | On Combobox. Options: **active** people only (FR-018); with `teamIds` ("Only members of the selected teams"), only active members of those teams, each showing its qualifying team names; `teamIds=[]` → disabled with `disabledReason` (FR-019). The signed-in person first with "(you)" when they qualify, then by name, then e-mail (FR-017). Option: avatar, full name (wraps, never cut), second line job title or e-mail; when two listed people share a name, job title and e-mail. `searchText` = name + e-mail + job title. Chips: `PersonLabel`; an inactive value is kept, labelled and tipped (FR-018); with `teamIds`, a value outside those teams shows `notMemberText` (FR-019) | FR-016: every person field in NorthStar |
| `TeamPicker` | `label`, `name`, `teams: TeamRef[]`, `value`, `onChange`, `error?`, `hint?`, `marking?` | Combobox, multiple; offers active teams; chips as `TeamPill`; an inactive value is kept with "(inactive)" and its tip | Person form; 004 plan teams |
| `PersonLabel` | `person: {name, initials, active, jobTitle?}`, `you?`, `detail?`, `size?: 'sm' \| 'md'` | Avatar + full name + " (you)" + " (inactive)" (text, not colour) + optional second line | SC-003: "(inactive)" on 100 % of fields — one component wherever a person is named (004–012) |
| `TeamPill`, `TeamPills` | `team: TeamRef`, `to?` · `teams: TeamRef[]`, `linked?` | Neutral chip with the colour dot (`--team-<colour>`, decorative) + name + " (inactive)"; a link to `/settings/teams/:id` when `to`/`linked`; `TeamPills` wraps several | Coloured team pills (D-025) in people lists, profile, 004 plans, 009 |
| `SelectField` | `label`, `name`, `value`, `onChange`, `options: {value, label}[]`, `error?`, `hint?`, `marking?` | Native `<select>` with TextField's label, marking, hint and error layout | Short single choices (colour, display currency, date format; 002–008) |
| `TextAreaField` | TextField's props + `rows?` | Multi-line TextField | Team description; 002 help texts; 004 plan description |
| `FilterPills` | `label`, `options: {value, label}[]`, `value`, `onChange` | ZenReach pill toggles: a `role="group"` of buttons with `aria-pressed` | Design system §6.1 status pills on every list |
| `SearchField` | `label` (visually hidden), `placeholder`, `value`, `onChange` | `type="search"` with a search icon and a **Clear search** button | Design system §6.1 list search |
| `FormPanel` (+ `FailureMessage`, `problemOf`, type `SaveFailure`) | as today in `features/dev/SamplePanels.tsx`, plus `saveLabel?` | Moved unchanged from the dev page; SamplePanels imports it | Every side-panel form (001 dev, 002, 003, 005) |
| `PageHeader` ~ | `breadcrumbs?: {label, to}[]` | `<nav aria-label="Breadcrumb">` above the eyebrow, items separated by "›" | Settings sub-pages (002 FR-001, here) |
| `ActionsMenu` ~ | item `hint?: string` | A second, smaller line under the item label, joined to it with `aria-describedby` | FR-006 and FR-008 tips in the ⋯ menu |

`tokens.css` adds `--team-blue #2a78d6`, `--team-violet #6e56cf`, `--team-teal #12a594`, `--team-magenta
#d6409f`, `--team-orange #e5701a`, `--team-green #2f9e44`, `--team-slate #5b6b7a`, `--team-gold #c79400`
(same in dark mode; the dots are decorative, the name is the text).

### Dates and money for the person (`src/lib`)

- `format.ts`: `type DateFormat = 'DD MMM YYYY' | 'DD/MM/YYYY' | 'MM/DD/YYYY' | 'YYYY-MM-DD'`;
  `DATE_FORMATS: {value, example, label}[]` (labels "12 Mar 2026 (DD MMM YYYY)" …);
  `formatDate(date, format = 'DD MMM YYYY')`, `formatDateTime(timestamp, format = 'DD MMM YYYY')` (the
  default keeps every 001 call working); `parseDate(text, format): string | null` → `YYYY-MM-DD` (1–2 digit
  day and month, month names in any case, real calendar dates only).
- `useFormatters()` → `{ dateFormat, date(d), dateTime(ts), parseDate(text), money(value), moneyCompact(value)
  }` bound to `useSession().user` (defaults when nobody is signed in). `money({amount, currency})` formats
  exactly what the API answered (already converted by the API, P6). Every page of 003 and later features
  shows and reads dates only through this hook (FR-024); 005's date range picker uses `parseDate`.
- `errors.ts`: `conflictMessage(error, noun, dateFormat?)`, `resolvedMessage(error, name, dateFormat?)`;
  callers pass `useFormatters().dateFormat`. 001's sample page, `SamplePanels` and `useUndoableDelete` are
  switched to it.

### Data hooks (`features/people/peopleApi.ts`)

`usePeople()` (`['people']`), `usePerson(id)` (`['people', id]`), `useTeams()` (`['teams']`), `useTeam(id)`
(`['teams', id]`), `useDisplayCurrencies()` (`['display-currencies']`); mutations `useCreatePerson`,
`useUpdatePerson`, `useCheckPersonImpact`, `useSetPersonStatus`, `useCreateTeam`, `useUpdateTeam`,
`useCheckTeamImpact`, `useSetTeamStatus`, `useDeleteTeam`, `useUpdateProfile`,
`useUpdateDisplayPreferences`. Every change, on settle, invalidates `['people']`, `['teams']`, `['session']`
(teams and names on the account button), `['activity-log']` and `['recycle-bin']` — membership changes show
on both sides at once (FR-012). `useUpdateProfile` also puts its answer into the session cache at once
(US4-2). `fetchLatestPerson` / `fetchLatestTeam` (staleTime 0) serve the conflict handler, as
`fetchLatestFolder` does. Impact checks change nothing and invalidate nothing.

### Save flow of the person and team forms

1. **Save**: when the form removed teams (person) or members (team) compared with the record it started
   from, call `…/impact` with the form's body first; field problems from it are shown exactly like a save's.
2. Warnings → `OwnerImpactDialog` (on `ConfirmDialog`, shared by both forms; with the helper
   `removedIds(before, after)` for step 1) "Some plans will need a new owner" listing each warning's message,
   **Keep editing** (focused) / **Save changes**; **Save changes** sends the PATCH. No warnings, or nothing
   removed → the PATCH right away.
3. PATCH answers as in 001's edit-form rule (validation, conflict with the refreshed version, not found).

### Pages

**People** (`PeoplePage.tsx`, US1, FR-001 – FR-009)
- `PageHeader` with breadcrumbs, eyebrow "Settings", title "People", description; primary **New person**.
- Card with toolbar: `SearchField` (name, e-mail, job title; case-insensitive), `FilterPills` All · Active ·
  Inactive (default All, P11), count. Table: Name (`PersonLabel` as a button-link opening the details;
  "(you)" for the signed-in person), E-mail, Job title ("—" when empty), Teams (`TeamPills` or "No team"),
  Status (`StatusChip` Active = positive, Inactive = neutral), `ActionsMenu` (**View**, **Edit**, then
  **Deactivate** with the hint "People are never deleted, so the history keeps their name." — for yourself
  the hint "You can't deactivate yourself — ask someone else to do it.", and choosing it shows that sentence
  as a message — or **Reactivate** for inactive people). There is no Delete anywhere (FR-008). The table
  scrolls sideways inside its card on narrow screens.
- `PersonDetailsPanel` (`SidePanel`, title = name): large avatar, status chip, E-mail, Job title, Teams
  (links to `/settings/teams/:id`; a `SearchField` "Search teams" when more than 10), **Owns and approves**
  (`texts.usageText`), "Created by … on … · Last changed by … on …" (`useFormatters().dateTime`). Footer:
  **Edit** (primary), **Deactivate** / **Reactivate** (secondary). For yourself, **Deactivate** has
  `aria-disabled="true"` with the reason below it, always visible (FR-006).
- `PersonFormPanel` (`FormPanel`, "New person" / "Edit person"): Name (Required), E-mail (Required, hint),
  Job title, Teams (`TeamField`). 001's edit-form rule (copy once, version kept, a conflict replaces only the
  version) and the save flow above. Toasts with **View** (→ `/settings/people/:id`).
- `DeactivateDialog` (`ConfirmDialog`, tone danger): loads `usePerson(id)` first; while loading, a skeleton
  line stands where the usage sentence goes and **Deactivate person** clicks are ignored; texts from
  `texts.deactivateText` (FR-005). Then `POST …/status {active: false, version}`.

**Teams** (`TeamsPage.tsx`, US2, FR-010 – FR-015)
- `PageHeader` (breadcrumbs, eyebrow "Settings", title "Teams", description), primary **New team**.
- Toolbar: `SearchField` (name and description), `FilterPills` All · Active · Inactive (default All), count.
  Table: Team (`TeamPill` button-link), Description (clamped to two lines; full text in the details),
  Members (up to 5 avatars + "<n> members"), Used by ("<n> plans" / "Not used"), Status, `ActionsMenu`
  (**View**, **Edit**, **Set team inactive** / **Reactivate**, separator, **Delete** in red).
- `TeamDetailsPanel`: colour, status, Description ("—"), **Members (<n>)** (`PersonLabel` with job title;
  inactive members labelled; `SearchField` "Search members" when more than 10), **Used by** (plan links to
  `/plans/:id`; plans in the bin as plain text with "(in the recycle bin)"), the record line. Footer
  **Edit**, **Set team inactive** / **Reactivate**, **Delete**.
- `TeamFormPanel`: Name (Required), Description (`TextAreaField`), Colour (`SelectField` with the colour dot
  next to it), Members (`PersonField multiple`); save flow above.
- `DeleteTeamDialog`: when `team.plans` is not empty — title "'<name>' can't be deleted", the plan list,
  **Cancel** / **Set team inactive** (`POST …/status {active: false}`); otherwise "Delete team '<name>'?"
  with the membership count, **Cancel** / **Delete team** (danger) → `useUndoableDelete({ invalidate:
  [['teams'], ['people'], ['session']] })`. A `409 in_use` from the API (a plan appeared meanwhile)
  refreshes the team and shows the first dialog.

**Fields for other features** (`PersonField.tsx`, `TeamField.tsx`, US3) — data-bound wrappers of
`PersonPicker` / `TeamPicker`: they load `usePeople()` / `useTeams()` and `useCurrentUser()`; while loading
a field-sized `Skeleton` under the label; on failure "People couldn't be loaded." / "Teams couldn't be
loaded." with **Try again**. Props: the picker's props without the data. `features/people/index.ts`
exports `PersonField`, `TeamField`, `usePeople`, `useTeams` and the types `PersonRef`, `TeamRef`.

**My profile** (`ProfilePage.tsx`, US4, FR-021 – FR-025)
- `PageHeader` eyebrow "My profile", title "Your profile", description. No primary page action.
- Card **Your details** (header action **Edit details**): large avatar, Name, Job title, E-mail with a lock
  icon and "Your e-mail is your sign-in name; it is changed in Settings › People." → `EditDetailsPanel`
  (Name Required, Job title) → `PATCH /api/me`; conflict noun "profile".
- Card **Display preferences** (header action **Edit preferences**): Display currency ("EUR — Euro"; an
  inactive one adds " (inactive)"), Date format ("12 Mar 2026 (DD MMM YYYY)"), the explanation line →
  `EditPreferencesPanel` (`SelectField`s; currencies from `useDisplayCurrencies()`) → `PATCH
  /api/me/preferences` with both fields; afterwards every query is refetched (`invalidateQueries()`), so
  the next amounts come converted by the API and dates use the new format (SC-006).
- Card **My teams**: `TeamPills` (linked), then "Manage teams in Settings › Teams" (→ `/settings/teams`);
  none: the empty sentence (US4-5).

### States

| Screen | Loading | Empty | Error | Validation | Conflict | Not found |
|---|---|---|---|---|---|---|
| People / Teams list | header and toolbar shown, `Skeleton` rows in the card | EmptyState with **New person** / **New team**; filters hiding all: "Nothing matches these filters" + **Clear filters** | `ErrorState` in the card, **Try again** | — | — | — |
| Details panel | `Skeleton` in the panel | — | `ErrorState` in the panel | — | — | the not-found message in the panel + **Close** |
| Form panel | Save ignored while saving (`aria-busy`) | — | `FailureMessage` with the API message (unreachable: 001's sentence) | `FormErrorSummary` + messages under fields, all at once (also from the impact check) | `conflictMessage(error, 'person' \| 'team' \| 'profile')` + 001's next step; typed values kept | `FailureMessage` with the not-found message |
| Status actions (⋯ / footer) | busy button | — | message with the API message | — | message with `conflictMessage`; lists refreshed | message with the not-found message |
| Pickers | field skeleton | no-match sentence | "… couldn't be loaded." + **Try again** | field error from the form | — | unknown ids are not shown as chips |
| My profile | from the session (no wait); currency choices: skeleton in the panel | My teams empty sentence | `ErrorState` in the panel | as forms | as forms | — |

Unsaved changes: every form panel uses 001's `useUnsavedChanges` through `FormPanel`.

## User-visible messages

Every text appears once; code and tests quote this table. `<…>` is filled in at run time.

| Where | Text |
|---|---|
| Breadcrumb | Settings › People · Settings › Teams |
| People header | Settings · People · Everyone who can sign in to NorthStar, with their job title and teams. · **New person** |
| Teams header | Settings · Teams · Groups of people that plans are assigned to. A plan's owner must belong to one of its teams. · **New team** |
| Settings cards (002 page) | People — Everyone who can sign in, with their job title and teams. · Teams — Groups of people that plans are assigned to. |
| Search | People: "Search name, e-mail or job title" · Teams: "Search team name or description" · Search members · Search teams · **Clear search** |
| Filters / counts | All · Active · Inactive · "<n> people" / "1 person" / "<n> of <total> people" · "<n> teams" / "1 team" / "<n> of <total> teams" |
| Table headings | Name · E-mail · Job title · Teams · Status · Team · Description · Members · Used by · Actions (visually hidden) |
| Cells | No team · "<n> members" / "1 member" / No members · "<n> plans" / "1 plan" / Not used · — |
| Status chips | Active · Inactive |
| Suffixes | " (inactive)" · " (you)" |
| Empty lists | No people yet — add the first person · No teams yet — create your first team · Nothing matches these filters · **Clear filters** |
| Row and panel actions | View · Edit · Deactivate · Reactivate · Set team inactive · Delete · **Edit details** · **Edit preferences** |
| Deactivate tip (⋯ and panel) | People are never deleted, so the history keeps their name. |
| Self-deactivation (web and API) | You can't deactivate yourself — ask someone else to do it. |
| Person details | E-mail · Job title · Teams · Owns and approves · Not in any team yet. A person needs a team to own a plan. · Doesn't own or approve anything yet. · "Owner of <owner list> · approver in <approver list>" (one part alone when the other is empty; list: "2 plans and 5 activities", "2 plans, 5 activities and 1 initiative") |
| Record line | Created by <name> on <date, time> · Last changed by <name> on <date, time> |
| Person form | New person · Edit person · Name · E-mail · Job title · Teams · E-mail hint (new): Their sign-in name. They can sign in straight away with the demo password. · E-mail hint (edit): Their sign-in name — the next sign-in uses the new e-mail. · Teams hint: Only active teams can be added. · **Save** · **Cancel** |
| Person toasts | Person '<name>' created · **View** · Person '<name>' saved · Person '<name>' deactivated · Person '<name>' reactivated |
| Deactivate dialog | Deactivate <name>? · <First> won't be able to sign in or be chosen in people lists. · <First> owns <owner list> and approves in <approver list>; these keep the name, shown as '<name> (inactive)'. (either part alone when the other is empty; the sentence is left out when both are) · Their teams are kept. · **Cancel** · **Deactivate person** |
| Team details | Description · Colour · Members (<n>) · Used by · "Used by 1 plan" / "Used by <n> plans" · (in the recycle bin) · No members yet. Add people so they can own this team's plans. · Not used by any plan yet. |
| Team form | New team · Edit team · Name · Description · Description hint: Up to 300 characters. · Colour · Colour hint: Shown as a dot next to the team's name. · Blue · Violet · Teal · Magenta · Orange · Green · Slate · Gold · Members · Members hint: Only active people can be added. |
| Team toasts | Team '<name>' created · **View** · Team '<name>' saved · Team '<name>' set inactive · Team '<name>' reactivated · (delete and Undo: 001's "Team '<name>' moved to the recycle bin" · **Undo** · "Team '<name>' restored") |
| Delete team (unused) | Delete team '<name>'? · It goes to the recycle bin together with its <n> team memberships. You can restore it. (1: "its team membership"; 0: "It goes to the recycle bin. You can restore it.") · **Cancel** · **Delete team** |
| Delete team (used) | '<name>' can't be deleted · It is used by <n> plans: <plan>, <plan> (in the recycle bin). Set it inactive instead: it won't be offered for new plans, and these plans keep it, shown as '<name> (inactive)'. (1: "It is used by 1 plan: …") · **Cancel** · **Set team inactive** |
| Owner confirmation dialog | Some plans will need a new owner · <each warning's message> · **Keep editing** · **Save changes** |
| Owner warning (API, team impact) | <Person> owns 1 plan that uses this team: <plan>. <First> will no longer belong to its teams, so its owner will need attention. · <Person> owns <n> plans that use this team: <plan>, <plan>. <First> will no longer belong to their teams, so their owner will need attention. |
| Owner warning (API, person impact) | as above with "uses <team name>" / "use <team name>" when one team is removed, "uses the removed teams" / "use the removed teams" otherwise |
| Picker | Search name, e-mail or job title · "<n> people" / "1 person" · No one matches '<query>'. Check the spelling, or add the person in Settings › People. · No members of the selected teams match. Add people to these teams in Settings › Teams. · Choose teams first — the owner must belong to one of the plan's teams. (default `disabledReason`) · Not a member of the plan's teams (default `notMemberText`) · Inactive people can't be chosen again. Keep them or choose someone else. · Remove <name> · People couldn't be loaded. · **Try again** |
| Team picker | Search teams · "<n> teams" / "1 team" · No team matches '<query>'. · Inactive teams can't be chosen again. Keep it or remove it. · Remove <name> · Teams couldn't be loaded. |
| My profile | My profile · Your profile · Your details and teams, and how NorthStar shows dates and money to you. · Your details · Display preferences · My teams · Your e-mail is your sign-in name; it is changed in Settings › People. · Amounts are shown in your display currency with today's rate in Settings; amounts typed in other currencies stay as they are. · Manage teams in Settings › Teams · You're not in any team yet, so you can't own a plan. Teams are managed in Settings › Teams. |
| Profile panels and toasts | Edit your details · Edit display preferences · Display currency · Date format · "<code> — <name>" · 12 Mar 2026 (DD MMM YYYY) · 12/03/2026 (DD/MM/YYYY) · 03/12/2026 (MM/DD/YYYY) · 2026-03-12 (YYYY-MM-DD) · Your details saved · Your preferences saved |
| Validation (API) | Enter a name. · Use 120 characters or fewer. · Use 80 characters or fewer. · Use 300 characters or fewer. · Use 254 characters or fewer. · Enter an e-mail address. · Enter a valid e-mail address. · <e-mail> already belongs to <name>. · <e-mail> already belongs to <name> (inactive) — reactivate them instead. · A team called '<name>' already exists. · Choose one of the colours. · <team> is inactive, so people can't be added to it. Reactivate the team first. · Choose teams from the list. · <name> is inactive and can't be added. Reactivate them first. · Choose people from the list. · Choose one of the active currencies. · Choose one of the date formats. · Choose active or inactive. (status `active` missing) · Reload the page and try again. (version) |
| Refusals (API) | This person doesn't exist. · This team no longer exists — someone may have deleted it. · <name> is used by 1 plan, so it can't be deleted. Set it inactive instead. (<n> plans) · You can't deactivate yourself — ask someone else to do it. · Not saved — <person> changed this person / team (001 format) · Can't undo — there is now another team called '<name>'. Restore it from the recycle bin instead. · Renamed to '<new name>' because the name was taken. |
| Seed | Loaded the demo data into <folder>: 6 people, 3 teams. |

## Tests

### Go — service rules (`package people`, fixed clock 2026-03-12 09:00 UTC, temporary store, fake hooks)

| File | Tests |
|---|---|
| `model_test.go` | `TestPerson_FirstNameAndInitials_FR001` (001, kept) + `TestPerson_InitialsOneWordName_FR009` ("Cher" → "CH") |
| `people_test.go` | `TestListPeople_EveryoneSortedWithTeams_FR001` · `TestCreatePerson_SavedActiveAndLogged_FR001_FR028` · `TestCreatePerson_AllProblemsAtOnce_FR002_SC007` (empty name + invalid e-mail; 121-char name, 255-char e-mail, 121-char title) · `TestCreatePerson_EmailTrimmedAndLowerCase_FR002` · `TestCreatePerson_DuplicateEmailNamesHolder_FR002` (active; inactive with "reactivate them instead"; other letter case) · `TestCreatePerson_WithTeamsShowsOnBothSides_FR012` · `TestCreatePerson_InactiveOrUnknownTeamRefused_FR011` · `TestUpdatePerson_ChangesLoggedOldAndNew_FR028` · `TestUpdatePerson_MissingVersion_FR002` · `TestUpdatePerson_VersionConflict_FR029` · `TestUpdatePerson_NoChangeNoVersionNoLog_FR028` · `TestUpdatePerson_KeepsInactiveTeam_FR014` · `TestSetPersonStatus_DeactivateKeepsMembershipsAndLogs_FR005_FR028` · `TestSetPersonStatus_Yourself_NotAllowed_FR006` · `TestSetPersonStatus_AlreadyInactiveUnchanged_FR005` · `TestSetPersonStatus_ReactivateMembershipsIntact_FR007` · `TestSetPersonStatus_MissingActiveAndVersion` · `TestGetPerson_UsageCountsFromHooks_FR004` (plan owner from PlanSource without bin plans; activity and approval-workflow PersonUsage; zero counts left out; singular/plural labels) · `TestGetPerson_NotFound_FR008` |
| `teams_test.go` | `TestCreateTeam_SavedWithMembersAndLogged_FR010_FR028` · `TestCreateTeam_AllProblemsAtOnce_FR011` (empty, 81 chars, duplicate ignoring case and spaces, 301-char description, bad colour, inactive and unknown member) · `TestCreateTeam_ColourLeastUsed_D025` · `TestUpdateTeam_RenameUnique_FR011` · `TestUpdateTeam_KeepsInactiveMember_FR011` · `TestUpdateTeam_VersionConflict_FR029` · `TestTeamEdit_MembershipShowsOnPerson_FR012` · `TestPersonEdit_MembershipShowsOnTeam_FR012` · `TestSetTeamStatus_DeactivateKeepsMembersAndLogs_FR014` · `TestSetTeamStatus_Reactivate_FR014` · `TestDeleteTeam_UsedByPlanRefused_FR014` (a stored plan; a plan only in the bin) · `TestDeleteTeam_UnusedMovesWithMemberships_FR015` · `TestListTeams_MembersAndPlans_FR010` |
| `memberships_test.go` | `TestMembershipChange_BothSidesNewVersionAndLog_FR012_FR028` · `TestTeamImpact_OwnerLosesLastPlanTeam_Warns_FR013` (exact message, count, nothing saved) · `TestTeamImpact_OwnerStillInAnotherPlanTeam_NoWarning_FR013` · `TestTeamImpact_InactiveTeamStillCounts_FR014` · `TestPersonImpact_OneAndTwoRemovedTeams_FR013` · `TestImpact_IgnoresPlansInRecycleBin_FR013` · `TestImpact_ValidatesLikeUpdate_SC007` · `TestUpdateTeam_RemovingOwnerIsAllowed_FR013` |
| `kinds_test.go` | `TestKinds_Register_FR028` · `TestPersonKind_RemoveRefused_FR008` · `TestTeamDelete_UndoRestoresMembershipsExactly_FR015` · `TestTeamUndo_NameTakenRefused_FR015` · `TestTeamRestore_NameTakenGetsSuffix_FR015` · `TestDescribe_TeamsMembersStatusColour_FR028` |
| `service_test.go` (001's, extended) | `TestPreferences_Defaults_FR011_FR014` updated (USD, DD MMM YYYY) · `TestPreferences_Old001RecordGetsDefaults_FR024` · `TestUpdatePreferences_DisplayCurrencyActiveOnly_FR023` (inactive current kept; inactive other refused) · `TestUpdatePreferences_DateFormatChoices_FR024` · `TestUpdatePreferences_NotLogged_D038` · `TestCurrentUser_VersionAndTeams_FR026` · `TestUpdateProfile_NameAndTitleLogged_FR022` · `TestUpdateProfile_EmptyName_FR022` · `TestDisplayCurrencies_ActivePlusCurrent_FR023` |
| `usage_test.go` | `TestPersonRefs_UnknownIdsLeftOut` · `TestTeamRefs` · `TestMemberIDs` · `TestIsMemberOfAny_InactiveTeamCounts_FR014` · `TestTeamIDsOf_FR026` · `TestDisplayRateOf_EURAndFallback_FR023` (EUR "0.92"; unknown code → USD "1"; inactive EUR kept) · `TestPeopleByDisplayCurrency_EmptyIsUSD` · `TestOnlyUSD` |

### Go — handler and contract tests (`package people_test`, `apitest`, every call validated)

- `handler_test.go` (001's, updated bodies): `TestGetMe_SignedInPerson_FR012` gains `version`, `teams` and the
  two preferences; `TestPatchPreferences_*` answer four fields; `TestPatchMe_UpdatesProfile_FR021_FR022`;
  `TestPatchMe_VersionConflict_FR022`; `TestPatchPreferences_DisplayCurrencyAndDateFormat_FR023_FR024`;
  `TestGetDisplayCurrencies_FR023`; `TestPreferences_SameInAnotherBrowser_FR025` (two sign-ins).
- `people_handler_test.go`: one test per people operation (200/201, 400 with every field problem, 404, 409)
  plus `TestListPeople_GivesPickerData_FR017_FR018`, `TestNewPersonCanSignIn_FR003`,
  `TestDeactivatedPersonCannotSignInAndSessionEnds_FR005`, `TestReactivatedPersonCanSignIn_FR007_SC004`,
  `TestDeactivateYourself_409NotAllowed_FR006`, `TestDeleteNotOffered_FR008` (`DELETE /api/people/U-1002` →
  JSON 404), `TestAnyoneManagesPeople_FR027` (Daniel edits a person Alex created),
  `TestPersonImpact_NoPlansNoWarnings_FR013`.
- `teams_handler_test.go`: one test per team operation; `TestDeleteTeam_ThenUndo_FR015` (through
  `POST /api/recycle-bin/{entryId}/undo`); `TestRestoreTeam_NameTaken_FR015`;
  `TestTeamImpact_NoPlansNoWarnings_FR013`. The `409 in_use` answer and warnings with content need plans: in
  003 the service tests prove them; feature 004 adds `TestDeleteTeam_InUse_FR014` and
  `TestTeamImpact_OwnerWarning_FR013` handler tests once real plans exist.
- `auth/handler_test.go`: sign-in answer bodies updated to the new `CurrentUser`.
- `app/app_test.go`: `TestRoutesMatchContract` and `TestEveryProtectedOperationRequiresSignIn_FR005` cover
  the new operations unchanged.
- `seed/seed_test.go`: `TestLoad_TeamsAndMemberships_FR029` (names, ids, colours, members; next ids
  `T-1004`, `TM-1009`); `cmd/seed/main_test.go`: the new summary line.

### Web — unit and component tests (Vitest, UTC, fixed time 2026-03-12T09:00:00Z)

| File | Proves |
|---|---|
| `lib/format.test.ts` | each date format; `formatDateTime` with a format; `parseDate` per format, any letter case, 1-digit parts, 31/02/2026 → null (FR-024) |
| `lib/useFormatters.test.tsx` | dates follow the preference; defaults when signed out; `money` formats the API's amount and currency unchanged (FR-024, SC-006, P6) |
| `api/errors.test.ts` | new kinds; `conflictMessage` with a date format |
| `components/*/*.test.tsx` | Combobox keyboard (↓ ↑ Enter Escape Backspace), count and live region, all matches rendered, disabled reason; PersonPicker: "finance" → Sara Novak (job title), "okoro@" → Daniel Okoro (e-mail), inactive people never offered and labelled when held, "(you)" first, team mode (qualifying team, disabled without teams, no-match text, not-member flag), same-name people show e-mail, 300 people → "300 people" (FR-016 – FR-020, SC-003, SC-005); TeamPicker; PersonLabel "(inactive)"; TeamPill/TeamPills; SelectField; TextAreaField; FilterPills; SearchField; FormPanel; PageHeader breadcrumbs; ActionsMenu hint |
| `features/people/texts.test.ts` | usage line, deactivate sentences, delete-team sentences, counts (singular/plural) |
| `features/people/PeoplePage.test.tsx` | list, search on three fields, filters, count, empty and no-match states, error + Try again; create with all problems at once, duplicate e-mail message, toast with View (US1-1, US1-2); details (US1-3); edit + conflict (US1-4); deactivate dialog with usage and Cancel focused (US1-5); self-deactivation reason (US1-6); reactivate (US1-7); no Delete, tip present (US1-8); removing a team calls the impact check and shows its warning |
| `features/people/TeamsPage.test.tsx` | create with two members (US2-1); details with members and plans (US2-3); duplicate name (US2-4); removing a member → impact warning → **Save changes** sends the PATCH, **Keep editing** sends nothing (US2-5); used team → can't-delete dialog → **Set team inactive** (US2-6); unused delete → Undo (US2-7) |
| `features/people/PersonField.test.tsx`, `TeamField.test.tsx` | loading, error + Try again, data passed to the pickers |
| `features/people/ProfilePage.test.tsx` | details with the lock text (US4-1); edit name → session updated at once, empty name error (US4-2); preferences panel saves both and refetches (US4-3, US4-4); no-teams sentence (US4-5) |
| `app/router.test.tsx`, `AccountMenu.test.tsx` | `/profile` shows My profile, not Coming soon; `/settings/people` and `/settings/teams` render |

### End-to-end (Playwright, against the seeded API; tests create their own people and teams)

| File | Journey |
|---|---|
| `people-and-teams.spec.ts` | **US1** as Alex: New person with all problems at once, then valid with team Lubricants Europe → toast → listed Active; a second browser context signs in as the new person (demo password); Alex deactivates them (dialog text, Cancel focused) → Inactive; the other context's next navigation lands on sign-in; sign-in refused; Reactivate → sign-in works (FR-001 – FR-008, SC-004). **US2**: New team with two members picked with the picker → toast; add a third from that person's form → the team shows them; delete the unused team → Undo brings it back with its members (FR-010 – FR-015). **US3**: in the member field type "finance" → Sara Novak; "okoro@" → Daniel Okoro; choose with the keyboard; Escape closes (FR-016 – FR-020). axe (WCAG 2.2 AA) on both lists, both details panels, the person form with errors, the deactivate and delete dialogs and the open picker; no horizontal page scroll at 390 px (SC-008) |
| `profile.spec.ts` | **US4** with a person created for the test: My profile shows details and teams; change the date format to DD/MM/YYYY → the People page's record line shows `dd/mm/yyyy`; change the name → the account button shows it at once; a second browser context signs in as the same person and sees the same preferences (FR-021 – FR-025, SC-006). axe on the page and both panels. Amounts in EUR are proven by 004's tests once 002 provides currencies |

## Requirement traceability

| FR | Implemented in | Tested by |
|---|---|---|
| FR-001 | `ListPeople`, `PeoplePage` | `TestListPeople_…`, `PeoplePage.test.tsx`, e2e US1 |
| FR-002 | `CreatePerson` / `UpdatePerson` checks | `TestCreatePerson_AllProblemsAtOnce…`, `…DuplicateEmail…`, `…EmailTrimmed…`, page test |
| FR-003 | 001 `auth` over the shared repository | `TestNewPersonCanSignIn_FR003`, e2e US1 |
| FR-004 | `GetPerson` usage, hooks | `TestGetPerson_UsageCountsFromHooks_FR004`, page test |
| FR-005 – FR-007 | `SetPersonStatus`, 001 `Authenticate`, `PersonLabel`, pickers | people tests, handler tests, picker tests, e2e US1 |
| FR-008 | no delete route or repository method; kind `Remove` refuses; menu tip | `TestDeleteNotOffered_FR008`, `TestPersonKind_RemoveRefused_FR008`, page test |
| FR-009 | `Person.Initials`, `Avatar` | `model_test.go` |
| FR-010, FR-011 | `ListTeams`, `CreateTeam`, `UpdateTeam`, `TeamsPage` | teams tests, page test, e2e US2 |
| FR-012 | memberships table, `applyMemberships`, query invalidation | membership and teams tests, page tests, e2e US2 |
| FR-013 | `ownerImpact`, `…/impact`, the save flow's dialog | membership tests, impact handler tests, page tests (004 adds a handler test with plans) |
| FR-014 | `DeleteTeam` guard, `SetTeamStatus`, `IsMemberOfAny`, `TeamPill` | teams and usage tests, page test |
| FR-015 | `history.MoveToBin`, team kinds, `useUndoableDelete` | kinds tests, `TestDeleteTeam_ThenUndo_FR015`, page test, e2e US2 |
| FR-016 – FR-020 | `Combobox`, `PersonPicker`, `TeamPicker`, `PersonField`, `TeamField` | component tests, `TestListPeople_GivesPickerData…`, e2e US3 |
| FR-021, FR-022 | `UpdateProfile`, `PATCH /api/me`, `ProfilePage` | profile tests, e2e US4 |
| FR-023 | preference, `DisplayCurrencies`, `DisplayRateOf` (conversion by 004–012) | preference tests, `TestDisplayRateOf_…`, `TestGetDisplayCurrencies_FR023` |
| FR-024 | preference, `format.ts`, `useFormatters` | `format.test.ts`, `useFormatters.test.tsx`, e2e US4 |
| FR-025 | preferences stored per person | `TestPreferences_SameInAnotherBrowser_FR025`, e2e US4 |
| FR-026 | `CurrentUser.teams`, `TeamIDsOf` | `TestCurrentUser_VersionAndTeams_FR026`, `TestTeamIDsOf_FR026` |
| FR-027 | no permission checks | `TestAnyoneManagesPeople_FR027` |
| FR-028 | log entries of data-model §6 | `…Logged…` tests, `TestDescribe_…`, kinds tests |
| FR-029 | `seed.go` | `TestLoad_TeamsAndMemberships_FR029` |

## Dependencies

### Uses from feature 001 (by name)

Go: `store.Transactor`, `ids.Generator` (`Next`), `record.Meta` (`NewMeta`, `Touch`, `CheckVersion`),
`errs` (`Problems`, `Validation`, `NotFound`, `SignInRequired`, `NameTaken`), `history.Service`
(`Register`, `Log`, `Diff`, `MoveToBin`), `history.Kind`, `history.RestoreContext`, `history.DeleteResult`,
`BinEntry.View()`, `web.Router` / `ReadJSON` / `WriteJSON` / `WriteError`, `actor`, `clock`, `apitest`
(`New`, `SignIn`, `Do`, `Advance`), `auth.Authenticate` behaviour for inactive people,
`people.ActivePeople`, `seed.Load`. Contract: responses `BadRequest`, `Unauthorized`, `NotFound`,
`Conflict`, `Error`; schemas `ActorRef`, `DeleteResult`. Web: `Avatar`, `Button`, `Card`, `ConfirmDialog`,
`EmptyState`, `ErrorState`, `FormErrorSummary`, `InfoTip`, `PageHeader`, `SidePanel`, `Skeleton`,
`StatusChip`, `TextField`, `Tooltip`, `useToast`, `ActionsMenu`, `useUndoableDelete`, `useUnsavedChanges`,
`usePageTitle`, `conflictMessage`, `call` / `api`, `useSession` / `useCurrentUser`, `SESSION_KEY`, `mockApi`,
`renderWithProviders` / `renderRoutes`, e2e `signInAs`, `uniqueName`, `expectNoAxeViolations`.

### Expects from other features

| From | What (by name) | Until then |
|---|---|---|
| 002 (contract) | Schemas **`StatusUpdate`** `{active, version}`, **`ChangeImpact`** `{warnings}`, **`ImpactWarning`** `{kind, message, count}` with `kind` gaining `ownerNeedsAttention`; error code **`in_use`** | 003 adds them with exactly these names and shapes if it is merged first |
| 002 (Go) | `errs.InUse` / `CodeInUse`; a currency list for `peopleService.UseCurrencies(...)` through an adapter in `internal/app` from 002's service to `people.Currency{Code, Name, Rate, Active}` (rate as 002's `money.Rate` text); a `settings.ReferenceSource` adapter that reports each person's display currency from `people.(*Service).PeopleByDisplayCurrency`; `money.FromUSD` for 004–012's conversion with `DisplayRateOf` | `people.OnlyUSD`; no "Display currency of" usage |
| 002 (web) | The Settings frame `features/settings/SettingsPage.tsx` lists `PEOPLE_SETTINGS_SECTIONS` beside its own cards. If 002 also builds `SelectField`, `TextAreaField`, `FilterPills`, `SearchField`, `Combobox`, `FormPanel` or `PageHeader` breadcrumbs, consolidation keeps one of each | People and Teams are reached by address and from the sidebar's Settings item |
| 004 | `peopleService.UsePlans(...)`: an adapter in `internal/app` over `plans.Service.References` (stored plans) and `ReferencesFromSnapshot` (plans in the recycle bin) giving `PlanUse{ID, Name, OwnerID, TeamIDs, InRecycleBin}` — **004's `References` must include the plan name**; the actor passed is `actor.System` | no plans: nothing used, no warnings |
| 005, 006 | `peopleService.AddPersonUsage(...)` for owners of campaigns, initiatives, activities | no counts |
| 008 | `peopleService.AddPersonUsage(...)` with role `approver`, record type `approvalWorkflow` | no counts |

### Provides to later features

- Contract schemas **`PersonRef`** `{id, name, initials, jobTitle, active}` and **`TeamRef`** `{id, name,
  colour, active}` (supersets of what 004 expects), `TeamColour`, `DateFormat`; `CurrentUser.version`,
  `CurrentUser.teams`, preferences `displayCurrency` and `dateFormat`; `GET /api/teams` with members.
- Go: `people.PersonRef`, `people.TeamRef`, `PersonRefs`, `TeamRefs`, `MemberIDs`, `TeamIDsOf`,
  `IsMemberOfAny`, `DisplayRateOf`, `PeopleByDisplayCurrency` (together they serve 004's `plans.Directory`
  adapter), the hook types `PlanUse` / `PlanSource`, `PersonUsage`, `Currency` / `CurrencySource`, and
  `UsePlans`, `AddPersonUsage`, `UseCurrencies`.
- Web: `PersonPicker` (props `teamIds`, `disabledReason`, `notMemberText` for 004's owner field),
  `TeamPicker`, `Combobox`, `PersonLabel`, `TeamPill`, `TeamPills`, `SelectField`, `TextAreaField`,
  `FilterPills`, `SearchField`, `FormPanel`, `PageHeader` breadcrumbs, `ActionsMenu` hints (from
  `@/components`); `PersonField`, `TeamField`, `usePeople`, `useTeams` (from `@/features/people`);
  `useFormatters()` — every later page shows and reads dates through it (005's date range picker uses
  `parseDate`); `testUser()` for tests.
- The R1 flag in team-restricted mode (`teamIds`, `notMemberText`): what the plan form does with it is
  feature 004's (FR-019).

## Changes to feature 001 files

| File | Change |
|---|---|
| `backend/internal/people/*` | Extended as above; `NewService` takes `ids`, `history`, `clock`; `Preferences` and `CurrentUser` gain fields |
| `backend/internal/platform/errs/errs.go` | `not_allowed` (and `in_use` if 002 has not added it) |
| `backend/internal/app/app.go` | Wiring order (history before people); people kinds registered |
| `backend/internal/people/handler_test.go`, `service_test.go`, `backend/internal/auth/handler_test.go` | Expected bodies and `newTestService` updated |
| `backend/seed/seed.go`, `seed_test.go`, `cmd/seed/main.go`, `main_test.go` | Teams, memberships, summary line |
| `frontend/src/app/navigation.ts`, `router.tsx` | My profile built; new routes |
| `frontend/src/lib/format.ts`, `api/errors.ts`, `lib/useUndoableDelete.ts`, `features/dev/*` | Date format parameter; the dev page through `useFormatters`; `FormPanel` moved to components |
| `frontend/src/components/PageHeader`, `ActionsMenu`, `index.ts`, `styles/tokens.css` | Breadcrumbs, item hint, exports, team colours |
| The 16 web tests that build a `CurrentUser` literal (listed in tasks T022) | Use `testUser()` from `src/test/users.ts` |
| `frontend/e2e/fixtures.ts` | Preference reset with all four preferences; `createPerson()`, `createTeam()` |

## Complexity Tracking

| Choice | Why needed | Simpler alternative rejected because |
|---|---|---|
| Usage and currency hooks (P5) | 003 is built before 004–008 and must show their counts and guard team deletes; importing them would create cycles | Duplicating owner lookups in 003, or making 004 keep a usage table, couples the features more |
| A membership change touches both records (P2) | Optimistic concurrency and truthful history on both sides of FR-012 | Logging one side only leaves the other record's History incomplete and lets a stale form silently undo a membership |
| Two impact endpoints (P3) | FR-013's warning must list the right plans before saving; only the server knows plans, owners and memberships consistently | Computing R1 in the browser duplicates a business rule (constitution IV) |
| Ten new shared web parts and two extensions | Each is reused by 002 and 004–008 (reasons in the table above); the picker must be one component (FR-016) | Building them inside the people pages forces later features to copy them |
| Team colour although the spec's FR list omits it (P10) | D-025 and domain model §6 already decided coloured team pills | Leaving it out makes 004 and 009 add a field to a record they don't own |

## Consolidation (2026-10-19, binding)

The Plan-module consolidation (the cross-feature analyze step) settled the points below. They and the build guide,
[docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md), win where this plan, its data model or its tasks
say otherwise (D-200).

- **Contract (D-192).** The fragment is merged into `backend/api/openapi.yaml`; the merge task (T001) is done. The
  feature's operations are listed in `notYetServed` in `backend/internal/app/app_test.go`; the backend removes each one in
  the change that serves it. Agents do not change the contract; a problem with it is reported.
- **Decisions.** This plan's proposed decisions are numbered D-056–D-071 in `docs/sdd/decision-log.md` (labels kept in the
  table above); the consolidation decisions are D-189–D-207.

- **Shared parts (D-197).** 003 builds `Combobox` (also used by 002's `ChoiceField`), `PersonPicker`, `TeamPicker`, `PersonLabel`,
  `TeamPill`/`TeamPills` and `FormPanel`, and the `ActionsMenu` `hint`. It does **not** build `SelectField`, `FilterPills`,
  `SearchField`, `TextAreaField` or `PageHeader` breadcrumbs: use 002's `ChoiceField`, `SegmentedFilter`, `SearchField`,
  `TextAreaField` (with `rows?`) and `Breadcrumb`. Tasks that create them reuse 002's instead.
- **001 schemas extended (D-191).** In the merged contract `CurrentUser.version`, `CurrentUser.teams`,
  `Preferences.displayCurrency` and `Preferences.dateFormat` are optional; 003's API always sends them and the web app falls back
  to USD and DD MMM YYYY when they are absent. `PATCH /api/me` sits beside 001's `GET /api/me`.
- **Error codes.** `in_use` is 002's (`errs.InUse`, defined once); `not_allowed` is 003's.
- **Display currency (D-063, D-201).** An inactive display currency stays in use on every page, Home included (009 follows).
- **Team colour (D-065, D-201).** `TeamRef.colour` is the only team colour of the module; 009's derived colour is withdrawn and
  `TeamPill` is the one pill used by 004, 009 and 010.
- **Plan usage.** 004's `References` include the plan name and whether it is in the recycle bin, so `PlanUse{ID, Name, OwnerID,
  TeamIDs, InRecycleBin}` can be filled by the `internal/app` adapter.
