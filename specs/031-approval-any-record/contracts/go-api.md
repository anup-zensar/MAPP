# Go API of the approval engine for other record types (consumed by feature 032)

Package `mapp/internal/approval`. Implements integration brief §8.2 exactly; additions are marked **(+)**.

## Record types

```go
type SubjectKind string

const (
	KindActivity                SubjectKind = "activity"
	KindPurchaseRequest         SubjectKind = "purchaseRequest"
	KindModifiedPurchaseRequest SubjectKind = "modifiedPurchaseRequest"
)

func (k SubjectKind) IsValid() bool   // (+) one of the three
func (k SubjectKind) Label() string   // (+) "Activity", "Purchase request", "Modified purchase request"
```

Workflows: `Workflow.ObjectType SubjectKind` (`Workflow.Kind()` (+) answers `KindActivity` for an empty
value). 032's workflow pickers list `stores.ApprovalStore.ListWorkflows` and keep `Active` workflows whose
`Kind()` is the field's kind (as `internal/app/activity_workflows.go` does for activities). Level sources:
`SourceOwner = "owner"`, `SourceRecordField = "recordField"` with `Level.RecordField ==
RecordFieldMoAApprover` (`"moaApprover"`, label "MoA approver").

## Port (declared by approval, implemented by internal/app over feature 032)

```go
type PurchaseRequests interface {
	PurchaseRequest(ctx context.Context, id string) (PurchaseRequestFacts, bool, error)
	ListInScope(ctx context.Context, scope authz.Scope, ids []string) ([]PurchaseRequestFacts, error)
	// ReadyCheck runs 032's submission gates (013 FR-030) for the round kind about to start.
	ReadyCheck(ctx context.Context, a actor.Actor, id string, kind SubjectKind) (ReadyCheck, error)
	// RoundChanged tells 032 a round started or ended so it sets the PR status (013 FR-031 mapping) and
	// applies or restores a modification; it writes no log entry (approvals writes it).
	RoundChanged(ctx context.Context, a actor.Actor, id string, kind SubjectKind, outcome Outcome) (PurchaseRequestFacts, error)
}

type PurchaseRequestFacts struct {
	ID, Name, Status       string // status: created, submitted, inReview, declined, approved, closed
	OwnerID, MoAApproverID string
	Amount                 *money.Money // total in USD; nil when no lines
	WorkflowID             string       // approval workflow (kind purchaseRequest)
	ModificationWorkflowID string       // approval workflow for modification (kind modifiedPurchaseRequest)
	Path                   []errs.RecordRef // plan … activity of the PR's lines, top level first
	record.Meta
	Access                 authz.PlanFacts
	Plan                   PlanFacts
}

func (s *Service) UsePurchaseRequests(p PurchaseRequests)
```

Every method runs inside the caller's transaction (ctx). Until `UsePurchaseRequests` is called, every
purchase-request operation answers `errs.NotFound("This purchase request no longer exists — someone may have
deleted it.")`.

**Events** passed to `RoundChanged` (data-model §8): `OutcomeOpen` when a round starts **and again after
every decision that approves a level while later levels remain** (the round stays open; 032 shows In review;
a partial approval of an "all must approve" level reports nothing), `OutcomeApproved`, `OutcomeDeclined`,
`OutcomeRecalled`, `OutcomeSuperseded` (+) (always followed at once by `OutcomeOpen` of the new round). A
round approved at submission reports `OutcomeOpen` then `OutcomeApproved`. `OutcomeRestarted` and
`OutcomeCancelled` are activity-only today (re-approval change, restore from the recycle bin); the engine
never sends them to purchase requests.

**Storage** (migration 00018): `approval_rounds.subject_kind`, `approval_rounds.subject_id`,
`approval_actions.subject_kind`, `approval_actions.subject_id`, `notifications.subject_kind`,
`notifications.subject_id`, `notifications.subject_name`. `subject_kind` is one of `activity`,
`purchaseRequest`, `modifiedPurchaseRequest`; a PR's rounds of both kinds have `subject_id` = the PR id, so a
scope function joins `approval_rounds r ON r.subject_id = pr.id AND r.subject_kind IN ('purchaseRequest',
'modifiedPurchaseRequest')`.

**Status constants** (+): `StatusCreated`, `StatusSubmitted`, `StatusInReview`, `StatusDeclined`,
`StatusApproved`, `StatusClosed` — the engine offers actions from them (data-model §9).

**ReadyCheck** problems (`SubmitProblem`): `Group` is `GroupPurchaseRequest` (`"purchaseRequest"`) for the
PR's own fields and `GroupPurchaseRequestLine` (+) (`"purchaseRequestLine"`) for a line; the engine adds the
workflow's problems after the PR's own (field `approvalWorkflowId`, or `modificationWorkflowId` for a
modification).

**History kind**: the engine logs every event with `history.LogInput{Kind: "purchaseRequest", RecordID:
<PR id>, …}`; 032 registers the history kind named `"purchaseRequest"`.

## Subject API

```go
// SubmitSubject submits (or re-submits) the record id of kind with an optional comment (≤ 2,000 characters).
func (s *Service) SubmitSubject(ctx context.Context, a actor.Actor, kind SubjectKind, id, comment string) (Section, error)

// RecallSubject recalls the record's open round (no reason).
func (s *Service) RecallSubject(ctx context.Context, a actor.Actor, kind SubjectKind, id string) (Section, error)

// DecideSubject approves or declines the current level of the record's open round.
func (s *Service) DecideSubject(ctx context.Context, a actor.Actor, kind SubjectKind, id string, decision SubjectDecision) (Section, error)

// SectionOf returns the record's approval section: every round of the record (both PR kinds), newest first.
func (s *Service) SectionOf(ctx context.Context, a actor.Actor, kind SubjectKind, id string) (Section, error)

type SubjectDecision struct {
	Action        ActionKind // ActionApproved or ActionDeclined
	DecisionInput            // roundId, level (as the person saw them), comment (required to decline)
}

type Section struct {
	Subject        SubjectView  `json:"subject"`
	OwnerID        string       `json:"ownerId"`
	Owner          *PersonView  `json:"owner"`
	Workflow       *WorkflowRef `json:"workflow"`       // the kind's workflow field now
	UpcomingLevels []LevelView  `json:"upcomingLevels"` // while the record has no round
	Rounds         []RoundView  `json:"rounds"`         // every round, newest first; RoundView.Kind says which
	LatestDecline  *DeclineNote `json:"latestDecline"`
	Viewer         Viewer       `json:"viewer"`
	Actions        ActionSet    `json:"actions"`        // computed for the kind asked (data-model §9)
}

type SubjectView struct {
	Kind          SubjectKind `json:"kind"`
	Label         string      `json:"label"`
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Status        string      `json:"status"`
	DisplayStatus string      `json:"displayStatus"`
	Version       int         `json:"version"`
}
```

The contract schemas `ApprovalSection` and `ApprovalSubject` in `backend/api/openapi.yaml` describe the
JSON of `Section` for 032's routes.

Errors: `errs.NotFound` (record not stored or not wired; round of another record), `errs.Validation` (all
problems at once; decline without comment; unknown decision), `errs.BadRequest` (not submittable now;
nothing to recall), `errs.Forbidden` (not allowed to decide or recall), `errs.VersionConflict`-shaped 409
(the round moved on, was recalled or replaced).
