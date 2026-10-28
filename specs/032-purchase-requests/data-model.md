# Data Model: Purchase Requests on MAPP's Side (032)

Money is integer minor units plus an ISO currency (constitution VII); calendar dates are `date`; timestamps
are UTC `timestamptz`. Readable IDs: `PR-1001` (purchase request), `PRL-1001` (line), `PRR-1001` (routing),
added to `platform/ids` as `PrefixPurchaseRequest = "PR"`, `PrefixPurchaseRequestLine = "PRL"`,
`PrefixPurchaseRequestRouting = "PRR"`.

## 1. Tables — migration `backend/internal/platform/database/migrations/00020_purchase_requests.sql`

### 1.1 `purchase_requests`

| Column | Type | Rule |
|---|---|---|
| id | text PK | `^PR-[0-9]+$` |
| plan_id | text NOT NULL | REFERENCES plans (id) DEFERRABLE INITIALLY DEFERRED; the plan of every line |
| name | text NOT NULL | 1–120 characters |
| description | text NOT NULL | ≤ 2,000 characters |
| owner_id | text NOT NULL | REFERENCES people (id) DEFERRABLE |
| moa_approver_id | text | REFERENCES people (id) DEFERRABLE; NULL until chosen |
| approval_workflow_id | text NOT NULL | soft reference (workflows can be deleted, as on activities) |
| modification_workflow_id | text NOT NULL | soft reference |
| status | text NOT NULL | IN ('created','submitted','inReview','declined','approved','closed') |
| change_pending | boolean NOT NULL | CHECK (NOT change_pending OR status = 'inReview') |
| attributes | json NOT NULL | json_typeof = 'object' (PR-level Shell attribute values) |
| vendor | text NOT NULL | copied at every save from the attribute with role `vendor`; '' when empty |
| approved_version | json | NULL or object; CHECK (status NOT IN ('approved','closed') AND NOT change_pending OR approved_version IS NOT NULL) |
| gsap_pr_number | text NOT NULL | '' until GSAP answers |
| gsap_pr_date | date | CHECK ((gsap_pr_number = '') = (gsap_pr_date IS NULL)) |
| gsap_source | text NOT NULL | IN ('', 'simulatedGsap', 'gsap'); CHECK ((gsap_source = '') = (gsap_pr_number = '')) |
| closed_at | timestamptz | CHECK ((status = 'closed') = (closed_at IS NOT NULL)) |
| version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name | as every record | `record.Meta` |

Indexes: `(plan_id)`, `(status)`, `(lower(vendor COLLATE "C"), id)`.

### 1.2 `purchase_request_lines`

| Column | Type | Rule |
|---|---|---|
| id | text PK | `^PRL-[0-9]+$` |
| purchase_request_id | text NOT NULL | REFERENCES purchase_requests (id) DEFERRABLE INITIALLY DEFERRED |
| position | integer NOT NULL | ≥ 1; UNIQUE (purchase_request_id, position) DEFERRABLE |
| line_item_id | text NOT NULL | REFERENCES line_items (id) DEFERRABLE; UNIQUE (purchase_request_id, line_item_id) DEFERRABLE (FR-014) |
| currency | text NOT NULL | REFERENCES currencies (code) DEFERRABLE |
| amount_minor | bigint NOT NULL | ≥ 0 (0 only after close; the service refuses ≤ 0 otherwise) |
| delivery_date | date NOT NULL | — |
| overall_limit_minor | bigint | NULL or ≥ 0 |
| description | text NOT NULL | ≤ 2,000 |
| attributes | json NOT NULL | object (PR-line Shell attribute values) |
| sent_on | date | set at the line's first routing, never changed |
| po_number, po_line, po_line_status | text NOT NULL | '' until GSAP answers (033) |
| po_date | date | CHECK ((po_line = '') = (po_date IS NULL)) |
| po_value_minor | bigint | CHECK ((po_line = '') = (po_value_minor IS NULL)); CHECK (po_line = '' OR po_number <> '') |
| version … updated_by_name | | `record.Meta` |

Indexes: `(purchase_request_id)`, `(line_item_id)`, `(po_number)`.

### 1.3 `purchase_request_routings` (write-once)

