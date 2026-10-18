# Data Model: Application Shell and Demo Sign-in

**Feature**: `001-app-shell-and-sign-in` · **Plan**: [plan.md](plan.md) · **Research**: [research.md](research.md)

The records this feature stores, their fields, rules and life cycles. Records are **relational in shape**
(one table per record type, ids, parent ids, no nesting) so that PostgreSQL can replace the JSON store
later (ADR-0004). In the JSON store a normal table is one file `backend/data/<table>.json` (an object
whose keys are record ids); the activity log is an **append-only** table `backend/data/activity_log.jsonl`
(one entry per line, research R3). JSON field names below are the names used in files and in the API.

Conventions used by every record type in NorthStar:

- **Timestamps** are UTC, RFC 3339; fractional seconds have up to 3 digits and trailing zeros are dropped
  (`2026-10-18T09:02:00.123Z`, `2026-10-18T09:02:00Z`). Never compare or sort them as text.
- **Text lengths** count Unicode characters after trimming surrounding spaces.
- **Actor reference** (`ActorRef`): `{"id": "U-1002", "name": "Priya Raman"}` — the name is stored as it
  was at the time, so history stays true.
- **Record reference** (`RecordRef`): `{"type": "sampleFolder", "id": "SF-1001", "name": "Ideas"}`.
- **Bookkeeping** (`record.Meta`, embedded in every business record and flattened into its JSON):

| Field | Type | Rule |
|---|---|---|
| `version` | integer ≥ 1 | 1 on create; +1 on every saved change; every update request must send the version it started from (FR-029) |
| `createdAt` | timestamp | Set once |
| `createdBy` | ActorRef | Set once |
| `updatedAt` | timestamp | Equals `createdAt` until the first change |
| `updatedBy` | ActorRef | Equals `createdBy` until the first change |

Bookkeeping fields are never listed as "changed fields" in the activity log.

---

## 1. Person — table `people`

Someone who can sign in. Feature 003 extends it with teams and profile fields; the rules below are
feature 003's (FR-002, FR-009), so both features agree.

| Field | Type | Rules |
|---|---|---|
| `id` | string | `U-1001`, `U-1002`, … (readable-ID counter `U`); never changes |
| `name` | string | Full name, 1–120 characters |
| `email` | string | At most 254 characters; unique ignoring case; stored in lower case |
| `jobTitle` | string | 0–120 characters |
| `active` | boolean | Only active people appear on the sign-in page, can sign in and keep their sessions |
| *bookkeeping* | `record.Meta` | Seed records are created by the system actor `{"id": "system", "name": "NorthStar"}` |

Derived (computed by the API, never stored):

- `firstName` = the first word of `name`.
- `initials` = the first letters of the first two words of `name`; for a one-word name, its first two
  letters; upper case ("Priya Raman" → "PR", "Mary Ann Smith" → "MA", "Cher" → "CH").

Seed (`backend/seed`, in the order of the demo-users table): Alex Morgan (`U-1001`), Priya Raman
(`U-1002`), Daniel Okoro (`U-1003`), Mei Lin (`U-1004`), Carlos Mendes (`U-1005`), Sara Novak
(`U-1006`), with the e-mails and job titles of
[demo-users-and-data.md](../../docs/product/demo-users-and-data.md); all active.

## 2. Preferences — table `preferences`

One record per person who changed a preference; the key is the person's id. A person without a record
uses the defaults. Not a business record: changes are not logged and have no version (decision D-038).

| Field | Type | Rules |
|---|---|---|
| `personId` | string | Key; an existing person |
| `theme` | enum | `system` (default — follow the computer), `light`, `dark` |
| `sidebarCollapsed` | boolean | Default `false` |

Validation: a `theme` other than the three values → field error `theme`: "Choose system, light or
dark." The API returns preferences as `PreferencesView` (`theme`, `sidebarCollapsed`, without
`personId`). Feature 003 adds date format and display currency here.

## 3. Session — table `sessions`

A signed-in person in one browser (key entity "Session").

| Field | Type | Rules |
|---|---|---|
| `id` | string | Hex SHA-256 of the session token; the token itself is never stored |
| `personId` | string | The signed-in person |
| `createdAt` | timestamp | When the person signed in |
| `lastUsedAt` | timestamp | Last request with this session; rewritten when it is at least 1 minute old |

Rules (FR-006):

- A session **expires** when `now − lastUsedAt > 12 h 1 min`. Because `lastUsedAt` lags the real last
  request by at most 1 minute, no session ends before 12 full hours without a request.
- An expired session is deleted and the request answers `401 session_expired`.
- When the person no longer exists or is inactive, the session is deleted and the request answers
  `401 sign_in_required` (feature 003 relies on this: "open sessions end at their next action").
- Several sessions per person are allowed (two browsers work independently — edge case).
- A sign-in deletes the browser's previous session (when its cookie is sent) and every expired session,
  of any person.

