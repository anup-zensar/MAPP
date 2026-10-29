package allwork

// BCPIR plans (feature 013): the marker of plan rows, the filter and the Breakdown grouping.

const (
	bcpirField = "plan.bcpir"
	byBCPIR    = "bcpir"
)

// isBCPIR reads the flag from the plan's "plan.bcpir" column, which the plan reader always fills.
func isBCPIR(n *Node) bool {
	v := n.Fields[bcpirField]
	return v.Yes != nil && *v.Yes
}

func bcpirFilter(q Query) []filter {
	plans := []Level{LevelPlan}
	switch q.BCPIR {
	case StrategicOnly:
		return []filter{{levels: plans, meets: isBCPIR}}
	case StrategicExclude:
		return []filter{{levels: plans, meets: func(n *Node) bool { return !isBCPIR(n) }}}
	}
	return nil
}

func bcpirChips(q Query) []string {
	switch q.BCPIR {
	case StrategicOnly:
		return []string{"BCPIR plans: Only"}
	case StrategicExclude:
		return []string{"BCPIR plans: Exclude"}
	}
	return nil
}

func bcpirValues(plan *tnode) []value {
	if isBCPIR(plan.n) {
		return []value{{key: AttrYes, label: labelYes, order: 0}}
	}
	return []value{{key: AttrNo, label: labelNo, order: 1}}
}