| Column | Type | Rule |
|---|---|---|
| id | text PK | `^PRR-[0-9]+$` |
| purchase_request_id | text NOT NULL | REFERENCES purchase_requests (id) DEFERRABLE |
| type | text NOT NULL | IN ('original','modification','resend','close') |
| routed_at | timestamptz NOT NULL | — |
| routed_by_id, routed_by_name | text NOT NULL | the person whose action routed it |
| recipients | json NOT NULL | array of strings, ≥ 1 element |
| form | json NOT NULL | array of PR form rows (§2.5), ≥ 1 element |

Trigger `purchase_request_routings_write_once BEFORE UPDATE OR DELETE … EXECUTE FUNCTION refuse_change()`.
Index `(purchase_request_id, routed_at)`.

### 1.4 `purchase_request_configuration`

| Column | Type | Rule |
|---|---|---|
| id | text PK | CHECK (id = 'purchase-requests') — one row |
| purchase_support | boolean NOT NULL | — |
| lines_on_approved | text NOT NULL | IN ('notAllowed','withModificationApproval','withoutApproval') |
| version … updated_by_name | | `record.Meta` |

No row reads as the defaults (on; withModificationApproval); the seed writes the row.

### 1.5 Changes to existing tables (same migration)

Each check is dropped and re-created with the values that exist after migrations 00017–00019 **plus** these
(read the latest definition before writing it; never edit an older migration):

- `shell_attributes.level` gains `'purchaseRequest'`, `'purchaseRequestLine'`.
- `shell_attributes.role` gains `'vendor'`, `'vendorCode'`, `'vendorEmail'`, `'shortText'`, `'prType'`,
  `'emailGroup'`, `'prRecipients'`, `'companyCode'`, `'costCentre'`, `'glCode'` (030 may have added
  `'companyCode'` already; keep one).
- `roles.permissions` allowed list gains `'purchaseRequests.edit'`, `'purchaseRequests.delete'`,
  `'purchaseRequests.submit'`, `'purchaseRequests.list'`.

### 1.6 Scope function

```sql
-- authz's view of a purchase request in SQL (research R3)
CREATE FUNCTION purchase_request_in_scope(
    target text, sees_everything boolean, viewer_id text, views_plans boolean, viewer_team_ids text[],
    lists boolean, list_team_ids text[]) RETURNS boolean LANGUAGE sql STABLE AS $$
  SELECT sees_everything
      OR p.owner_id = viewer_id OR coalesce(p.moa_approver_id = viewer_id, false)
      OR plan_in_scope(p.plan_id, false, viewer_id, views_plans, viewer_team_ids)
      OR plan_in_permission_scope(p.plan_id, false, viewer_id, lists, list_team_ids)
      OR EXISTS (SELECT 1 FROM approval_rounds r, <approvers of r.levels as in activity_in_scope>
                 WHERE r.<subject kind column> IN ('purchaseRequest', 'modifiedPurchaseRequest')
                   AND r.<subject id column> = p.id AND <approver id> = viewer_id)
  FROM purchase_requests p WHERE p.id = target
$$;
```

The subject columns are the ones 031's migration 00018 gives `approval_rounds`.

## 2. Go types — `backend/internal/purchaserequest`

### 2.1 `model.go`