```
             sign in                     request at most 12 h 1 min after lastUsedAt
  (none) ──────────────► active ─────────────────────────────────────────────┐
                           │ ▲                                                │
                           │ └──────────── lastUsedAt moves forward ◄────────┘
             sign out      │      later request, or person gone / inactive
  deleted ◄────────────────┴─────────────────────────────► deleted (session_expired / sign_in_required)
```

## 4. Readable-ID counter — table `counters`

| Field | Type | Rules |
|---|---|---|
| `prefix` | string | Key: `P`, `C`, `I`, `A`, `L`, `U`, `LOG`, `BIN`, `SF`, `SN`, … |
| `last` | integer | Last number given; the first ID of a prefix is 1001; only ever increases |

Two IDs taken in one transaction are different (the transaction reads its own writes, research R3).
Services reach the counters only through `ids.Generator`, which uses the `ids.Counters` repository
interface (JSON store now, a PostgreSQL implementation later).

## 5. Activity-log entry — append-only table `activity_log` (`activity_log.jsonl`)

One entry per business change (FR-022, SC-004). Entries are only ever appended: nothing in NorthStar
updates or deletes them (FR-023). They carry everything feature 011 shows, because entries written by
features 002–010 cannot be completed later (011 Key Entities and Assumptions).

| Field | Type | Rules |
|---|---|---|
| `id` | string | `LOG-1001`, … |
| `seq` | integer | The number of the id; orders entries, also those written in the same millisecond |
| `at` | timestamp | When the change was saved |
| `actor` | ActorRef | Who did it |
| `action` | enum | `created`, `changed`, `statusChanged`, `moved`, `copied`, `deleted`, `restored` (feature 011 adds `deletedPermanently` and `emptiedRecycleBin`) — used for filters and the action chip |
| `verb` | string | The plain verb for the sentence "<person> <verb> <record type> '<name>'": `created`, `changed`, `moved`, `copied`, `deleted`, `restored` by default; a status change names its own (`published`, `submitted`, `approved`, …) |
| `recordType` | string | Kind name, e.g. `sampleFolder` (later `plan`, `campaign`, …) |
| `recordTypeLabel` | string | The kind's mid-sentence name at the time, e.g. `sample folder`, `Shell attribute` (capitalise the first letter at the start of a sentence) |
| `recordId` | string | Readable ID (planning and sample records) or internal id (Settings records) |
| `recordName` | string | The record's name at the time |
| `path` | list of RecordRef | Where the record was at the time: its ancestors, top level first (`[]` for top-level records). The plan a record belongs to is the first element when its type is `plan` |
| `changes` | list of Change | `changed` and `statusChanged`: every changed field; `moved`: field `parent` with the old and new place; `restored`: the changes the restore made (a status reset, a name suffix); otherwise empty |
| `comment` | string | A reason the person gave (for example a decline reason, feature 008); empty when none |
| `details` | string | Plain text, empty when there is nothing to add. Deletes: "with 2 sample notes". Restores: "Undo" or "Restored from the recycle bin", followed by ", with 2 sample notes" when it had contents |

**Change**: `{field, label, old, new, oldText, newText}` — `field` is the JSON field name; `old`/`new`
the stored JSON values; `label` and `oldText`/`newText` are what people read, fixed at the time:

- `label`: from the kind's `Describe` hook, otherwise the field name in words (`plannedCost` →
  "Planned cost");
- `oldText`/`newText`: from `Describe` (option labels, people's names, money with currency, dates as
  `DD MMM YYYY`), otherwise a default: text as it is, `true`/`false` → "Yes"/"No", numbers as written,
  lists joined with ", ", empty or missing → "Not filled in".

## 6. Recycle-bin entry — table `recycle_bin`

A deleted record and everything that was under it (FR-024).

| Field | Type | Rules |
|---|---|---|
| `id` | string | `BIN-1001`, … |
| `recordType`, `recordTypeLabel`, `recordId`, `recordName` | strings | The record the person deleted (`recordTypeLabel` as in §5) |
| `path` | list of RecordRef | Its ancestors at the time of the delete, top level first (011: "Was in <path>") |
| `parent` | RecordRef or null | Its parent at the time of the delete (the last element of `path`; null for top-level records) |
| `deletedAt` | timestamp | |
| `deletedBy` | ActorRef | |
| `contents` | list of `{recordType, label, count}` | Records under it by kind, e.g. `{"recordType": "sampleNote", "label": "sample notes", "count": 2}`; `label` is the singular when `count` is 1 |
| `snapshots` | list of Snapshot | The record first, then its descendants, parents before children; removed when the entry is resolved |
| `resolution` | `{how, by, at}` or null | null while in the bin; `how` = `undone` or `restored` (011 adds `deletedPermanently`); `by` ActorRef; `at` timestamp |

**Snapshot**: `{"kind": "sampleNote", "id": "SN-1003", "name": "Ideas", "parentId": "SF-1001", "data": {…}}`
— `data` is the record exactly as it was stored.

Life cycle:

```
  record ──delete──► in the bin ──Undo (≤ 2 min)─────────────────────► resolution "undone"   (records back)
                         │
                         ├──Restore (parent must exist)───────────────► resolution "restored" (records back)
                         │
                         └──Delete permanently (feature 011)──────────► resolution "deletedPermanently"
```

