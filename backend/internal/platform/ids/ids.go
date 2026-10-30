// Package ids gives records permanent readable IDs, "P-1001": each prefix has a counter that only goes up,
// raised inside the caller's Write so a new ID and its record are saved together.
package ids

import (
	"context"
	"fmt"
)

// One prefix per record type, never shared, so every counter belongs to one table.
const (
	PrefixPerson       = "U"
	PrefixLogEntry     = "LOG"
	PrefixBinEntry     = "BIN"
	PrefixSampleFolder = "SF"
	PrefixSampleNote   = "SN"

	PrefixModel             = "MOD"
	PrefixActivityType      = "AT"
	PrefixShellAttribute    = "SA"
	PrefixShellOption       = "SO"
	PrefixFinanceBudget     = "FB"
	PrefixFinanceBudgetItem = "FBI"

	PrefixTeam           = "T"
	PrefixTeamMembership = "TM"

	PrefixPlan       = "P"
	PrefixCampaign   = "C"
	PrefixInitiative = "I"
	PrefixActivity   = "A"
	PrefixLineItem   = "L"

	PrefixApprovalWorkflow = "WF"
	PrefixApprovalRound    = "AR"
	PrefixApprovalAction   = "AH"
	PrefixNotification     = "NT"

	PrefixSavedView = "SV"

	PrefixFinancialTransaction = "FT"
	PrefixFinancialImport      = "FIM"

	PrefixApprovalDelegation = "DLG"
	PrefixRole               = "RL"
	PrefixRoleAssignment     = "RA"
	PrefixGlobalBudget       = "GB"
	PrefixBudgetAllocation   = "BA"

	PrefixExchangeRate = "XR"
	PrefixRateImport   = "RI"

	PrefixSwbsRun = "SWR"

	PrefixPurchaseRequest        = "PR"
	PrefixPurchaseRequestLine    = "PRL"
	PrefixPurchaseRequestRouting = "PRR"
	PrefixSimulatedGSAPPR        = "GPR"
	PrefixSimulatedPO            = "GPO"

	PrefixBulkAction = "BLK"
	PrefixPlanImport = "IMP"

	PrefixPerformanceSource    = "PS"
	PrefixPerformanceMapping   = "PSM"
	PrefixPerformanceImport    = "PIM"
	PrefixPerformanceFact      = "PF"
	PrefixPerformanceReviewRow = "PRV"
	PrefixMediaCode            = "MC"
	PrefixExternalNameAlias    = "XN"
	PrefixIntegrationID        = "IID"
)

var Prefixes = []string{
	PrefixPerson, PrefixLogEntry, PrefixBinEntry, PrefixSampleFolder, PrefixSampleNote,
	PrefixModel, PrefixActivityType, PrefixShellAttribute, PrefixShellOption, PrefixFinanceBudget, PrefixFinanceBudgetItem,
	PrefixTeam, PrefixTeamMembership,
	PrefixPlan, PrefixCampaign, PrefixInitiative, PrefixActivity, PrefixLineItem,
	PrefixApprovalWorkflow, PrefixApprovalRound, PrefixApprovalAction, PrefixNotification,
	PrefixSavedView,
	PrefixFinancialTransaction, PrefixFinancialImport,
	PrefixApprovalDelegation,
	PrefixRole, PrefixRoleAssignment, PrefixGlobalBudget, PrefixBudgetAllocation,
	PrefixExchangeRate, PrefixRateImport,
	PrefixSwbsRun,
	PrefixPurchaseRequest, PrefixPurchaseRequestLine, PrefixPurchaseRequestRouting, PrefixSimulatedGSAPPR, PrefixSimulatedPO,
	PrefixBulkAction, PrefixPlanImport,
	PrefixPerformanceSource, PrefixPerformanceMapping, PrefixPerformanceImport, PrefixPerformanceFact,
	PrefixPerformanceReviewRow, PrefixMediaCode, PrefixExternalNameAlias, PrefixIntegrationID,
}

const firstNumber = 1001

type Counters interface {
	// Advance raises the counter of prefix by count and returns the last number given out: 1001 the first time.
	Advance(ctx context.Context, prefix string, count int) (int, error)
}

type Generator struct {
	counters Counters
}

func NewGenerator(c Counters) *Generator {
	return &Generator{counters: c}
}

func (g *Generator) Next(ctx context.Context, prefix string) (string, error) {
	number, err := g.NextNumber(ctx, prefix)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%d", prefix, number), nil
}

// NextNumber orders the activity log: LOG-1001 has the sequence number 1001.
func (g *Generator) NextNumber(ctx context.Context, prefix string) (int, error) {
	return g.counters.Advance(ctx, prefix, 1)
}

func (g *Generator) NextIDs(ctx context.Context, prefix string, count int) ([]string, error) {
	if count < 1 {
		return []string{}, nil
	}
	last, err := g.counters.Advance(ctx, prefix, count)
	if err != nil {
		return nil, err
	}
	list := make([]string, 0, count)
	for number := last - count + 1; number <= last; number++ {
		list = append(list, fmt.Sprintf("%s-%d", prefix, number))
	}
	return list, nil
}