```go
type Status string // StatusCreated "created" · StatusSubmitted · StatusInReview "inReview" · StatusDeclined · StatusApproved · StatusClosed
type LinesOnApproved string // NotAllowed "notAllowed" · WithModificationApproval · WithoutApproval
type RoutingType string // RoutingOriginal "original" · RoutingModification · RoutingResend · RoutingClose

type PurchaseRequest struct {
	ID, PlanID, Name, Description string
	OwnerID, MoAApproverID        string // "" when no MoA approver
	WorkflowID, ModificationWorkflowID string
	Status        Status
	ChangePending bool                     // a modification round is open
	Attributes    settings.AttributeValues // never nil
	Vendor        string                   // copy of the Vendor attribute
	Approved      *Version                 // nil until the first approval
	GSAP          GSAPHeader
	ClosedAt      *time.Time
	Lines         []Line // in position order; at least one once saved
	record.Meta
}

type Line struct {
	ID, PurchaseRequestID string
	Position              int
	LineItemID, Currency  string
	Amount                money.Amount
	DeliveryDate          date.Date
	OverallLimit          *money.Amount
	Description           string
	Attributes            settings.AttributeValues
	SentOn                *date.Date
	PO                    PurchaseOrderLine // zero until GSAP answers
	record.Meta
}

type GSAPHeader struct { PRNumber string; PRDate *date.Date; Source string } // Source "", "simulatedGsap", "gsap"
type PurchaseOrderLine struct { Number, Line, Status string; Date *date.Date; Value *money.Amount }

type Version struct { // the approved version (§1.1 approved_version)
	Name, Description, OwnerID, MoAApproverID string
	Attributes settings.AttributeValues
	Lines      []VersionLine
	ApprovedAt time.Time
}
type VersionLine struct {
	ID, LineItemID, Currency string
	Amount       money.Amount
	DeliveryDate date.Date
	OverallLimit *money.Amount
	Description  string
	Attributes   settings.AttributeValues
}

type Routing struct {
	ID, PurchaseRequestID string
	Type       RoutingType
	RoutedAt   time.Time
	RoutedBy   actor.Actor
	Recipients []string
	Form       []FormRow
}

// FormRow is one row of the PR form, as routed (spec "Requisitioner Attachment").
type FormRow struct {
	PurchaseRequestID, PurchaseRequestName, LineID, HXKey, ShortText string
	Vendor, VendorCode, CostCentre, SwbsCode, GLCode            string
	Amount                                                       money.Amount
	Currency                                                     string
	DeliveryDate                                                 date.Date
	MoAApproverName, MoAApproverEmail                            string
	Changed                                                      bool
}

type Configuration struct { PurchaseSupport bool; LinesOnApproved LinesOnApproved; record.Meta }

// ViewScope is where the acting person sees purchase requests: their plans.view and purchaseRequests.list scopes.
type ViewScope struct { Plans, List authz.Scope }
```

`HXKey(lineItemID string) string` returns `"HX" + strings.TrimPrefix(lineItemID, "L-")`.

### 2.2 `store.go` — the Store the service needs

```go
type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
	Write(ctx context.Context, fn func(ctx context.Context) error) error
}

type Store interface {
	// Get returns the purchase request with its lines whoever asks (internal reads and writes after a scoped read).
	Get(ctx context.Context, id string) (PurchaseRequest, bool, error)
	// GetInScope returns it only when the scope sees it (scope-isolation test).
	GetInScope(ctx context.Context, scope ViewScope, id string) (PurchaseRequest, bool, error)
	// ListInScope returns the purchase requests among ids that the scope sees, ordered by id (engine inbox; isolation test).
	ListInScope(ctx context.Context, scope ViewScope, ids []string) ([]PurchaseRequest, error)
	// Page lists the purchase requests of the plans where list holds purchaseRequests.list (isolation test).
	Page(ctx context.Context, list authz.Scope, q ListQuery) (Page, error)
	// OnLineItems returns the purchase requests the scope sees that draw on any of lineItemIDs (isolation test).
	OnLineItems(ctx context.Context, scope ViewScope, lineItemIDs []string) ([]PurchaseRequest, error)
	// Claims returns every line on lineItemIDs, of every stored purchase request, with its PR's status:
	// what the balance rule counts (never shown as records).
	Claims(ctx context.Context, lineItemIDs []string) ([]Claim, error)
	// DrawnOn returns the purchase requests that draw on any of lineItemIDs, as references (delete guards).
	DrawnOn(ctx context.Context, lineItemIDs []string) ([]errs.RecordRef, error)
	Put(ctx context.Context, pr PurchaseRequest) error // header and the given lines (upsert)
	Delete(ctx context.Context, id string) error        // header only (history removes lines first)
	GetLine(ctx context.Context, id string) (Line, bool, error)
	LineIDsOf(ctx context.Context, purchaseRequestID string) ([]string, error)
	PutLine(ctx context.Context, l Line) error
	DeleteLine(ctx context.Context, id string) error
	AddRouting(ctx context.Context, r Routing) error
	Routings(ctx context.Context, purchaseRequestID string) ([]Routing, error) // newest first
	Routing(ctx context.Context, id string) (Routing, bool, error)
	Configuration(ctx context.Context) (Configuration, error) // the defaults when no row exists
	PutConfiguration(ctx context.Context, c Configuration) error
	All(ctx context.Context) ([]PurchaseRequest, error) // Settings "Used by" references only (unrestricted)
}

type Claim struct { LineID, PurchaseRequestID, LineItemID, Currency string; Amount money.Amount; Status Status }

type ListQuery struct {
	Search                   string
	Statuses                 []Status
	PRTypeOptionIDs          []string // with NotCompleted
	PRTypeAttributeID        string   // the attribute with role prType; "" when none
	Vendors                  []string // exact vendor texts, ignoring case; NotCompleted matches ''
	OwnerIDs                 []string
	CreatedFrom, CreatedTo   *date.Date
	WaitingForGSAP           bool
	Sort                     string // allowlist: name, id, status, amount, created (and "-" for descending)
	Page, PageSize           int    // PageSize 50, 100 or 200
}
```

