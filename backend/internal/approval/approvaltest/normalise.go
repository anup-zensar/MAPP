package approvaltest

import (
	"cmp"

	"mapp/internal/approval"
	"mapp/internal/platform/actor"
)

// normalisedWorkflow saves a level with the defaults of one stored before feature 017, as the PostgreSQL store
// does with approval's unexported normaliser: named people, any one approves.
func normalisedWorkflow(w approval.Workflow) approval.Workflow {
	levels := make([]approval.Level, 0, len(w.Levels))
	for _, l := range w.Levels {
		l.ApproverType = cmp.Or(l.ApproverType, approval.SourcePeople)
		l.Mode = cmp.Or(l.Mode, approval.ModeAny)
		if l.ApproverIDs == nil {
			l.ApproverIDs = []string{}
		}
		levels = append(levels, l)
	}
	w.Levels = levels
	return w
}

func normalisedRound(r approval.Round) approval.Round {
	if r.Levels == nil {
		return r
	}
	levels := make([]approval.RoundLevel, 0, len(r.Levels))
	for _, l := range r.Levels {
		l.ApproverType = cmp.Or(l.ApproverType, approval.SourcePeople)
		l.Mode = cmp.Or(l.Mode, approval.ModeAny)
		l.Resolution = cmp.Or(l.Resolution, approval.ResolutionDecide)
		if l.Approvers == nil {
			l.Approvers = []actor.Actor{}
		}
		if l.Excluded == nil {
			l.Excluded = []approval.Excluded{}
		}
		levels = append(levels, l)
	}
	r.Levels = levels
	return r
}
