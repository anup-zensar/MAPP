package allwork

// keepCompleteBranches hides every plan, campaign and initiative without a shown activity below it
// (Complete branches only, FR-007) and reports whether n reaches one.
func keepCompleteBranches(n *tnode) bool {
	if n.n.Level == LevelActivity {
		return n.shown
	}
	reaches := false
	for _, c := range n.children {
		if keepCompleteBranches(c) {
			reaches = true
		}
	}
	if !reaches {
		n.shown = false
	}
	return reaches
}

// keepWithoutChildren keeps under the plan p only the records of level that have no child at all and match
// the filters on their level, with the records above them as context rows (Campaigns without initiatives,
// Initiatives without activities, FR-007); everything else is hidden.
func keepWithoutChildren(p *tnode, level Level) {
	var visit func(n *tnode) bool
	visit = func(n *tnode) bool {
		if n.n.Level == level {
			keep := n.match && len(n.children) == 0
			n.match, n.shown = keep, keep
			for _, c := range n.children {
				hide(c)
			}
			return keep
		}
		n.match, n.shown = false, false
		for _, c := range n.children {
			if visit(c) {
				n.shown = true
			}
		}
		return n.shown
	}
	visit(p)
}

// hide hides n and everything below it.
func hide(n *tnode) {
	n.match, n.shown = false, false
	for _, c := range n.children {
		hide(c)
	}
}