`Page` holds the PRs of the page (with lines), the total count and, per vendor on the page, its number of
PRs and the USD amounts of all its matching PRs. Lists are never nil.

### 2.3 `ports.go` — what the service reads from other areas (implemented in `internal/app`)

```go
type Deps struct {
	LineItems  LineItems
	Plans      Plans
	People     People
	Currencies Currencies
	Attributes Attributes
	Workflows  Workflows
	Invoices   Invoices
}

type LineItems interface {
	// Facts returns the live line items among ids, keyed by id; one in the recycle bin is left out.
	Facts(ctx context.Context, ids []string) (map[string]LineItemFacts, error)
	// Search returns at most limit live line items the scope sees whose name, id or HX key contains text,
	// of planID when it is not "", ordered by id.
	Search(ctx context.Context, scope authz.Scope, text, planID string, limit int) ([]LineItemFacts, error)
	// OfActivity returns the live line items of a live activity, in their order, and false otherwise.
	OfActivity(ctx context.Context, activityID string) ([]LineItemFacts, bool, error)
}

type LineItemFacts struct {
	ID, Name, OwnerID, Currency      string
	PlannedCost                      money.Amount
	ActivityID, ActivityName         string
	ActivityStatus                   string // stored status: created, submitted, declined, approved
	PlanID                           string
	Path                             []errs.RecordRef // plan, campaign, initiative, activity
	SwbsCode, SwbsStatus             string           // 030: "" until received; notRequested … received
	CompanyCode, CostCentre, GLCode  string           // option labels through the attribute roles; "" when empty
	BusinessUnit                     []string         // the plan's Business unit option ids
}

type Plans interface {
	// Facts returns the plan's access facts, name and plan currency, and false when it is not live.
	Facts(ctx context.Context, planID string) (PlanFacts, bool, error)
}
type PlanFacts struct { Access authz.PlanFacts; Name, Currency string }

type People interface {
	Person(ctx context.Context, id string) (people.PersonRef, bool, error)
	Email(ctx context.Context, id string) (string, error)
	DisplayRateOf(ctx context.Context, personID string) (people.DisplayRate, error)
}

type Currencies interface {
	Currency(ctx context.Context, code string) (settings.Currency, bool, error)
	Rates(ctx context.Context) (money.Rates, error) // today's rates (D-303)
}

type Attributes interface {
	Check(ctx context.Context, level settings.Level, in settings.AttributeInput) (settings.AttributeResult, error)
	Missing(ctx context.Context, level settings.Level, values settings.AttributeValues, businessUnit []string) ([]settings.MissingField, error)
	Describe(ctx context.Context, field string, value any) (label, text string, err error)
	// Roles returns, per role among roles, the values' texts: a text value as typed, the labels of chosen
	// options; nil when empty or when no attribute has the role. AttributeID gives the attribute of a role.
	Roles(ctx context.Context, values settings.AttributeValues, roles []string) (map[string][]string, error)
	AttributeID(ctx context.Context, role string) (string, error)
}

type Workflows interface {
	// Workflow returns the approval workflow's name, whether it is active and its object type (031).
	Workflow(ctx context.Context, id string) (WorkflowFacts, bool, error)
}
type WorkflowFacts struct { ID, Name, ObjectType string; Active bool }

type Invoices interface {
	// InvoicedOn returns, per PO line, its invoices minus credit notes in USD at today's rates.
	InvoicedOn(ctx context.Context, lines []POLineRef) (map[POLineRef]money.Amount, error)
}
type POLineRef struct{ PONumber, Line string }

// Approvals is the approval engine of 031 (brief §8.2); *approval.Service implements it.
type Approvals interface {
	SubmitSubject(ctx context.Context, a actor.Actor, kind approval.SubjectKind, id, comment string) error
	RecallSubject(ctx context.Context, a actor.Actor, kind approval.SubjectKind, id string) error
	DecideSubject(ctx context.Context, a actor.Actor, kind approval.SubjectKind, id string, d approval.Decision) error
	SectionOf(ctx context.Context, a actor.Actor, kind approval.SubjectKind, id string) (approval.Section, error)
}

// GSAP answers a routed purchase request with GSAP's PR number and a PO line per PR line (brief §8.3;
// feature 033 implements it; ADR-0016).
type GSAP interface {
	CreatePurchaseOrder(ctx context.Context, pr RoutedPR) (GSAPOrder, error)
}

type RoutedPR struct {
	ID, Name                string
	RequestedBy             actor.Actor
	ShortText               string
	VendorName, VendorCode  string
	GSAPPRNumber, PONumber  string // "" before GSAP's first answer
	Lines                   []RoutedLine
}
type RoutedLine struct {
	ID, LineItemID, HXKey, SwbsCode, CostCentre, GLCode, Currency string
	Amount       money.Amount
	DeliveryDate date.Date
	POLine       string // "" when the line has no PO line yet
}
type GSAPOrder struct {
	Source       string // "simulatedGsap" now; "gsap" for a live feed
	GSAPPRNumber string
	GSAPPRDate   date.Date
	PONumber     string
	Lines        []OrderLine // one per RoutedLine
}
type OrderLine struct {
	PRLineID, POLine, Status string
	Date                     date.Date
	Value                    money.Amount // in the PR line's currency
}
```

