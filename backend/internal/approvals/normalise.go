package approvals

import "fmt"

// Data stored before feature 017 reads with the defaults of data-model §10 (017-P19): no migration, and
// every existing workflow and round behaves exactly as before. The JSON repository normalises what it reads.

// normalisedLevels returns levels with the defaults of a level stored before 017: named people, any one
// approves, approvers never nil.
func normalisedLevels(levels []Level) []Level {
	for i, l := range levels {
		if l.ApproverType == "" {
			l.ApproverType = SourcePeople
		}
		if l.Mode == "" {
			l.Mode = ModeAny
		}
		if l.ApproverIDs == nil {
			l.ApproverIDs = []string{}
		}
		levels[i] = l
	}
	return levels
}

// normalised returns w with its levels normalised.
func (w Workflow) normalised() Workflow {
	w.Levels = normalisedLevels(w.Levels)
	return w
}

// normalised returns r with the defaults of a round stored before 017: each level of named people, any one
// approves, decided by its approvers, nobody left out.
func (r Round) normalised() Round {
	for i, l := range r.Levels {
		if l.ApproverType == "" {
			l.ApproverType = SourcePeople
		}
		if l.Mode == "" {
			l.Mode = ModeAny
		}
		if l.Resolution == "" {
			l.Resolution = ResolutionDecide
		}
		if l.Approvers == nil {
			l.Approvers = nonNil(l.Approvers)
		}
		if l.Excluded == nil {
			l.Excluded = []Excluded{}
		}
		r.Levels[i] = l
	}
	return r
}

// daysText writes a number of days as the texts do: "1 day", "3 days".
func daysText(n int) string {
	if n == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", n)
}
