package approval

import "fmt"

// normalisedLevels gives a level saved before feature 017 its defaults, so it behaves as before, unmigrated.
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

func daysText(n int) string {
	if n == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", n)
}
