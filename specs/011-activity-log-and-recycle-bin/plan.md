# Implementation Plan: Activity Log and Recycle Bin

**Branch**: `011-activity-log-and-recycle-bin` | **Date**: 2026-10-19 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/011-activity-log-and-recycle-bin/spec.md`

**Design artifacts**: [data-model.md](data-model.md) · [contracts/openapi.yaml](contracts/openapi.yaml) (fragment) ·
[tasks.md](tasks.md) · builds on [001 plan](../001-app-shell-and-sign-in/plan.md), [001 data model §5–§7](../001-app-shell-and-sign-in/data-model.md)
and [ADR-0007](../../docs/architecture/adr/0007-activity-log-and-recycle-bin-core.md)

## Summary

Feature 001 built the mechanism: `internal/history` writes one log entry per change, moves deletes to
the recycle bin with everything underneath, and offers Undo and Restore. This feature adds what people
see and the remaining bin behaviour, **by extending `internal/history`** (no new Go package):

- **Activity log page** (`/activity-log`): four summary cards, filters (person, record type, action,
  plan, record with items inside, date range, search) applied on the server to the whole log, entries
  grouped by local day, expandable details, links that know whether a record still exists, is in the
  bin or was deleted permanently, 50 entries at a time with **Load more**.
- **History section** on the plan, campaign, initiative, activity and line item pages (the pages of
  004–007), with **Include items inside** / **Include line items** remembered per person and **Open in
  activity log**.
- **Recycle bin page** (`/recycle-bin`): one row per entry with path, contents, who and when, restore
  state (restorable · parent also in the bin · parent deleted permanently), search that also searches
  the contents, type pills, **Restore** with a **preview** that states destination, contents, every
  status change and every name change, **Restore both / Restore all N**, **Delete permanently**,
  **Empty recycle bin**.

The restore preview is a **dry run of the real restore** inside a write transaction that is rolled back,
so the dialog can never promise something the restore does not do. Log filters by plan or by record
"with the items inside" use the path stored in each entry (data model §5 of 001) plus the records moved
in later, so no current tree has to be walked.

## Technical Context

**Language/Version**: Go 1.27 (API); TypeScript 5.9 strict + React 19.3 (web app) — as feature 001

**Primary Dependencies**: none new. API: standard library. Web app: `react-router` 8, `@tanstack/react-query` 5
(`useInfiniteQuery` for **Load more**), `radix-ui` (DropdownMenu with checkbox items for the Action filter),
`lucide-react`, `openapi-fetch` — all from 001 research R2/R12

**Storage**: the JSON store of 001. Tables used: `activity_log` (append-only, unchanged format plus one new
field `items` for new entries), `recycle_bin` (resolution `how` gains `deletedPermanently`), `preferences`
(new field `historyIncludeInside`). No new table, no new ID prefix

**Testing**: Go `testing` with fake kinds and the fixed clock; `apitest` contract tests for every
operation; Vitest + Testing Library (UTC, fake timers); Playwright + axe end to end

**Target Platform**: as 001 (developer laptops, current Chrome/Edge)

**Project Type**: web application (React SPA + Go REST API)

**Performance Goals**: SC-006 — with 50,000 log entries the first page and every filter change answer in
under 2 seconds (the server keeps tables in memory; one query is one pass over the entries)

**Constraints**: 127.0.0.1 only; entries are never changed or removed (001 FR-023); nothing leaves the bin
automatically (FR-031); every restore, permanent delete and empty is logged with the actor (SC-007);
WCAG 2.2 AA and keyboard-only use (SC-008); no personal data in server logs

**Scale/Scope**: 2 pages + 1 section reused on 5 record pages; 7 API operations (3 new, 4 extended from 001)

## Constitution Check

*GATE: checked before design and again after it.*

| Principle | How this plan complies | Result |
|---|---|---|
| I. Spec first | Every element traces to FR-001–FR-032, a story, an edge case or an SC (traceability table). No `[NEEDS CLARIFICATION]` in the spec | Pass |
| II. Shell business, plain language | Shell levels in every sentence; Hive9's "Plan Log" is "Activity log"; every text in *User-visible messages* | Pass |
| III. Simple, friendly experience | Standard list pattern (search, pills, count, empty states); destructive actions confirmed with **Cancel** focused; restore states explained on the row; nothing disabled without a reason (Restore stays visible when it cannot work, FR-024); summary cards link to their filters | Pass |
| IV. Simple, layered architecture | Extends `history` (handler → service → repository) instead of adding a package; every rule (restore state, scope matching, counts, record state) runs in Go; the web app only formats, groups by local day and filters the small bin list | Pass |
| V. Tested business rules | One Go test per rule (named with the FR), a contract test per operation, a component test per page and dialog, Playwright journeys for US1–US3 (P1) and US4–US5; fixed clock everywhere | Pass |
| VI. Safe by default | Purge and empty need an explicit confirmation and are logged; the empty request names the entries the person saw; the actor is passed to every service call; entries stay append-only | Pass |
| VII. Keep it small | No new dependency, package or table; the preview reuses the restore code; one new error code | Pass |

**Post-design re-check**: Pass. One feature component crosses feature folders (Complexity Tracking).

## Decisions

No `[NEEDS CLARIFICATION]` markers. Decision-log defaults used: D-004 (no permissions; everyone may
restore, purge and empty), D-013 / domain model §2.5 (restored line items send their activity back),
D-019 (IDs kept and never reused), D-031 / R14 (Undo exact, later restore resets statuses), D-037 (Undo
window), D-038 (display preferences not logged), D-039 (entries carry path and readable values).

New decisions (numbered at consolidation, 2026-10-19):

| # | Decision | Reason |
|---|---|---|
| D-177 (011-A) | The restore confirmation is built from a **dry run**: `PreviewRestore` performs the real restore (parents first, every kind's `PrepareRestore`, the `AfterRestore` hook of 007/008) inside a `Write` that returns a sentinel error, so nothing is saved, and reports every changed field of every record and every hook note | SC-004 needs 100 % of status and name changes stated; a separate "predict" function per kind would drift from the real rules |
| D-178 (011-B) | The **Plan** filter and **Include items inside** match an entry when its record is the chosen record, when the chosen record is in the entry's stored path, or when the record was later moved into the chosen record (a `moved` entry whose path contains it). Once a record matched, all its entries match | Uses only what entries store (works for deleted and permanently deleted records); no walk of today's tree for 50,000 entries |
| D-179 (011-C) | Time boundaries are computed by the **browser** in the person's time zone and sent as UTC instants (`todayStart`, `weekStart`, `from`, `to`); the week starts on Monday; entries are grouped by local day in the browser and each day heading counts the entries loaded under it | Only the browser knows the person's time zone (001 research R5); no time-zone database on the server |
| D-180 (011-D) | **Empty recycle bin** sends the IDs of the entries the person saw; entries deleted after the dialog opened stay in the bin | Nothing is purged without a confirmation that showed it (SC-007) |
| D-181 (011-E) | A purged entry stays in `recycle_bin` as resolved (`how: deletedPermanently`, by, at) without snapshots; entries that belonged inside it stay in the bin marked "Can't be restored" and can be purged separately | Lets "already done by someone else" name who and when (FR-032); keeps 001's resolved-entry model |
| D-182 (011-F) | New error code `parent_deleted_permanently` (409) for a restore whose parent (or an ancestor) was purged; `restore_parent_first` stays for a parent that is still in the bin | The web app must explain the two cases differently (FR-023, FR-024) |
| D-183 (011-G) | The empty-bin entry has `recordType: "recycleBin"`, `recordId: ""`, the action `emptiedRecycleBin` and the purged entries in a new entry field `items`; `items` is `[]` on every other entry (also old ones) | FR-030 asks for one entry with the list |
| D-184 (011-H) | One per-person preference `historyIncludeInside` (default off) serves both **Include items inside** and **Include line items** | FR-012 asks for one remembered choice; display preference, not logged (D-038) |
| D-185 (011-I) | The recycle-bin list is filtered and searched **in the browser** (the API returns every entry with its contents); the activity log is filtered on the server | The bin holds tens of entries; the log holds up to 50,000 |
| D-186 (011-J) | Bin entries holding a kind that is not registered in this API (developer sample records outside local/development) are left out of the list, the preview, purge and empty | Nothing can be done with them there (001 answers "This item can't be restored here.") |
| D-187 (011-K) | `HistorySection` is a feature component in `features/activity-log/` imported by the record pages of 004–007 | It loads data, so it does not belong in the presentational `@/components`; it is owned by this feature |
| D-188 (011-L) | The demo history is written by one seed step of 011 (`seed/history.go`) after all other seed steps, in time order; seed steps of 002–008 write no log entries | Entries are ordered by sequence number; back-dated entries appended by several steps would be out of order |

## Project Structure

### Documentation (this feature)

```text
specs/011-activity-log-and-recycle-bin/
├── spec.md · checklists/requirements.md
├── plan.md                  # this file
├── data-model.md            # stored and derived types
├── contracts/openapi.yaml   # fragment: changed and new paths and schemas
└── tasks.md
```

### Source code (repository root)

```text
backend/
├── api/openapi.yaml                      # merge the fragment (changed: 4 operations, 5 schemas; new: 3 operations)
├── internal/platform/errs/errs.go        # + CodeParentDeletedPermanently, ParentDeletedPermanently(...) (+ errs_test.go)
├── internal/history/
│   ├── model.go                          # + actions, LogItem, EntryView, RecordNow, LogQuery, LogPage fields,
│   │                                     #   LogSummary, BinListItem, BinRecord, LabelledRef, RestorePreview,
│   │                                     #   RecordChange, PurgeResult, EmptyResult
│   ├── service.go                        # refactor: appendEntry, putBack → restoreEntry, missing-parent check,
│   │                                     #   log collector for the dry run; ListLog delegates to QueryLog
│   ├── query.go · query_test.go          # NEW: QueryLog, LogSummary, scope expansion, record state
│   ├── bin.go · bin_test.go              # NEW: ListBinItems, restore states, PreviewRestore, RestoreWithParents,
│   │                                     #   Purge, EmptyBin
│   └── handler.go · handler_test.go      # extended and new routes
├── internal/people/model.go · service.go # + historyIncludeInside preference (+ tests)
└── seed/history.go · seed.go · seed_test.go   # NEW step addHistory (two weeks of history, one deleted initiative)

