package performance

import "strconv"

const (
	activityKind    = "activity"
	factKind        = "performanceFact"
	mediaCodeKind   = "mediaCode"
	aliasKind       = "externalNameAlias"
	integrationKind = "integrationId"

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
	msgNameHeld    = "Not saved — %s is already the external name or alias of activity %s. A name can belong to only one activity."
)

const (
	deactivatedVerb = "deactivated"
	reactivatedVerb = "reactivated"
)

func codeText(code int64) string {
	return strconv.FormatInt(code, 10)
}

const (
	fieldColumnName = "columnName"
	fieldMeaning    = "meaning"
	fieldStage      = "stage"
	fieldWebMetric  = "webMetric"
	fieldModelID    = "modelId"

	msgColumnLength   = "Give the column a name of 1 to 120 characters."
	msgMeaningUnknown = "Choose what this column means."
	msgStageNeeded    = "A stage column names its stage, and only a stage column does."
	msgMetricNeeded   = "A web-metric column names its metric, and only a web-metric column does."
	msgMetricUnknown  = "A web metric is sessions, visits or engagement touches."
	msgModelNeeded    = "Only a stage or revenue column is supplied by a model, and it must name one."
)

const (
	reasonNoKey     = "unattributed:noKey"
	reasonUnknown   = "unmatched:unknown"
	reasonAmbiguous = "unmatched:ambiguous"
	reasonDisagree  = "unmatched:disagree"
)
