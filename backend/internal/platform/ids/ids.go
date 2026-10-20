// Package ids gives records their permanent readable IDs: "P-1001", "P-1002", … (FR-027, research R4).
//
// Each prefix has its own counter, which only ever goes up, so an ID is never given out twice, not even
// after its record was deleted. The counter is read and raised inside the caller's Write, so a new ID and
// its new record are saved together. Every record with a readable ID, seed records included, gets it
// from a Generator.
//
// The prefixes are the constants below: one per record type, never shared, so that every counter belongs to
// one table.
package ids

import (
	"context"
	"fmt"
)

// The readable-ID prefixes (001 data model §4 and the data models of features 002–008). Pass them to Next.
const (
	// Feature 001.
	PrefixPerson       = "U"   // people: U-1001
	PrefixLogEntry     = "LOG" // activity-log entries (their number orders the log)
	PrefixBinEntry     = "BIN" // recycle-bin entries
	PrefixSampleFolder = "SF"  // developer sample folders
	PrefixSampleNote   = "SN"  // developer sample notes

	// Feature 002, Settings (internal ids, never shown on screens; a currency's id is its ISO code).
	PrefixModel             = "MOD" // models
	PrefixActivityType      = "AT"  // activity types
	PrefixShellAttribute    = "SA"  // Shell attributes
	PrefixShellOption       = "SO"  // Shell attribute options
	PrefixFinanceBudget     = "FB"  // finance budgets
	PrefixFinanceBudgetItem = "FBI" // finance budget items

	// Feature 003, people and teams.
	PrefixTeam           = "T"  // teams
	PrefixTeamMembership = "TM" // team memberships

	// Features 004–007, the planning records (D-019).
	PrefixPlan       = "P" // plans (004)
	PrefixCampaign   = "C" // campaigns (005)
	PrefixInitiative = "I" // initiatives (005)
	PrefixActivity   = "A" // activities (006)
	PrefixLineItem   = "L" // line items (007)

	// Feature 008, approvals (D-201: workflows keep 006's WF).
	PrefixApprovalWorkflow = "WF" // approval workflows
	PrefixApprovalRound    = "AR" // approval rounds
	PrefixApprovalAction   = "AH" // approval history entries
	PrefixNotification     = "NT" // notifications

	// Feature 010, All work (D-125).
	PrefixSavedView = "SV" // saved views

	// Feature 014, access control and global budgets.
	PrefixRole             = "RL" // roles
	PrefixRoleAssignment   = "RA" // role assignments
	PrefixGlobalBudget     = "GB" // global budgets
	PrefixBudgetAllocation = "BA" // budget allocations
)

// Prefixes lists every prefix above, in that order.
var Prefixes = []string{
	PrefixPerson, PrefixLogEntry, PrefixBinEntry, PrefixSampleFolder, PrefixSampleNote,
	PrefixModel, PrefixActivityType, PrefixShellAttribute, PrefixShellOption, PrefixFinanceBudget, PrefixFinanceBudgetItem,
	PrefixTeam, PrefixTeamMembership,
	PrefixPlan, PrefixCampaign, PrefixInitiative, PrefixActivity, PrefixLineItem,
	PrefixApprovalWorkflow, PrefixApprovalRound, PrefixApprovalAction, PrefixNotification,
	PrefixSavedView,
	PrefixRole, PrefixRoleAssignment, PrefixGlobalBudget, PrefixBudgetAllocation,
}

// firstNumber is the number of the first ID of every prefix: "P-1001".
const firstNumber = 1001

// Counters is the repository of the last number given per prefix. The JSON store keeps it in the table
// "counters"; PostgreSQL will have its own implementation.
type Counters interface {
	// Increment raises the counter of prefix by one and returns the new number, 1001 the first time.
	// It runs inside the caller's Write.
	Increment(ctx context.Context, prefix string) (int, error)
}

// Generator gives out readable IDs.
type Generator struct {
	counters Counters
}

// NewGenerator returns a Generator that keeps its counters in c.
func NewGenerator(c Counters) *Generator {
	return &Generator{counters: c}
}

// Next returns the next ID of prefix, such as "P-1001". Call it inside a Write.
func (g *Generator) Next(ctx context.Context, prefix string) (string, error) {
	number, err := g.NextNumber(ctx, prefix)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%d", prefix, number), nil
}

// NextNumber returns the number of the next ID of prefix, without the prefix. The activity log uses it to
// order its entries (LOG-1001 has the sequence number 1001). Call it inside a Write.
func (g *Generator) NextNumber(ctx context.Context, prefix string) (int, error) {
	return g.counters.Increment(ctx, prefix)
}