frontend/src/
├── app/router.tsx · navigation.ts        # /activity-log and /recycle-bin leave COMING_SOON_SECTIONS (+ tests)
├── api/errors.ts                         # + kind parentDeletedPermanently; resolvedMessage knows deletedPermanently (+ test)
├── lib/format.ts                         # + formatWeekdayDate (+ format.test.ts); uses formatRelativeTime (008/009)
├── features/activity-log/
│   ├── ActivityLogPage.tsx (+ .module.css, .test.tsx)
│   ├── LogFilters.tsx (+ .module.css, .test.tsx)
│   ├── LogEntryList.tsx (+ .module.css, .test.tsx)     # day groups + Load more
│   ├── LogEntry.tsx (+ .module.css, .test.tsx)          # one entry, expandable
│   ├── HistorySection.tsx (+ .module.css, .test.tsx)    # used by the record pages of 004–007
│   ├── showHistory.ts                                   # scrolls to and focuses #history (for ⋯ More › View history)
│   ├── logFilters.ts (+ .test.ts)                       # URL ⇄ filters ⇄ API query; local date ranges
│   ├── logText.ts (+ .test.ts)                          # sentences, action chips, detail lines, day labels
│   └── activityLogApi.ts                                # queries
├── features/recycle-bin/
│   ├── RecycleBinPage.tsx (+ .module.css, .test.tsx)
│   ├── BinRow.tsx (+ .module.css, .test.tsx)
│   ├── RestoreDialog.tsx (+ .test.tsx) · PurgeDialog.tsx (+ .test.tsx) · EmptyBinDialog.tsx (+ .test.tsx)
│   ├── binText.ts (+ .test.ts)                          # count lines, restore sentences, search matching
│   └── recycleBinApi.ts                                 # queries and mutations
└── features/{plans,campaigns,activities,line-items}/…  # 004–007 pages: add <HistorySection> and wire View history

frontend/e2e/activity-log.spec.ts · recycle-bin.spec.ts  # Playwright
```

**Structure decision**: as 001; `internal/history` grows two files (`query.go`, `bin.go`) so `service.go`
stays readable.

## Backend design

### Types (Go, `internal/history/model.go`)

```go
const (
    ActionDeletedPermanently Action = "deletedPermanently" // one entry per Delete permanently (FR-030)
    ActionEmptiedRecycleBin  Action = "emptiedRecycleBin"  // one entry per Empty recycle bin (FR-030)
)

// Entry gains one stored field (data-model §1); old entries decode with nil and are served as [].
//   Items []LogItem `json:"items"`

type LogItem struct { // one purged entry listed in an emptiedRecycleBin entry
    RecordType      string `json:"recordType"`
    RecordTypeLabel string `json:"recordTypeLabel"`
    RecordID        string `json:"recordId"`
    RecordName      string `json:"recordName"`
    Details         string `json:"details"` // contentsText: "with 2 activities and 5 line items"; "" when nothing inside
}

type RecordState string // "exists" | "inRecycleBin" | "deletedPermanently" | "none"

type RecordNow struct { // where the entry's record is now (FR-008)
    State      RecordState `json:"state"`
    Name       string      `json:"name"`       // current name (exists) or name in the bin; "" otherwise
    Path       []RecordRef `json:"path"`       // current ancestors when it exists; [] otherwise
    BinEntryID string      `json:"binEntryId"` // the unresolved bin entry holding it; "" otherwise
}

type EntryView struct { // ActivityLogEntry as the API returns it
    Entry                         // flattened
    RecordNow RecordNow `json:"recordNow"`
}

type LogQuery struct {
    ActorIDs      []string   // who acted; empty = anyone
    RecordTypes   []string   // kinds; empty = every kind
    Actions       []string   // filter values (see "Action filter"); empty = every action
    PlanIDs       []string   // entries about these plans and anything that is or was inside them (D-178); nil = no restriction
    RecordID      string     // one record
    IncludeInside bool       // with RecordID: also entries of records that are or were below it
    From, To      *time.Time // From inclusive, To exclusive
    Search        string     // record name, record ID or person name, ignoring letter case
    BeforeSeq     int        // only entries with a smaller seq (Load more); 0 = from the newest
    Limit         int        // 1–200
}

// LogPage (exists) becomes: Items []EntryView; Total int (matching, ignoring BeforeSeq);
// UnfilteredTotal int (whole log); Record *RecordRef (the record of RecordID: current name, or the
// latest name in the log; nil without RecordID or when unknown).

type LogSummary struct {
    ChangesToday, ChangesThisWeek, PeopleActiveThisWeek, DeletionsThisWeek, Total int
    People      []actor.Actor     // everyone with an entry, sorted by name (Person filter)
    RecordTypes []RecordTypeCount // {type, label, count}, types present except "recycleBin", sorted by label
    Plans       []RecordRef       // plans named in the log (latest name), sorted by name (Plan filter)
}

type LabelledRef struct { Type, TypeLabel, ID, Name string } // json: type, typeLabel, id, name

type BinRecord struct { RecordType, RecordTypeLabel, RecordID, RecordName, ParentID string } // a snapshot without data

