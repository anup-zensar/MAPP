package allwork

const (
	StrategicInclude = "include"
	StrategicOnly    = "only"
	StrategicExclude = "exclude"

	ShowEverything                   = "everything"
	ShowCompleteBranches             = "completeBranches"
	ShowCampaignsWithoutInitiatives  = "campaignsWithoutInitiatives"
	ShowInitiativesWithoutActivities = "initiativesWithoutActivities"

	PeriodMonth   = "month"
	PeriodQuarter = "quarter"
	PeriodYear    = "year"

	AttrNotFilledIn = "none"
	AttrYes         = "yes"
	AttrNo          = "no"

	AmountsInDisplay = "display"
	AmountsInPlan    = "plan"
)

const ViewBreakdown = "breakdown"

type AttrFilter struct {
	AttributeID string   `json:"attributeId"`
	Values      []string `json:"values"`
}

type Query struct {
	Year              int
	View              string
	Plans             []string
	PlanStatuses      []string
	ContainerStatuses []string
	ActivityStatuses  []string
	Owners            []string
	Teams             []string
	BusinessUnits     []string
	ActivityTypes     []string
	Finance           []string
	Strategic         string
	BCPIR             string
	ShowLevels        string
	Search            string
	Columns           []string
	Sort              *Sort
	Period            string

	YearFrom, YearTo int
	PlanTypes        []string
	PlanCurrencies   []string
	Attributes       []AttrFilter
	ComparePhased    bool
	By, ThenBy       string
	AmountsIn        string

	attrs []attrTest
}
