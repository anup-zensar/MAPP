# Data Model: SWBS Codes from a Simulated GSAP

## 1. Line item — SWBS fields (table `line_items`, package `lineitem`)

| Field (JSON) | Column | Type | Rule |
|---|---|---|---|
| `requestSwbs` | `request_swbs` | boolean | Existing (feature 007). Cannot become false while `swbsCode` is set (FR-004). |
| `swbsCode` | `swbs_code` | text, `''` until received | Set only by a run (FR-002); unique among line items; never removed (FR-003). Format `SIM-<cc>-<nnnn>` now. |
| `swbsRequestedAt` | `swbs_requested_at` | timestamptz, null until requested | The run's time. |
| `swbsReceivedAt` | `swbs_received_at` | timestamptz, null until received | The run's time; set exactly when `swbsCode` is set. |

Constraints (migration `00017_swbs.sql`):

- `(swbs_code = '') = (swbs_received_at IS NULL)`;
- `swbs_received_at IS NULL OR (swbs_requested_at IS NOT NULL AND swbs_requested_at <= swbs_received_at)`;
- `swbs_code = '' OR request_swbs`;
- unique `swbs_code` where it is not empty;
- `swbs_code` is empty or 1–40 characters of `A–Z`, `0–9` and `-`.

Derived, never stored:

- **SWBS status** (`SwbsStatus`): `received` when `swbsCode` is set; else `notRequested` when
  `requestSwbs` is false; else `readyToSend` when the activity's stored status is `approved`; else
  `waitingForApproval`.
- **HX key** (`hxKey`): `"HX"` + the digits of the ID (`L-1006` → `HX1006`).

Copies (duplicate, activity copy, plan copy) start with empty SWBS fields. Delete, restore, move and
edits keep them. Receiving a code touches the line item's bookkeeping (version + 1, updated by the person
who ran it, at the run's time).

## 2. SWBS run (tables `swbs_runs`, `swbs_run_codes`; package `swbs`)

`Run` embeds `record.Meta` (version 1, created = updated = the run).

| Field | Column | Type | Rule |
|---|---|---|---|
| `id` | `swbs_runs.id` | text `SWR-<n>` | From counter `SWR`. |
| `source` | `swbs_runs.source` | `simulatedGsap` · `demoData` | `demoData` only for the seed's run. |
| `createdAt` / `createdBy` | `created_at`, `created_by_id`, `created_by_name` | | When and by whom (MAPP for the demo data). |
| `version`, `updatedAt`, `updatedBy` | same names | | Never change: runs are only added. |
| `codes[]` | `swbs_run_codes` (`run_id`, `position`) | | In send order (line-item ID order). |
| `codes[].lineItemId` | `line_item_id` | text | No foreign key: the run records what was exchanged even if the line item is later purged. |
| `codes[].lineItemName` | `line_item_name` | text | The name at the time of the run. |
| `codes[].companyCode` | `company_code` | text, `''` when none | The activity's company code at the time. |
| `codes[].swbsCode` | `swbs_code` | text, unique | |

History kind `swbsRun` (top level, "SWBS run" / "SWBS runs"): one `created` entry per run, verb "ran",
record name "Simulated GSAP" (or "Demo data (simulated GSAP)"), details listing the codes. The seed's run
writes no entry (D-188).

## 3. API shapes

- `SwbsRunRequest`: `{ lineItemIds?: string[] }` — 1 to 200 IDs, each 1–40 characters; left out = every
  ready line item.
- `SwbsRun`: `{ id, source, ranAt, ranBy, sent, codes: SwbsRunCode[] }`.
- `SwbsRunCode`: `{ lineItemId, lineItemName, companyCode, swbsCode }`.
- `SwbsRunPage`: `{ items: SwbsRun[], total, readyToSend }` — newest first, `limit` 1–100 (default 20).
- `LineItem` gains `swbsCode`, `swbsRequestedAt`, `swbsReceivedAt`, `hxKey`; `SwbsStatus` gains `received`.

## 4. Ports and adapter (Go)

```go
// package swbs
type Request struct{ LineItemID, LineItemName, CompanyCode string }
type Answer struct{ LineItemID, SwbsCode string }
type GSAP interface {
	RequestCodes(ctx context.Context, requests []Request) ([]Answer, error)
}
type Issued interface {
	CountCodes(ctx context.Context, prefix string) (int, error) // codes given that start with prefix
}
type Candidate struct {
	Request
	Ready bool
}
type Receiver interface {
	Receive(ctx context.Context, a actor.Actor, answers []Answer, at time.Time) error
}
type LineItems interface {
	Candidates(ctx context.Context, scope authz.Scope, ids []string) ([]Candidate, error)
	Receiver
}

// package lineitem
func SwbsStatusOf(li LineItem, f ActivityFacts) SwbsStatus
func (li LineItem) HXKey() string
type SwbsReceipt struct {
	By    actor.Actor
	At    time.Time
	Codes []SwbsCodeReceived
}
func ReceiveSwbsCodes(ctx context.Context, repo Store, receipt SwbsReceipt) error
```

`Candidates` returns every live line item of the activities the scope reaches (only those with `ids`
when `ids` is not nil), each with whether its status is `readyToSend`. `Receive` stores every answer
inside the caller's Write and fails, saving nothing, for a line item that is gone or already has a code.

## 5. Web app view models

- `features/simulated-gsap`: `SwbsRunPage`, `SwbsRun`, `SwbsRunCode` (generated types); query key
  `['simulated-gsap', 'swbs-runs']`; the panel's local state: dialog open, last result.
- `features/line-items`: `LineItem` (generated) with the new fields; `SwbsCard` props `{ lineItem,
  canSend }`; `HxKey` props `{ value }`.

## 6. Demo data

| Line item | Activity (status, company code) | SWBS status after the seed | Code |
|---|---|---|---|
| L-1001 Agency fee, L-1002 Film production | Launch film (Created) | Waiting for activity approval | — |
| L-1003 Venue hire | Social always-on (Approved, DE03) | SWBS code received | `SIM-DE03-0001` (run SWR-1001) |
| L-1004 Speaker fees | Social always-on (request off) | Not requested | — |
| L-1005 Media booking | Trade magazine ads (Submitted, NL02) | Waiting for activity approval | — |
| L-1006 Stand build | Billboard winter wave (Approved, GB01) | Ready to send to GSAP | — (first run: `SIM-GB01-0001`) |
| L-1007 – L-1010 | Created activities | Waiting for activity approval | — |