type BinListItem struct {
    BinEntryView                         // flattened: 001's RecycleBinEntry
    RecordTypePlural string              // the kind's plural, for the type pills
    RestoreState     string              // "restorable" | "parentInRecycleBin" | "parentDeletedPermanently"
    MissingParent    *LabelledRef        // the parent in the bin, or the purged ancestor; nil when restorable
    ParentEntryIDs   []string            // entries "Restore all" brings back first, top-most first; [] otherwise
    InsideEntryIDs   []string            // other entries whose restore needs this one (FR-028)
    Records          []BinRecord         // the record first, then everything inside, parents first
}

type RecordChange struct { // one changed field of one record, in a preview
    RecordType, RecordTypeLabel, RecordID, RecordName string
    Restored                                          bool   // true: the record comes back; false: an existing record the restore changes (FR-020)
    Field, Label, OldText, NewText                    string
}

type RestorePreview struct {
    Entry         BinListItem
    Destination   []RecordRef    // the parent's current path including the parent; [] for top-level records
    ParentEntries []BinListItem  // entries restored first (Restore both / all N), top-most first
    Changes       []RecordChange // every changed field, records in restore order
    Notes         []string       // the PrepareRestore / AfterRestore notes, in order
}

type PurgeResult struct { RecordType, RecordTypeLabel, RecordID, RecordName string }
type EmptyResult struct { Count int }
```

### Service operations

All take `(ctx, a actor.Actor, …)`. Reads use `tx.Read`; changes one `tx.Write`; all join a caller's
transaction.

| Operation | Rules (FR) |
|---|---|
| `QueryLog(ctx, a, LogQuery) (LogPage, error)` | Validate (all problems at once): `Limit` 1–200 (`limit`); each action one of the 8 filter values (`action`); `To` after `From` (`to`); `BeforeSeq` ≥ 0. Read all entries; match every filter (AND); newest first by `seq` (FR-004); `Total` = matching count before `BeforeSeq`; then entries with `seq < BeforeSeq`, at most `Limit` (FR-009); `UnfilteredTotal` = all entries (FR-003 "Showing N of M"); each item gets `RecordNow` (FR-008) and `Items` normalised to `[]`. Filters: actor ID equal; record type equal; action per *Action filter*; plan and record-with-inside per *Scope matching*; record without inside: `RecordID` equal; `At ≥ From`, `At < To`; search = case-insensitive substring of `RecordName`, `RecordID` or `Actor.Name` (FR-003). Nothing is written (FR-010) |
| `ListLog(ctx, a, LogFilter)` (001) | Kept for 001 callers: maps `RecordType`, `RecordID`, `Limit` onto `QueryLog` and returns `(entries, total)` |
| `LogSummary(ctx, a, todayStart, weekStart time.Time) (LogSummary, error)` | Both required, `weekStart ≤ todayStart` (else `validation_failed` on `weekStart`). Over the **whole** log, ignoring filters (FR-002): today = `At ≥ todayStart`; week = `At ≥ weekStart`; people = distinct `Actor.ID` this week; deletions = actions `deleted`, `deletedPermanently`, `emptiedRecycleBin` this week; facets as the type says |
| `ListBinItems(ctx, a) ([]BinListItem, error)` | Unresolved entries whose kinds are all registered (D-186), newest first (FR-015); restore state per *Restore state*; `InsideEntryIDs` = entries in state `parentInRecycleBin` whose `ParentEntryIDs` contain this entry (FR-028); `Records` from the snapshots |
| `PreviewRestore(ctx, a, entryID) (RestorePreview, error)` | Unknown → `404 not_found` "This item is not in the recycle bin."; kind not registered → `404` "This item can't be restored here."; resolved → `409 already_resolved` (who, when). State `parentDeletedPermanently` → preview with `Destination [] , ParentEntries [], Changes [], Notes []` (the dialog explains, FR-024). Otherwise the **dry run** (D-177): in one `Write` with a log collector in the context, restore `ParentEntries` then the entry exactly as `RestoreWithParents`, record `before/after` of every snapshot → `Changes` with `Restored: true` (field changes from `Diff` + the kind's `Describe`), every collected log entry of action `changed`/`statusChanged` about a record not restored → `Changes` with `Restored: false` (FR-020), `Destination` = ancestors of the entry's parent **after** the parents came back (edge case: the parent's current place), then return the sentinel `errDryRun` so the store keeps nothing (FR-018, FR-019, FR-020, FR-022, FR-023) |
| `Restore(ctx, a, entryID)` (001, refined) | 001's checks; a missing parent that is in the bin → `409 restore_parent_first` (parent); a missing parent (or ancestor) that is neither stored nor in the bin → `409 parent_deleted_permanently` (FR-024); restore rules come from the kinds (`PrepareRestore`: plans → Draft (004), activities → Created (006), name suffix (004–007, FR-022); `AfterRestore` of 007/008: activity back to Created, FR-020); never blocked by any other rule (FR-025); entry resolved `restored`, one `restored` log entry with the actor, the top record's changes and the details "Restored from the recycle bin, with …. <notes>" (FR-026); IDs and values kept (FR-021) |
| `RestoreWithParents(ctx, a, entryID)` | Same checks; then, in one `Write`, restore each entry of `ParentEntryIDs` (top-most first) and then the entry, each with its own `restored` log entry (FR-023). With nothing missing it equals `Restore`. A purged ancestor → `parent_deleted_permanently`, nothing changes |
| `Undo(ctx, a, entryID)` (001) | Unchanged: exact, statuses included (FR-027); the refined missing-parent check applies |
| `Purge(ctx, a, entryID) (PurgeResult, error)` | Unknown / unregistered → `404`; resolved → `409 already_resolved` (FR-032). Sets `Resolution{how: deletedPermanently, by: a, at: now}`, empties `Snapshots` (the records are removed for good; IDs are never reused, FR-030); appends one entry: action `deletedPermanently`, verb `permanently deleted`, the entry's type, label, ID, name and **stored path**, details = contents text (FR-030). Entries inside it stay in the bin (D-181). Log entries about the records are kept (append-only) |
| `EmptyBin(ctx, a, entryIDs []string) (EmptyResult, error)` | `entryIDs` must hold at least one ID (`validation_failed` on `entryIds`, "Reload the page and try again."). In one `Write`, purge every listed entry that is still unresolved and registered (duplicates once; others skipped, D-180); when at least one was purged, append **one** entry: action `emptiedRecycleBin`, verb `emptied the recycle bin`, `recordType "recycleBin"`, `recordTypeLabel "recycle bin"`, `recordId ""`, `recordName "Recycle bin"`, path `[]`, details "<n> items" / "1 item", `items` = the purged entries (FR-029, FR-030). Returns the count |

Nothing removes entries from the bin on a timer (FR-031): there is no scheduled job.

**Action filter** (FR-003): the values `created`, `changed`, `statusChanged`, `moved`, `copied`, `deleted`,
`restored` match the stored action of the same name; `deletedPermanently` matches the stored actions
`deletedPermanently` **and** `emptiedRecycleBin`.

**Scope matching** (D-178), used for `PlanIDs` and for `RecordID` with `IncludeInside`:
1. `S` = the chosen record IDs.
2. Walk all entries in `seq` order; for an entry with action `moved` whose path contains an ID in `S`, add
   its `RecordID` to `S` (the record moved in; features log a move after the record is saved in its new
   place, so the path is the new place).
3. An entry matches when its `RecordID` is in `S` or any `path[i].id` is in `S`.

**Record state** (`RecordNow`, FR-008), per item, with an index `recordID → unresolved bin entry` built
from the snapshots of unresolved entries:
- `emptiedRecycleBin` entry, or a kind that is not registered → `none`;
- the kind's `Snapshot` finds it → `exists`, current `Name`, `Path` = current ancestors;
- the index holds it → `inRecycleBin`, `Name` = snapshot name, `BinEntryID`;
- otherwise → `deletedPermanently`.

**Restore state** (FR-017), per unresolved entry `E`; `p := E.Parent`, chain `C := []`:
1. `p == nil`, or `p` is stored (its kind registered and `Snapshot` finds it) → stop.
2. `p.ID` is in an unresolved entry `F` (any of its snapshots) → append `F` to `C`; `p := F.Parent`; repeat
   (a visited set prevents loops).
3. Otherwise `p` is gone for good → **blocked** at `p`.

State: blocked → `parentDeletedPermanently`, `MissingParent` = the gone record (with its kind's label);
`C` not empty → `parentInRecycleBin`, `MissingParent` = `E.Parent`, `ParentEntryIDs` = `C` reversed (the
top-most first, the restore order); otherwise `restorable`.

### Changes to existing history code (`service.go`)

- `appendEntry(ctx, a, e Entry) (Entry, error)`: gives the id and `seq` (`NextNumber("LOG")`), `At`,
  `Actor`, normalises `Changes`/`Items` to `[]`, appends; `Log` builds its entry and calls it; `Purge` and
  `EmptyBin` call it with their own fields (their records are no longer stored, so `Log`'s path lookup
  cannot run). When the context carries a **log collector** (dry run), `appendEntry` also adds a copy.
- `putBack` is split: `restoreEntry(ctx, a, entry, undo, now) ([]restoredPair, error)` puts the
  snapshots back (calling `PrepareRestore`, and `AfterRestore` once 007/008 add it), resolves the entry,
  writes the `restored` entry, and returns every `{kind, before, after}`; `putBack` (Undo, Restore) and
  `RestoreWithParents` call it.
- `checkCanComeBack`: the "parent not stored" branch looks the parent up in the unresolved bin entries:
  found → `RestoreParentFirst(parent)`; not found → `ParentDeletedPermanently(entryKindLabel, parentLabelled)`.

### Errors and messages (API)

| Case | Status · code | `message` |
|---|---|---|
| Unknown bin entry | 404 `not_found` | This item is not in the recycle bin. |
| Kind not registered | 404 `not_found` | This item can't be restored here. |
| Already undone, restored or purged | 409 `already_resolved` (+ `resolution`) | '<name>' is no longer in the recycle bin. |
| Parent in the bin (Restore without parents) | 409 `restore_parent_first` (+ `parent`) | Restore '<parent name>' first |
| Parent or ancestor purged | 409 `parent_deleted_permanently` (+ `parent`) | This <level> can't be restored because its <parent level> '<parent name>' was permanently deleted. |
| `limit` invalid | 400 `validation_failed`, field `limit` | Choose a number from 1 to 200. |
| `action` invalid | field `action` | Choose an action from the list. |
| `from` / `to` / `todayStart` / `weekStart` not a timestamp | field of the same name | Choose a valid date. |
| `to` not after `from`; `weekStart` after `todayStart` | field `to` / `weekStart` | The end date must be on or after the start date. / Reload the page and try again. |
| `beforeSeq` < 1 or not a number; `includeInside` not true/false; `withParents` not true/false | field of the same name | Reload the page and try again. |
| `todayStart`/`weekStart` missing; `entryIds` empty | field of the same name | Reload the page and try again. |

`errs.ParentDeletedPermanently(level string, parent RecordRef, parentLevel string) *Error` (409, sets
`Parent`) — added to `internal/platform/errs` with its code constant.

### API operations

Contract: [contracts/openapi.yaml](contracts/openapi.yaml). All need a session (`401` otherwise).

| Method and path | Request | Response | Errors |
|---|---|---|---|
| `GET /api/activity-log` (extended) | query `actorId`, `recordType`, `action` (repeatable), `planId`, `recordId`, `includeInside`, `from`, `to`, `q`, `beforeSeq`, `limit` (default 50) | `200 ActivityLogPage {items: ActivityLogEntry[], total, unfilteredTotal, record}` | 400 |
| `GET /api/activity-log/summary` (new) | query `todayStart`, `weekStart` (both required) | `200 ActivityLogSummary` | 400 |
| `GET /api/recycle-bin` (extended) | — | `200 RecycleBinPage {items: RecycleBinListItem[], total}` | — |
| `GET /api/recycle-bin/{entryId}/restore-preview` (new) | — | `200 RestorePreview` | 404, 409 `already_resolved` |
| `POST /api/recycle-bin/{entryId}/restore` (extended) | query `withParents` (default false) | `200 RestoreResult` | 404, 409 `already_resolved` · `restore_parent_first` · `parent_deleted_permanently` |
| `DELETE /api/recycle-bin/{entryId}` (new) | — | `200 PurgeResult` | 404, 409 `already_resolved` |
| `POST /api/recycle-bin/empty` (new) | `EmptyRecycleBinRequest {entryIds: string[]}` | `200 EmptyRecycleBinResult {count}` | 400 |

`POST /api/recycle-bin/{entryId}/undo` is unchanged. `ActivityLogEntry` gains `items` and `recordNow`;
`Error.code` gains `parent_deleted_permanently`; `Resolution.how` gains `deletedPermanently`;
`Preferences`/`PreferencesUpdate` gain `historyIncludeInside`.

### History kinds, activity-log entries, preferences

- **Kinds**: this feature registers none. It relies on the kinds of 004–007 (`plan`, `campaign`,
  `initiative`, `activity`, `lineItem`) and their `PrepareRestore`/`AfterRestore` hooks.
- **Log entries written**: Restore / Restore both → one `restored` per entry brought back; Delete
  permanently → one `deletedPermanently`; Empty → one `emptiedRecycleBin` with `items`; reads and the
  preview → none (the dry run's entries are rolled back).
- **Preference** `historyIncludeInside` (people package): `PreferencesUpdate.HistoryIncludeInside *bool`;
  default `false`; not logged, no version (D-038).

### Demo data (`backend/seed/history.go`, step `addHistory`, last step of `seed.Load`)

Written straight into the history repository (no service), in time order, so `seq` follows time:

1. For every seeded plan, campaign, initiative, activity and line item (read through the repositories of
   004–007): one `created` entry by its owner (Alex Morgan when it has none), between 14 and 8 days before
   the seed time, parents before children, with its path.
2. For every Published plan: one `statusChanged` entry, verb `published`, by its owner, 7 days before.
3. For every approval-history record seeded by 008 (submitted, approved, declined, …): one
   `statusChanged` entry with 008's verb, actor, time and comment (008 seeds them within the last 14 days).
4. One `changed` entry by Priya Raman 5 days before: planned cost of her first FY2026 Lubricants Europe
   activity with a planned cost, from (current − 5,000.00) to current, texts as money "40,000.00 USD".
5. The **deleted initiative**: initiative "Trade shows" (next `I-` id) in campaign "Rimula fleet" of
   FY2026 Lubricants Europe, owner Priya Raman, dates of the campaign, budget 20,000.00 USD; activity
   "Fleet expo stand" (next `A-` id, type Event, Created, planned cost 18,000.00 USD); line item "Stand
   build" (next `L-` id, USD 18,000.00). `created` entries 12 days before; then one bin entry `BIN-…` by
   Priya Raman 2 days before (path FY2026 Lubricants Europe › Rimula fleet, contents 1 activity and 1 line
   item, snapshots built from 005–007's stored types — the records are **not** put in their tables) and
   its `deleted` entry "with 1 activity and 1 line item".

`seed.Load` prints "…: 6 people." today; the seed message is extended by the consolidation step.

## Frontend design

### Routes and navigation

| Path | Page | Tab title |
|---|---|---|
| `/activity-log` | `ActivityLogPage` | Activity log · NorthStar |
| `/recycle-bin` | `RecycleBinPage` | Recycle bin · NorthStar |

`navigation.ts`: `RECYCLE_BIN` and `ACTIVITY_LOG` become plain `NavItem`s (no `comingSoon`) and leave
`COMING_SOON_SECTIONS`; `router.tsx` adds both routes inside `RequireSignIn` with `handle.title`.

### Activity log page (US2, FR-001–FR-010)

- `PageHeader`: eyebrow, title, highlight, description of the messages table. No primary action (FR-010).
- **Summary cards** (`SummaryCard` with `tip`, `to` and `linkLabel` "show these entries", the link variant added by 009 — see Dependencies):
  Changes today (`?range=today`), Changes this week (`?range=week`), People active this week
  (`?range=week`), Deletions this week (`?range=week&action=deleted&action=deletedPermanently`). Data:
  `GET /api/activity-log/summary` with `todayStart` = local midnight today and `weekStart` = local midnight of
  this week's Monday (`logFilters.ts`), refetched when the page regains focus.
- **Filters** (`LogFilters`), all kept in the URL (so links, Back and **Open in activity log** work):

  | URL | Control | API |
  |---|---|---|
  | `person=U-1003` | select "Person" (Anyone + summary `people`) | `actorId` |
  | `type=activity` | select "Record type" (Any type + summary `recordTypes`, label capitalised) | `recordType` |
  | `action=…` (repeat) | "Action" menu with 8 checkboxes (Radix DropdownMenu `modal={false}`) | `action` |
  | `plan=P-1001` | select "Plan" (Any plan + summary `plans`) | `planId` |
  | `record=A-1042&inside=1` | none (set by History, FR-013); chip only | `recordId`, `includeInside` |
  | `range=today\|week\|30d\|year\|custom`, `from=YYYY-MM-DD`, `to=YYYY-MM-DD` | pill group "Any time · Today · This week · Last 30 days · This year · Custom" + two date inputs for Custom | `from`/`to` instants: today = local midnight; week = Monday midnight; 30d = midnight of today − 29 days; year = 1 Jan midnight; custom = midnight of `from` and midnight after `to` (exclusive) |
  | `q=film` | search box "Search name, ID or person" (applied 300 ms after typing stops) | `q` |

  Every active filter is a removable chip (button "Remove filter <chip text>"), then **Clear all**; the
  count line reads "Showing N of M entries" when a filter is active, otherwise "M entries". Custom with
  `to` before `from` shows the field error and sends nothing.
- **List** (`LogEntryList`): `useInfiniteQuery` (`limit=50`, next page `beforeSeq` = last item's `seq`);
  entries grouped by local day: heading "Today", "Yesterday", or "Monday 12 Oct 2026" with
  "· n entries" (entries loaded under it, D-179); under the list, when loaded < `total`, "Showing the
  last <loaded> of <total>" and **Load more** (FR-009).
- **Entry** (`LogEntry`, FR-005–FR-008): `Avatar` (initials from the actor's name), sentence (see
  `logText`), record ID, "in <path names joined by ' › '>" when the stored path is not empty, the time
  (`formatTime`) with the full date and time in a `Tooltip`, and the action chip (`StatusChip`):

  | Action | Word | Tone |
  |---|---|---|
  | created | Created | positive |
  | changed · moved · copied | Changed · Moved · Copied | neutral |
  | statusChanged · restored | Status changed · Restored | active |
  | deleted | Deleted | problem |
  | deletedPermanently · emptiedRecycleBin | Permanently deleted | problem |

  Record name: `recordNow.state` `exists` → link to `recordHref` (with " — now '<name>'" when
  `recordNow.name` differs); `inRecycleBin` → link to `/recycle-bin?record=<recordId>`; `deletedPermanently`
  → plain text + " (permanently deleted)"; `none` → plain text. **Show details** / **Hide details**
  (`aria-expanded`) appears when there is anything to show and lists (FR-007): each change
  "<label>: <oldText> → <newText>" (full text, wrapping); "Comment: <comment>"; details by action
  (deleted: "Deleted <details>"; others: the details as stored); for `emptiedRecycleBin` one line per item
  "<Type label capitalised> '<name>' (<id>) <details>". No edit or delete control exists (FR-010).
- States: loading → `Skeleton` (cards and 5 rows); error → `ErrorState`; empty log → `EmptyState` "No
  changes yet"; filters hide everything → "Nothing matches these filters" with **Clear filters**.

### History section (US3, FR-011–FR-013)

`<HistorySection recordType recordId recordName level />` — a `Card` with `id="history"` titled "History":

- `level` `'plan' | 'campaign' | 'initiative'` shows the checkbox **Include items inside**; `'activity'`
  shows **Include line items**; `'lineItem'` shows none. Its value is the person's
  `preferences.historyIncludeInside`; changing it saves `PATCH /api/me/preferences
  {historyIncludeInside}` at once (the 001 preference pattern: the screen changes first; a failed save
  shows "Your choice couldn't be saved." with **Try again**).
- List: `GET /api/activity-log?recordId=…&includeInside=…&limit=20`, `useInfiniteQuery` with **Load more**,
  rendered with `LogEntryList` (same day groups and `LogEntry`). Entries about another record name it in
  their sentence (FR-012).
- **Open in activity log** → `/activity-log?record=<id>` plus `&inside=1` when included (FR-013); the log
  page shows the chip "Record: <Type label> '<name>' (<id>)" (+ " and items inside"), name from the page's
  `record` field.
- `showHistory()` (`showHistory.ts`) scrolls `#history` into view and focuses its heading: the
  **View history** items of 004–007's **⋯ More** menus call it (replacing their "Coming soon").
