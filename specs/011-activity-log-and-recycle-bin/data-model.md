# Data Model: Activity Log and Recycle Bin

**Feature**: `011-activity-log-and-recycle-bin` · **Plan**: [plan.md](plan.md) · **Builds on**:
[001 data model §5–§7](../001-app-shell-and-sign-in/data-model.md)

This feature adds **no table and no ID prefix**. It extends three stored records of feature 001 and
derives four read models. JSON names are the names in files and in the API. Conventions (timestamps,
`ActorRef`, `RecordRef`, text lengths) are 001's.

---

## 1. Activity-log entry — append-only table `activity_log` (extended)

All fields of 001 data model §5 stay as they are. Entries are still only appended (FR-010, 001 FR-023).

| Change | Field / value | Rule |
|---|---|---|
| New action | `deletedPermanently` | One entry per **Delete permanently** (FR-030). Verb `permanently deleted`. `recordType`, `recordTypeLabel`, `recordId`, `recordName` and `path` are the recycle-bin entry's (the path at the time of the delete — the record is no longer stored). `details` = what was inside ("with 2 activities and 5 line items", "" when nothing). `changes` `[]` |
| New action | `emptiedRecycleBin` | One entry per **Empty recycle bin** that purged at least one entry (FR-029, FR-030). Verb `emptied the recycle bin`; `recordType` `recycleBin`; `recordTypeLabel` `recycle bin`; `recordId` `""`; `recordName` `Recycle bin`; `path` `[]`; `details` "3 items" / "1 item"; `items` = the purged entries |
| New field | `items` — list of **LogItem** | Only on `emptiedRecycleBin` entries; `[]` on every other entry. Entries stored before this feature have no `items`; the API serves them with `[]` |

**LogItem**: `{recordType, recordTypeLabel, recordId, recordName, details}` — one purged recycle-bin entry;
`details` as for `deletedPermanently`.

**Action filter values** (FR-003): `created`, `changed`, `statusChanged`, `moved`, `copied`, `deleted`,
`restored`, `deletedPermanently`. The value `deletedPermanently` matches both stored actions
`deletedPermanently` and `emptiedRecycleBin` (both show the chip **Permanently deleted**).

**Action chip** (FR-005, design system §4):

| Stored action | Word | Tone (`StatusChip`) |
|---|---|---|
| `created` | Created | positive (green) |
| `changed`, `moved`, `copied` | Changed, Moved, Copied | neutral (grey) |
| `statusChanged`, `restored` | Status changed, Restored | active (blue) |
| `deleted` | Deleted | problem (red) |
| `deletedPermanently`, `emptiedRecycleBin` | Permanently deleted | problem (red) |

## 2. Recycle-bin entry — table `recycle_bin` (extended)

All fields of 001 data model §6 stay. `resolution.how` gains `deletedPermanently`.

Life cycle (adds the dotted transitions):

```
  record ──delete──► in the bin ──Undo (≤ 2 min, exact)──────────────────► resolution "undone"
                         │
                         ├──Restore (parent stored)─────────────────────────► resolution "restored"
                         ├┄┄Restore both / all N (parents in the bin first)┄► resolution "restored" (each entry)
                         ├┄┄Delete permanently┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄► resolution "deletedPermanently"
                         └┄┄Empty recycle bin (listed entries)┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄► resolution "deletedPermanently" (each entry)
```

- A resolved entry keeps `resolution {how, by, at}` and has `snapshots: []` — for a purge the records are
  then gone for good; their IDs are never given out again (counters only increase, 001 FR-027).
- Nothing changes an entry on a timer (FR-031).
- Any action on a resolved entry answers `409 already_resolved` with its `resolution` (FR-032).

## 3. Preferences — table `preferences` (extended)

| Field | Type | Default | Rule |
|---|---|---|---|
| `historyIncludeInside` | boolean | `false` | The person's choice for **Include items inside** / **Include line items** on every History section (FR-012). Display preference: no version, not logged (D-038). Any JSON boolean is valid; another type is refused by `web.ReadJSON` (`bad_request`) |

(Feature 003 adds `displayCurrency` and `dateFormat`, feature 009 adds `homeScope`; 001 has `theme` and
`sidebarCollapsed`.)

## 4. Derived: record state (`recordNow`, not stored)

Computed for every entry the API returns (FR-008).

| `state` | When | `name` | `path` | `binEntryId` |
|---|---|---|---|---|
| `exists` | The record's kind is registered and its `Snapshot` finds it | current name | current ancestors, top first | `""` |
| `inRecycleBin` | Not stored; an unresolved bin entry holds a snapshot with its ID | name in the bin | `[]` | that entry |
| `deletedPermanently` | Registered kind, not stored, in no unresolved bin entry | `""` | `[]` | `""` |
| `none` | `emptiedRecycleBin` entry, or the kind is not registered here | `""` | `[]` | `""` |