- **Undo** puts every record back exactly as it was (same IDs, data, versions and statuses — domain model
  R14). When someone has meanwhile taken the name (a kind's name rule), Undo is refused with
  `name_taken` and nothing changes; the item stays in the bin (spec edge case, D-037).
- **Restore** puts every record back and applies each kind's restore rules in `PrepareRestore`: a
  taken name gets the suffix " (restored)" (then " (restored 2)", …, as feature 011 FR-022), plans go
  back to Draft and activities to Created (features 004 and 006); changed records get a new version.
- Both refuse, and change nothing, when: the parent is no longer stored → `restore_parent_first`
  "Restore '<parent name>' first"; the entry is already resolved → `already_resolved` (who and when);
  the entry's kind is not registered here → `not_found`; and, for Undo only, more than 2 minutes have
  passed → `undo_expired`.
- A successful Undo or Restore writes one `restored` log entry, sets `resolution` and removes
  `snapshots`. The recycle-bin list shows unresolved entries only, newest first. The summary returned by
  the API (`RecycleBinEntry`) is this record without `snapshots` and `resolution`.

## 7. Record kinds (registry, not stored)

Each business package registers a `history.Kind` for every record type it owns, once, in
`internal/app` (research R8).

| Part | Meaning | Sample folder | Sample note |
|---|---|---|---|
| `Name` | Code stored in log and bin entries | `sampleFolder` | `sampleNote` |
| `Singular` / `Plural` | Mid-sentence names | `sample folder` / `sample folders` | `sample note` / `sample notes` |
| `Parent` | Name of the parent kind | — | `sampleFolder` |
| `Snapshot(ctx, id)` | Read one record as a Snapshot (with its name and parent id) | ✓ | ✓ |
| `ChildIDs(ctx, parentID)` | Ids of this kind under the parent (required when `Parent` is set) | — | notes whose `folderId` = parent |
| `Remove(ctx, id)` | Delete from storage | ✓ | ✓ |
| `Put(ctx, snapshot)` | Write a snapshot back | ✓ | ✓ |
| `Describe` (optional) | Field label and readable text for the log | — (defaults suffice) | — |
| `PrepareRestore` (optional) | Check or adjust a snapshot before it is put back; receives the actor, the time and whether it is an Undo; may refuse with an error (Undo of a taken name) | Undo: refuse when the name is taken; Restore: name suffix | — |

Later features register `plan` (004), `campaign` and `initiative` (005), `activity` (006), `lineItem`
(007), settings kinds (002) and people/teams (003). Hooks that only later features need are added then
as further optional fields, without changing existing registrations: `AfterRestore` (007/008: an
activity whose line items come back returns to Created), `BeforeDelete` (008: end an open approval
round) and a restore preview (011).

## 8. Sample folder — table `sample_folders` (developer page only)

The demonstration record type of User Story 5 (local and development environments only, D-036).

| Field | Type | Rules |
|---|---|---|
| `id` | string | `SF-1001`, … |
| `name` | string | **Required** ("Enter a name."); trimmed; at most 80 characters ("Use 80 characters or fewer."); unique among sample folders ignoring case and surrounding spaces ("There is already a sample folder called '<name>'. Choose another name."). A taken name refuses an Undo (`name_taken`); on Restore it gets " (restored)", " (restored 2)", … |
| *bookkeeping* | `record.Meta` | |

## 9. Sample note — table `sample_notes` (developer page only)

| Field | Type | Rules |
|---|---|---|
| `id` | string | `SN-1001`, … |
| `folderId` | string | An existing sample folder; never changes |
| `title` | string | **Required** ("Enter a title."); trimmed; at most 120 characters ("Use 120 characters or fewer.") |
| *bookkeeping* | `record.Meta` | |

Activity-log behaviour of samples (the pattern every later feature follows):

| Person's action | Log entries written |
|---|---|
| Create a folder | 1 × `created` |
| Rename a folder (version matches) | 1 × `changed` with `changes: [{field: "name", label: "Name", old, new, oldText, newText}]` |
| Save without any change | none (no new version) |
| Save from an out-of-date copy | none (refused with `version_conflict`) |
| Add a note | 1 × `created` (record type `sampleNote`, path `[the folder]`) |
| Delete a folder with 2 notes | 1 × `deleted`, details "with 2 sample notes"; one bin entry with 3 snapshots |
| Undo that delete | 1 × `restored`, details "Undo, with 2 sample notes" |

## 10. Web-app state (not stored on the server)

| Item | Where | Why |
|---|---|---|
| Last theme (`light` / `dark` / `system`) | `localStorage["northstar.theme"]` | Paint the right colours before the API answers; the theme switch on the sign-in and public "Page not found" pages stores its choice only here; after sign-in the person's saved preference takes over |
| Cached API data | TanStack Query cache (memory only) | Cleared on sign-out, when the session is lost and when another person signs in in another tab |
