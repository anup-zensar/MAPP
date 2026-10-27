package stores

import "mapp/internal/approvals"

// ApprovalStore is the approvals store as the wiring and the seed use it: the service's Store and what other areas' adapters call.
type ApprovalStore interface {
	approvals.Store
}