The web app shows " — now '<name>'" when `state` is `exists` and `name` differs from `recordName`.

## 5. Derived: restore state of a bin entry (not stored)

| `restoreState` | Rule | `missingParent` | `parentEntryIds` |
|---|---|---|---|
| `restorable` | No parent, or the parent is stored | `null` | `[]` |
| `parentInRecycleBin` | The parent is in another unresolved entry, and following parents through unresolved entries reaches a stored record or the top | the direct parent (`{type, typeLabel, id, name}`) | the entries to restore first, top-most first |
| `parentDeletedPermanently` | Following parents reaches a record that is neither stored nor in an unresolved entry | that record | `[]` |

- `insideEntryIds`: the unresolved entries in state `parentInRecycleBin` whose `parentEntryIds` contain this
  entry — they can no longer be restored once this one is purged (FR-028).
- `records`: the snapshots without their data — `{recordType, recordTypeLabel, recordId, recordName,
  parentId}` — the record first, then everything inside it, parents before children (contents tree and
  search, FR-015, FR-016).
- `recordTypePlural`: the kind's plural ("activities") for the type pills (FR-016).
- Entries holding a kind that is not registered here are not listed (plan D-186).

## 6. Derived: restore preview (not stored)

The result of a rolled-back dry run of the restore (plan D-177):

| Field | Meaning |
|---|---|
| `entry` | The chosen entry as in §5 |
| `destination` | The parent's **current** path including the parent (after parents in the bin came back); `[]` for top-level records |
| `parentEntries` | The entries **Restore both / all N** brings back first, top-most first |
| `changes` | One row per changed field per record: `{recordType, recordTypeLabel, recordId, recordName, restored, field, label, oldText, newText}`. `restored: true` for records that come back (status reset, name suffix); `false` for existing records the restore changes (an activity sent back to Created, FR-020). Bookkeeping fields never appear |
| `notes` | The plain sentences the kinds' hooks returned, in order |

For `parentDeletedPermanently`, the preview has no dry run: `destination`, `parentEntries`, `changes` and
`notes` are empty and the dialog explains why (FR-024).

## 7. Derived: activity-log summary (not stored)

Computed over the **whole** log (FR-002), for boundaries the browser sends in UTC:

| Field | Rule |
|---|---|
| `changesToday` | Entries with `at ≥ todayStart` |
| `changesThisWeek` | Entries with `at ≥ weekStart` (local Monday 00:00) |
| `peopleActiveThisWeek` | Distinct `actor.id` among this week's entries |
| `deletionsThisWeek` | This week's entries with action `deleted`, `deletedPermanently` or `emptiedRecycleBin` |
| `total` | All entries |
| `people` | Every actor with at least one entry, sorted by name (Person filter options) |
| `recordTypes` | `{type, label, count}` for each record type present, except `recycleBin`, label = latest `recordTypeLabel`, sorted by label |
| `plans` | Every plan named in the log — entries whose `recordType` is `plan`, and `path[0]` when its type is `plan` — with its latest name, sorted by name (Plan filter options) |

## 8. Scope matching (Plan filter, Include items inside)

For chosen record IDs `S`: walk entries in `seq` order and add the `recordId` of every `moved` entry whose
`path` contains an ID in `S`; an entry then matches when its `recordId` is in `S` or its `path` contains an
ID in `S`. This covers records that **are** inside (their entries carry the path, or they moved in) and
records that **were** inside (deleted and permanently deleted ones keep their entries' paths).

## 9. Demo data added by this feature

| Record | Values |
|---|---|
| Activity-log entries | About two weeks before the seed time, in time order: `created` for every seeded plan, campaign, initiative, activity and line item (by its owner); `statusChanged` `published` for each Published plan; one `statusChanged` per approval-history record of 008's seed (verb, actor, time, comment); one `changed` planned cost by Priya Raman (−5,000.00 USD before) |
| Deleted initiative | "Trade shows" (`I-…`) under FY2026 Lubricants Europe › Rimula fleet, owner Priya Raman, budget 20,000.00 USD; with activity "Fleet expo stand" (`A-…`, Event, Created, 18,000.00 USD) and line item "Stand build" (`L-…`, 18,000.00 USD). One bin entry (`BIN-…`) deleted by Priya Raman 2 days before the seed time, contents "1 activity" and "1 line item", plus its `created` and `deleted` log entries. The three records are not in their tables |