`approval.Decision` and `approval.Section` stand for the types 031 gives `DecideSubject` and `SectionOf`.
The service is built with `NewService(store Store, tools history.Tools, deps Deps)` and receives the engine
and GSAP through `UseApprovals(Approvals)` and `UseGSAP(GSAP)` (set after construction: the engine needs
the service first). Until `UseApprovals` is set, approval operations answer not found; until `UseGSAP` is
set, nothing calls GSAP (033 adds the caller).

### 2.4 What 032 implements for the engine (brief §8.2)

`*Service` implements `approval.PurchaseRequests`: `PurchaseRequest(ctx, id)`, `ListInScope(ctx, scope,
ids)` (plans scope from the engine, list scope from the context's grants), `ReadyCheck(ctx, a, id, kind)`
(the gates of §3.4) and `RoundChanged(ctx, a, id, kind, outcome)` (§3.5). Facts: ID, Name, Status, OwnerID,
MoAApproverID, Amount (USD total), WorkflowID, ModificationWorkflowID, Path (plan … activity of the first
line), Meta, Access and Plan facts of the PR's plan.

### 2.5 Views (contract schemas in brackets)

`View` (PurchaseRequest), `LineView` (PurchaseRequestLine), `VersionView` (PurchaseRequestVersion),
`RoutingView` (PurchaseRequestRouting), `Readiness` (PurchaseRequestReadiness), `Problem`
(PurchaseRequestProblem), `AccessView` (PurchaseRequestAccess), `Defaults` (PurchaseRequestDefaults),
`LineItemOption` (PurchaseRequestLineItemOption), `PageView` (PurchaseRequestListPage, rows and vendor
groups), `ClosePreview` (PurchaseRequestClosePreview), `ApprovalView` (PurchaseRequestApproval),
`Submission` (PurchaseRequestSubmission), `Preview` (PurchaseRequestApprovalPreview), `LineItemSection`
(LineItemPurchaseRequests), `ActivitySection` (ActivityPurchaseRequests), `ConfigurationView`
(PurchaseRequestConfiguration). Inputs: `Input` (PurchaseRequestInput), `LineInput`
(PurchaseRequestLineInput), `ConfigurationInput`, `CloseInput`, `ResendInput`, `SubmitInput`. Fields are in
the contract.

## 3. Rules (each has a unit test, plan "Rules and their tests")

### 3.1 Available to request (FR-012, FR-017, FR-018)

`requested(li) = Σ convert(c.Amount, c.Currency → li.Currency)` over `Claims(li)` excluding the line being
saved; `available = li.PlannedCost − requested`. A line passes when `0 < convert(amount → li.Currency) ≤
available`. Conversion: `money.Convert(amount, rates[from], rates[to])`, today's rates, half-up to cents. The
message: "Only <available> <li currency> of line item <id> is not yet requested." plus, for another
currency, "<amount> <currency> is <converted> <li currency> at today's rate."

### 3.2 Save rules (FR-005, FR-007, FR-010–FR-016, FR-036–FR-039)

All problems are collected before answering (400 `validation_failed`, `fields` per input path such as
`lines[1].amount`):
- name 1–120; description ≤ 2,000; owner an active person; MoA approver, when set, an active person;
  workflows exist, are active and have the right object type; attribute problems from `Attributes.Check`
  at each level; vendor e-mail well formed.
- 1 ≤ lines; each line item live, of the PR's plan, at most once; currency active; amount > 0 with 2
  decimals; delivery date valid; overall limit ≥ 0; the balance of §3.1; one filled company code.
- the version equals the stored version (409 `version_conflict`).
- status rules: Created/Declined `then` = save; Submitted/In review (first round) `then` = resubmit;
  Approved `then` = submitChanges, or save when the only change is added lines and the setting is
  withoutApproval; In review with a change pending `then` = resubmit; Closed refused (409 `not_allowed`,
  "Closed purchase requests cannot be changed."). Removing a line only on Created/Declined; adding a line to
  an Approved PR follows `lines_on_approved`; a modified amount ≥ the line's invoiced amount.
- Purchase Support on, else 409 `not_allowed` with "Purchase requests are switched off for your organisation."

### 3.3 Activity and SWBS facts

A line's activity counts as approved when its stored status is `approved` (In progress and Complete are
displays of Approved). SWBS: the line item's `SwbsCode` is not empty (030 status `received`).

