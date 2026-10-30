package performance

const (
	sourceKind         = "performanceSource"
	sourceKindSingular = "data source"
	sourceKindPlural   = "data sources"

	msgSourceNotFound  = "This data source doesn't exist or you can't see it."
	msgMappingNotFound = "This column mapping doesn't exist or you can't see it."
	msgHoldsFacts      = "Not deleted — %s holds performance data. Deactivate it instead: its data stays and it stops accepting new loads."
	msgSourceInactive  = "Not loaded — %s is deactivated and accepts no further data. Reactivate it first."
	msgStageSupplied   = "Not saved — %s already supplies %s for this model. Deactivate that source first, or choose another stage."
	msgRevenueSupplied = "Not saved — %s already supplies revenue for this model. Deactivate that source first."
)

const (
	msgNameLength  = "Give the data source a name of 1 to 120 characters."
	msgOwnerNeeded = "Choose an owner for this data source."
)

const (
	deactivatedVerb = "deactivated"
	reactivatedVerb = "reactivated"
)