- Empty: "No changes recorded yet."; error: `ErrorState`.

### Recycle bin page (US1, US4, US5, FR-014–FR-032)

- `PageHeader` with the count line as description and, while the bin is not empty, **Empty recycle bin**
  (`Button variant="danger"`).
- Toolbar: search "Search name or ID" (matches the entry's record and every record in `records`, ignoring
  letter case, FR-016); type pills "All · Plans · Campaigns · Initiatives · Activities · Line items" plus
  the `recordTypePlural` (capitalised) of any other type present; "Showing N of M" (FR-016). Filtering is
  in the browser (D-185).
- Row (`BinRow`): level tag (004's `LevelTag` for the five planning types; a neutral `StatusChip` with the
  capitalised type label otherwise), name, ID; "Was in <path>" (omitted for top-level records); contents
  "2 activities · 5 line items" (or "Nothing inside"); `Avatar` + "Deleted by <name>" + relative time with
  the full date and time in a `Tooltip`; the restore-state line when not restorable (FR-017); a search hit
  inside the contents: "Contains <id> '<name>'" (+ " and n more"); **Restore** (`aria-label` "Restore <type
  label> '<name>'"); `ActionsMenu` ("Actions for <type label> '<name>'") with **Delete permanently**
  (danger); **Show contents** / **Hide contents** when there are contents → a nested list (tree by
  `parentId`) "<Type label> '<name>' (<id>)".
- `?record=<recordId>` (links from the log): the row whose `records` contain it is scrolled into view,
  expanded when the record is inside, marked (`data-highlighted`) and its heading focused; when no row
  holds it, the notice "That item is no longer in the recycle bin." appears above the list.
- **Restore** opens `RestoreDialog` (`ConfirmDialog`), which loads `GET …/restore-preview`:
  - restorable / parent in the bin: title "Restore <type label> '<name>'?"; the destination sentence; one
    sentence per change (*Restore sentences*); notes as given; for a parent in the bin the parent block
    and the confirm label **Restore both** (2 entries) or **Restore all <N>**; otherwise **Restore <type
    label>**. Confirm → `POST …/restore?withParents=true|false`; success → close, toast "<Type label
    capitalised> '<name>' restored — View" (View → `recordHref`), invalidate every query except
    `['session']`.
  - parent deleted permanently: title "This <type label> can't be restored"; the explanation; confirm
    label **Delete permanently** (danger) → closes and opens `PurgeDialog` (FR-024; nothing changes).
  - preview loading → `Skeleton` inside the dialog; preview or restore failure: `already_resolved` →
    `resolvedMessage` toast, close, refetch (FR-032); `restore_parent_first` / `parent_deleted_permanently`
    → the API message as a toast, close, refetch; others → the API message in the dialog.
- **Delete permanently** → `PurgeDialog` (`tone="danger"`, **Cancel** focused, FR-028) → `DELETE …` →
  toast "<Type label capitalised> '<name>' deleted permanently"; `already_resolved` as above.
- **Empty recycle bin** → `EmptyBinDialog` (FR-029) with the IDs of all listed entries (not only the
  filtered ones) → `POST /api/recycle-bin/empty` → toast by count.
- States: loading → `Skeleton` rows; error → `ErrorState`; empty bin → `EmptyState` (messages table);
  filters hide everything → "Nothing matches these filters" with **Clear filters**.

**Restore sentences** (`binText.ts`), from `RestorePreview.changes` (Subject = "It" for the chosen
entry's own record, otherwise "<Type label capitalised> '<name>' (<id>)"):

| Change | Sentence |
|---|---|
| `status`, `restored`, type `activity` | <Subject> comes back as <newText> and must be submitted for approval again. |
| `status`, `restored`, type `plan` | <Subject> comes back as <newText> and must be published again. |
| `status`, not `restored`, type `activity` | <Subject> goes back to <newText> because its line items change, and must be submitted for approval again. |
| `status`, any other | <Subject> changes from <oldText> to <newText>. |
| `name` | <Subject> comes back as '<newText>' because the name '<oldText>' is taken. |
| any other field | <Subject>: <label> changes from <oldText> to <newText>. |

Destination sentence: "It returns to <destination names joined by ' › '>" + (contents ? " with <contents
text>" : "") + "."; with an empty destination: plans "It returns to the Plans list"; other types "It
returns to its list". Contents text: counts joined "2 activities and 5 line items" (labels from
`contents`).

### API errors in the web app

`api/errors.ts`: `KIND_BY_CODE.parent_deleted_permanently = 'parentDeletedPermanently'` (new
`ApiErrorKind`); `resolvedMessage` adds `how === 'deletedPermanently'` → "deleted it permanently".

### Shared formatting

- `formatWeekdayDate(date: string)` → "Monday 12 Oct 2026" (new in `lib/format.ts`).
- `formatRelativeTime(timestamp, now?)` → "just now", "5 minutes ago", "3 hours ago", "yesterday",
  "4 days ago", then `formatDate` (from 008, or added by 009 — see Dependencies).

### Query keys

`['activity-log', 'summary', todayStart, weekStart]`, `['activity-log', 'entries', apiQuery]`,
`['history', recordId, includeInside]`, `['recycle-bin']`, `['recycle-bin', 'preview', entryId]`.
Restores, purges and empties invalidate everything except `['session']` (a restore changes plans,
totals, Home and approvals).

## User-visible messages

Every text this feature shows; code, tests and e2e quote this table. `<…>` is filled in at run time.

| Where | Text |
|---|---|
| Log header | Activity log · Every change, · on the record. · See who changed what, and when. Entries can't be edited or removed. |
| Log cards and tips | Changes today — Entries since midnight, in your time zone. · Changes this week — Entries since Monday. · People active this week — Different people who changed something since Monday. · Deletions this week — Deletes, permanent deletes and emptied recycle bins since Monday. |
| Card link names | <label>: <value> — show these entries |
| Log filters | Person · Anyone · Record type · Any type · Action · Any action · Plan · Any plan · Any time · Today · This week · Last 30 days · This year · Custom · From · To · Search name, ID or person |
| Action words | Created · Changed · Status changed · Moved · Copied · Deleted · Restored · Permanently deleted |
| Chips | Person: <name> · Type: <Type label> · Action: <word> · Plan: <name> · Record: <Type label> '<name>' (<id>) · Record: <Type label> '<name>' (<id>) and items inside · Date: <range word> · Date: <DD MMM YYYY> – <DD MMM YYYY> · Search: '<text>' · Remove filter <chip text> · Clear all |
| Count lines | Showing <N> of <M> entries · <M> entries (1: 1 entry) · Showing the last <loaded> of <N> · Load more |
| Day headings | Today · Yesterday · <Weekday DD MMM YYYY> · <n> entries (1 entry) |
| Entry sentence | <person> <verb> <type label> '<name>' · <person> emptied the recycle bin (<n> items) (1: (1 item)) · in <path> · — now '<current name>' · (permanently deleted) |
| Entry details | Show details · Hide details · <label>: <old> → <new> · Comment: <comment> · Deleted <details> |
| Log empty | No changes yet · Changes to plans, settings and people appear here. · Nothing matches these filters · Clear filters |
| Custom range error | The end date must be on or after the start date. |
| History section | History · Every change to this <type label>, newest first · Include items inside · Include line items · Open in activity log · No changes recorded yet. · Load more |
| Bin header | Recycle bin · Deleted, · not gone. · <n> items in the recycle bin · Items stay here until you restore or delete them (1: 1 item in the recycle bin · …) · Empty recycle bin |
| Bin toolbar | Search name or ID · All · Plans · Campaigns · Initiatives · Activities · Line items · <Other plural> · Showing <N> of <M> |
| Bin row | Was in <path> · Nothing inside · Deleted by <name> · Restore · Actions for <type label> '<name>' · Delete permanently · Show contents · Hide contents · Contains <id> '<name>' · and <n> more · Restore <type label> '<name>' |
| Restore states | Its <parent type label> '<name>' is also in the recycle bin · Can't be restored — its <parent type label> '<name>' was permanently deleted |
| Highlight miss | That item is no longer in the recycle bin. |
| Bin empty | Nothing in the recycle bin — when you delete a plan, campaign, initiative, activity or line item, it comes here so you can bring it back. · Nothing matches these filters · Clear filters |
| Restore dialog | Restore <type label> '<name>'? · (destination and change sentences above) · Its <parent type label> '<name>' is also in the recycle bin. Restoring brings these back first: · <Type label> '<name>' (<id>) with <contents> · Cancel · Restore <type label> · Restore both · Restore all <N> |
| Restore blocked dialog | This <type label> can't be restored · This <type label> can't be restored because its <parent type label> '<name>' was permanently deleted. You can delete it permanently instead. · Cancel · Delete permanently |
| Restore done | <Type label> '<name>' restored — View |
| Purge dialog | Delete '<name>' permanently? · This also deletes <contents>. · This can't be undone. · 1 other item in the recycle bin belongs inside it and can no longer be restored. · <n> other items in the recycle bin belong inside it and can no longer be restored. · Cancel · Delete permanently |
| Purge done | <Type label> '<name>' deleted permanently |
| Empty dialog | Empty the recycle bin? · This permanently deletes <n> items and everything inside them: <contents totals>. (1: 1 item) · This can't be undone. · Cancel · Empty recycle bin |
| Empty done | The recycle bin is empty — <n> items deleted permanently (1: 1 item deleted permanently) · Nothing was deleted — someone else already emptied the recycle bin. |
| Already resolved (web) | '<name>' is no longer in the recycle bin — <person> restored it / undid the delete / deleted it permanently at <HH:mm> (other day: on <DD MMM YYYY> at <HH:mm>) |
| Restore notes expected from kinds | Plan (004): '<name>' comes back as Draft — publish it again. · Activity (006): '<name>' comes back as Created — submit it for approval again. · Name rule (004–007): Renamed to '<new name>' because the name was taken. |
| API messages | see *Errors and messages (API)* |

## Dependencies

### Uses from feature 001

`history.Service` (`Register`, `Log`, `Diff`, `MoveToBin`, `Undo`, `Restore`, `ListLog`, `ListBin`, `Kind`,
`Snapshot`, `RestoreContext`, `BinEntry`, `Entry`, `Change`, `ContentCount`, `contentsText`), `errs`,
`store.Transactor`, `ids.Generator`, `record.Meta`, `apitest`; people preferences (`PATCH
/api/me/preferences`); web: `PageHeader`, `SummaryCard`, `Card`, `StatusChip`, `Avatar`, `Button`,
`ConfirmDialog`, `ActionsMenu`, `EmptyState`, `ErrorState`, `Skeleton`, `Tooltip`, `InfoTip`, `useToast`,
`call`, `ApiError`, `resolvedMessage`, `formatDate`, `formatTime`, `formatDateTime`, `localDate`,
`usePageTitle`, `useSession`/`useCurrentUser`.

### Expects from other features (by name; the consolidation step aligns names)

| From | What | Used for |
|---|---|---|
| 004–007 | Kinds registered with history: `plan`, `campaign`, `initiative`, `activity`, `lineItem` (singular "plan", "campaign", "initiative", "activity", "line item") | Every log and bin row |
| 004 / 006 | `PrepareRestore` on Restore: plan → Draft with the note "'<name>' comes back as Draft — publish it again."; activity → Created with "'<name>' comes back as Created — submit it for approval again." (R14); every planning kind: name suffix " (restored)", " (restored 2)" … with the 001 note (FR-022) | FR-019, FR-022 |
| 007 / 008 | `history.Kind.AfterRestore` (field and its call in `putBack`/`restoreEntry`) that, on a Restore (not Undo), sends a Submitted/Approved activity whose line items come back to Created **and logs it with `history.Log`** (`statusChanged`, change on `status`) | FR-020 and its preview |
| 004–008 | Features log a **move** after the record is saved in its new place (path = new place, change on `parent`); copies name their source in `details` | Scope matching, FR-007 |
| 004–008 | `Describe` hooks give readable texts (money "45,000.00 USD", option labels, people's names, "Approved", …) | FR-007 |
| 004–007 | Stored types `plans.Plan`, `campaigns.Campaign`, `campaigns.Initiative`, `activities.Activity`, `lineitems.LineItem` and `NewJSONRepository()` with `List(ctx)` each | Seed `addHistory` |
| 008 | Seeded approval history readable through its repository (`approvals.NewJSONRepository()` → history records with activity ID, action verb, actor, time, comment), times within the last 14 days; 002–008 seed steps write no log entries | Seed `addHistory` |
| 004–007 | The record pages / details views: `features/plans/PlanPage.tsx` (004), campaign and initiative details (005), `features/activities/ActivityPage.tsx` (006), line item details (007), each with a **⋯ More › View history** item | Where `HistorySection` goes |
| 004 | Shared components `LevelTag` (level tag with level colour), `PillFilter` (single-choice pill group), `SearchField`, `FilterChips` (removable chips + **Clear all**) | Bin row and toolbars; filter chips |
| 009 | `SummaryCard` props `to`, `linkLabel` and `detail` (link variant); `frontend/src/lib/recordLinks.ts` `recordHref(type, id)`; `formatRelativeTime` in `lib/format.ts` (or 008) | Cards, record links, relative times |

If 011 is built before 009, 011 adds the three 009 items exactly as 009's plan specifies them.

### Provides to later features

- `history.Service.QueryLog`, `LogQuery`, `EntryView`/`RecordNow` (**009 Recent changes**, 012 bulk actions'
  log checks), `LogSummary`.
- `RestoreWithParents`, `PreviewRestore`, `Purge`, `EmptyBin`; error code `parent_deleted_permanently`.
- Web: `HistorySection`, `showHistory()`, `LogEntryList`/`LogEntry` (009's Recent changes may render
  entries with `LogEntry` in its compact mode — see 009 plan), `recordHref` users.

## Tests

### Go service tests (`internal/history`, fixed clock, fake kinds `fakePlan` › `fakeItem` › `fakeLeaf` with a `status` and a `name`)

| Rule | Test |
|---|---|
| Person, type, action, plan, record, date and search filters, combined, applied to the whole log | `query_test.go` `TestQueryLog_FilterPerson_FR003`, `_FilterRecordType_FR003`, `_FilterAction_FR003` (`deletedPermanently` also matches `emptiedRecycleBin`), `_FilterDateRange_FR003` (From inclusive, To exclusive), `_Search_NameIDPerson_FR003`, `_CombinedFilters_TotalAndUnfiltered_FR003` |
| Plan / items-inside scope: path at the time, deleted records, records moved in | `TestQueryLog_PlanScope_IsOrWasInside_FR003`, `TestQueryLog_RecordWithItemsInside_FR012_FR013`, `TestQueryLog_MovedInRecordMatches_FR003` |
| Newest first, same-millisecond order | `TestQueryLog_NewestFirstBySeq_FR004` |
| 50 at a time, Load more by `beforeSeq`, totals unchanged | `TestQueryLog_BeforeSeqPaging_FR009` |
| Record state: exists (renamed), in bin (inside another entry), purged, none | `TestQueryLog_RecordNowStates_FR008` |
| Validation of limit, action, dates | `TestQueryLog_Validation_AllProblemsAtOnce` |
| Summary counts the whole log, local boundaries, deletions incl. purge and empty | `TestLogSummary_Counts_FR002`, `TestLogSummary_IgnoresFilters_FR002`, `TestLogSummary_Facets_FR003` |
| Entries are never changed (also after purge and empty) | `TestLog_EntriesNeverChangeOrDisappear_FR010_FR030` |
| Restore states incl. a two-level chain and a purged ancestor | `bin_test.go` `TestListBin_RestoreStates_FR017`, `TestListBin_ChainParentsFirst_FR023` |
| Rows carry records and inside entries; unregistered kinds hidden | `TestListBin_RecordsAndInside_FR015_FR028`, `TestListBin_UnregisteredKindHidden` |
| Preview: destination (parent moved), contents, status and name changes, outside changes, nothing saved | `TestPreviewRestore_DestinationIsCurrentPlace_EdgeCase`, `TestPreviewRestore_StatusChanges_FR018_FR019`, `TestPreviewRestore_NameClash_FR022`, `TestPreviewRestore_OutsideRecordChange_FR020`, `TestPreviewRestore_ChangesNothing_FR018` |
| Restore both / all N, parents first, one step; purged parent refused | `TestRestoreWithParents_ParentsFirstOneStep_FR023`, `TestRestore_ParentPurged_Refused_FR024`, `TestRestore_ParentInBin_RestoreParentFirst_FR023` |
| Same IDs and values; not blocked by other rules; logged with actor and changes | `TestRestore_SameIDsAndValues_FR021`, `TestRestore_NotBlockedByOtherRules_FR025`, `TestRestore_LogsActorAndStatusChanges_FR019_FR026` |
| Undo stays exact | `TestUndo_StillExact_FR027` |
| Purge: snapshots gone, resolution, one log entry with stored path; dependents stay unrestorable | `TestPurge_RemovesForGoodAndLogs_FR030`, `TestPurge_InsideEntriesBecomeUnrestorable_FR028` |
| Empty: one entry with the list; only listed, unresolved entries | `TestEmptyBin_OneEntryWithItems_FR029_FR030`, `TestEmptyBin_SkipsResolvedAndUnlisted_FR032` |
| Already done by someone else names who and when (restore after purge, purge after restore) | `TestActions_AlreadyResolved_FR032` |
| Nothing leaves the bin on its own (clock + 400 days) | `TestBin_NothingLeavesAutomatically_FR031` |
| New error | `platform/errs/errs_test.go` `TestParentDeletedPermanently` |
| Preference saved, default off, not logged | `people/service_test.go` `TestUpdatePreferences_HistoryIncludeInside_FR012` |
| Seed: two weeks of history in time order, one bin entry with 1 activity and 1 line item | `seed/seed_test.go` `TestSeed_HistoryAndRecycleBin_011` |

### Handler / contract tests (`internal/history/handler_test.go`, `apitest` with the seed)

`TestListActivityLog_Filters_Contract`, `TestListActivityLog_InvalidParams_400`,
`TestActivityLogSummary_Contract`, `TestListRecycleBin_Items_Contract`,
`TestRestorePreview_Contract` (+ 404, 409), `TestRestore_WithParents_Contract` (+ 409 both codes),
`TestPurge_Contract` (+ 404, 409), `TestEmptyRecycleBin_Contract` (+ 400), and 001's
`TestRoutesMatchContract` / `TestEveryProtectedOperationRequiresSignIn_FR005` cover the new routes.

### Web component tests (Vitest)

| File | Covers |
|---|---|
| `ActivityLogPage.test.tsx` | FR-001 header; FR-002 cards and their links; FR-003 filters → API query, chips, Clear all, counts; FR-004 day groups; FR-009 Load more; FR-010 no edit controls; loading, error, both empty states |
| `LogEntry.test.tsx` | FR-005 sentence, ID, path, time, chip word and tone per action; FR-006 verbs (incl. emptied bin); FR-007 details (changes, comment, delete details, items); FR-008 links per state, "now", "(permanently deleted)" |
| `logFilters.test.ts` · `logText.test.ts` | URL ⇄ query round trip; local date ranges (Monday week start, 30 days, year, custom exclusive end; fake timers in UTC and a non-UTC offset via mocked `Date`); day labels |
| `HistorySection.test.tsx` | FR-011 list and Load more (20); FR-012 toggle labels per level, saves the preference; FR-013 Open in activity log URL; empty and error |
| `RecycleBinPage.test.tsx` | FR-014 header, count line, Empty button only when not empty; FR-015 rows; FR-016 search in contents + pills + count; FR-017 state lines; `?record=` highlight and miss; empty states |
| `RestoreDialog.test.tsx` | FR-018 destination and sentences; FR-019/FR-020 status sentences; FR-022 name sentence; FR-023 Restore both / all N and `withParents=true`; FR-024 blocked dialog → purge; FR-026 toast and invalidation; FR-032 already resolved |
| `PurgeDialog.test.tsx` · `EmptyBinDialog.test.tsx` | FR-028 texts, focus on Cancel, inside-entry warning; FR-029 counts, entry IDs sent; results and FR-032 |
| `binText.test.ts` · `errors.test.ts` · `format.test.ts` · `navigation.test.ts` · `router.test.tsx` | sentences and contents text; new error kind and resolved text; weekday date; sections leave Coming soon |

### End to end (Playwright, `frontend/e2e/`)

| File | Journey |
|---|---|
| `recycle-bin.spec.ts` | US1: the seeded initiative row, search by its activity's ID ("Contains …"), Restore → dialog text → toast → plan page shows it, activity Created; delete an approved activity, wait for the Undo message to close, restore from the bin, its status is Created. US4: delete an activity then its initiative → row text → Restore both. US5: Delete permanently with texts; Empty recycle bin; log shows "emptied the recycle bin". axe on the page and each dialog |
| `activity-log.spec.ts` | US2: cards, filter person + action, chips and count, expand an entry, link to a record and to the bin. US3: change an activity twice and a line item once, open its History with and without Include line items, Open in activity log. axe on the page and the History section; keyboard-only filter and expand |

## Requirement traceability

| FR | Implemented in | Tested by |
|---|---|---|
| FR-001 | `ActivityLogPage`, router, navigation | `ActivityLogPage.test.tsx`, e2e |
| FR-002 | `LogSummary`, summary endpoint, cards | `TestLogSummary_*`, page test |
| FR-003 | `QueryLog`, `LogFilters`, `logFilters.ts` | `TestQueryLog_Filter*`, page and filter tests, e2e |
| FR-004 – FR-008 | `QueryLog` (order, RecordNow), `LogEntryList`, `LogEntry`, `logText` | query tests, `LogEntry.test.tsx`, `logText.test.ts` |
| FR-009 | `beforeSeq`, `useInfiniteQuery` | `TestQueryLog_BeforeSeqPaging_FR009`, page test |
| FR-010 | no write route for entries | `TestLog_EntriesNeverChangeOrDisappear_FR010_FR030`, route test, page test |
| FR-011 – FR-013 | `HistorySection`, `showHistory`, scope matching, preference | `TestQueryLog_RecordWithItemsInside_*`, people test, `HistorySection.test.tsx`, e2e |
| FR-014 – FR-017 | `ListBinItems`, `RecycleBinPage`, `BinRow` | bin tests, page tests, e2e |
| FR-018 – FR-022 | `PreviewRestore`, `restoreEntry`, kinds' hooks, `RestoreDialog`, `binText` | preview and restore tests, dialog tests, e2e |
| FR-023, FR-024 | `RestoreWithParents`, refined `checkCanComeBack`, dialog | `TestRestoreWithParents_*`, `TestRestore_ParentPurged_*`, dialog test, e2e |
| FR-025 – FR-027 | `restoreEntry`, unchanged Undo | `TestRestore_NotBlocked…`, `TestRestore_Logs…`, `TestUndo_StillExact_FR027` |
| FR-028 – FR-030 | `Purge`, `EmptyBin`, `appendEntry`, dialogs | purge/empty tests, dialog tests, e2e |
| FR-031 | no timer | `TestBin_NothingLeavesAutomatically_FR031` |
| FR-032 | `already_resolved` with `deletedPermanently`, web refetch | `TestActions_AlreadyResolved_FR032`, dialog tests |
| SC-006 | one pass over in-memory entries | `TestQueryLog_50000Entries_SC006` (generated entries; < 2 s with the race detector off) |
| SC-008 | accessible markup | e2e axe + keyboard journeys |

## Complexity Tracking

| Choice | Why needed | Simpler alternative rejected because |
|---|---|---|
| Dry-run restore for the preview (a `Write` that is rolled back) | SC-004: every status and name change must be stated before confirming, including hook effects owned by 004–008 | A per-kind "what would change" function duplicates every restore rule and drifts |
| A log collector in the context during the dry run | `AfterRestore` effects on records outside the entry (FR-020) are only visible through the entries the hooks write | Asking 007/008 for a second "preview" hook doubles their work |
| `HistorySection` imported across feature folders | Five record pages owned by 004–007 need the same data-loading section | Copying it five times; putting a data-loading part in `@/components` breaks that folder's rule |

## Consolidation (2026-10-19, binding)

The Plan-module consolidation (the cross-feature analyze step) settled the points below. They and the build guide,
[docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md), win where this plan, its data model or its tasks
say otherwise (D-200).

- **Contract (D-192).** The fragment is merged into `backend/api/openapi.yaml`; the merge task (T001) is done. The
  feature's operations are listed in `notYetServed` in `backend/internal/app/app_test.go`; the backend removes each one in
  the change that serves it. Agents do not change the contract; a problem with it is reported.
- **Decisions.** This plan's proposed decisions are numbered D-177–D-188 in `docs/sdd/decision-log.md` (labels kept in the
  table above); the consolidation decisions are D-189–D-207.

- **001 schemas extended (D-191).** In the merged contract the new properties of `ActivityLogEntry` (`items`, `recordNow`),
  `ActivityLogPage` (`unfilteredTotal`, `record`), `RecycleBinListItem` (beyond 001's `RecycleBinEntry`), `Preferences`
  (`historyIncludeInside`) are optional; 011 always sends them. `Resolution.how` gains `deletedPermanently` and `Error.code`
  `parent_deleted_permanently`. 001's web fixtures keep compiling, so T003 only adds the new fields where tests need them.
- **Hooks.** 007 adds `history.Kind.AfterRestore`; 006 calls `ApprovalHooks.ActivityRestored` after a restore (D-196); both log
  through `history.Log` so the dry-run preview sees them (D-177).
- **Shared parts (D-197).** 011 owns `HistorySection` (`features/activity-log/`), imported by the record pages of 004-007; it uses
  004's `recordHref` (D-194) and `FilterChips`, 002's `SearchField`, `SegmentedFilter`.
- **Seed (D-188, D-202).** `addHistory` runs last; 002-008 seed steps write no log entries.