### 3.4 Gates (FR-021) — `ReadyCheck`, `GET …/approval/submission`, the completeness chip

Problems, all at once, each `{field, group, recordId, message}` with group `purchaseRequest`, `line`,
`activity` or `lineItem`: no line; an activity not approved ("Activity '<name>' must be approved before this
purchase request can be submitted."); an SWBS code missing ("Line item <id> '<name>' has no SWBS code yet
(<SWBS status label>)."); two company codes, or none ("Activity '<name>' has no company code."); a workflow
inactive or of the wrong type; MoA approver empty; an overall limit empty; each needed-to-submit attribute
empty (header and lines). `ready` is true when there is none. A refused submission answers 409 `not_ready`
with the problems in `error.missing`, one `ReadinessGroup` per record (type `purchaseRequest`,
`purchaseRequestLine`, `activity` or `lineItem`), each problem a `MissingField` whose `reason` is the message.

### 3.5 Status transitions (Status Model; `RoundChanged`)

| Kind | Outcome | From | To | Side effects |
|---|---|---|---|---|
| purchaseRequest | open | created, declined | submitted | — |
| purchaseRequest | open | submitted, inReview | inReview | (a level approved; more remain) |
| purchaseRequest | superseded | submitted, inReview | created (within the same step) | — |
| purchaseRequest | approved | submitted, inReview | approved | approved version := current; route Original; Sent dates |
| purchaseRequest | declined | submitted, inReview | declined | — (reservation kept) |
| purchaseRequest | recalled, cancelled | submitted, inReview | created | — |
| modifiedPurchaseRequest | open | approved, inReview (change pending) | inReview, change pending | — |
| modifiedPurchaseRequest | superseded | inReview | inReview | — |
| modifiedPurchaseRequest | approved | inReview | approved | approved version := current; route Modification (changed lines marked); Sent dates of new lines |
| modifiedPurchaseRequest | declined, recalled, cancelled | inReview | approved | restore approved version: values back, lines it added removed |

Any other pair is refused (an internal error: the engine broke the contract). `RoundChanged` refuses every
outcome with 409 `not_allowed` while Purchase Support is off (the engine's Write rolls back).

### 3.6 Routing (FR-030–FR-035)

`route(pr, type, by, now)`: recipients = labels of the `emailGroup` and `prRecipients` roles' values, in
that order, else ["Requisitioner queue"]; one form row per line (Changed = the line differs from the
previous approved version or is new, Modification only); add the routing; set `SentOn = today (UTC)` on
lines without one (Original, Modification); write one "routed" log entry.

### 3.7 Waiting for GSAP (FR-032)

`status = approved AND routed at least once AND (no line has a PO line OR some line has po_line = '' OR
some line's (amount, currency) ≠ (po_value, currency))` — the store's `WaitingForGSAP` filter and the view's
`waitingForGsap` use the same rule.

### 3.8 Close (FR-041)

Allowed when status approved and no change pending. Per line: `invoiced = convert(InvoicedOn(PO line),
USD → line currency)` (0.00 without a PO line); `closed = min(amount, max(invoiced, 0))`; release `amount −
closed`. Then status closed, `closed_at`, route Close, log.

### 3.9 Delete and restore (FR-040, FR-042, FR-015)

Delete: Created/Declined only (reasons of the Record Operations table), Purchase Support on,
`purchaseRequests.delete` in the plan; `history.MoveToBin(kind purchaseRequest)` with its lines. Restore
(`PrepareRestore`): Purchase Support on; every line item live; every line fits §3.1 now; status set to
Created (Undo keeps the stored status). A line's restore: its PR exists, is Created or Declined, the line
item is not already on it, and the balance fits.

## 4. Settings additions

`settings.LevelPurchaseRequest = "purchaseRequest"`, `settings.LevelPurchaseRequestLine =
"purchaseRequestLine"` appended to `Levels`; nouns "purchase request(s)", "PR line(s)"; markings optional or
needed to submit; no `multipleChoiceSplit`. Roles `RoleVendor`, `RoleVendorCode`, `RoleVendorEmail`,
`RoleShortText`, `RolePRType`, `RoleEmailGroup`, `RolePRRecipients`, `RoleCompanyCode`, `RoleCostCentre`,
`RoleGLCode`; an attribute with a role has the locks of the Business unit attribute for level, type and
delete. Reference source `purchaseRequests` reports each PR (level purchaseRequest, attributes) and each line
(level purchaseRequestLine, attributes, currency).

## 5. Web app view models and state (`frontend/src/features/purchase-requests/`)

- Server state through TanStack Query; key factory `purchaseRequestKeys` — `all`, `list(filters)`,
  `detail(id)`, `approval(id)`, `submission(id)`, `lineItem(id)`, `activity(id)`, `options(search, prId)`,
  `defaults(lineItemId)`, `closePreview(id)`, `configuration`. Every mutation invalidates `detail`,
  `approval`, `list` and the line item and activity keys of its lines.
- URL state of the Purchase requests page, parsed by Zod (`schemas.ts`, fast-check property test): `q`
  (search), `status[]`, `type[]`, `vendor[]`, `owner[]`, `from`, `to` (YYYY-MM-DD), `waiting` (boolean),
  `sort`, `page`; an invalid value falls back to the default.
- Form state (React Hook Form + Zod `purchaseRequestFormSchema`): header fields, `attributes`, `lines[]`
  (`id?`, `lineItemId`, `currency`, `amount`, `deliveryDate`, `overallLimit`, `description`, `attributes`),
  `then`, `comment`; fast feedback only (required, decimal format, e-mail format); every rule is the server's.
- Local state: picker dialog open, confirmation dialogs, expanded vendor groups and PR rows (not shared).

## 6. Seed (research R15)

Requisitioner role RL-1006 and assignment; person U-1007 Lea Santos with membership in Global Marketing Ops;
configuration; 18 PR attributes and 3 PR line attributes (spec Field Inventory) with their options; workflows
"Purchase request approval (MoA)" and "Purchase request change approval (MoA)"; PR-1001 with line PRL-1001.
New permissions added to Global planner and Planner (spec FR-049). Seed order: after financial transactions,
before the demo history (D-306).
