package settings

// The Shell attribute levels and attribute roles of feature 032's purchase requests (032 data-model §4).
const (
	LevelPurchaseRequest     Level = "purchaseRequest"
	LevelPurchaseRequestLine Level = "purchaseRequestLine"
)

const (
	RoleVendor       = "vendor"
	RoleVendorCode   = "vendorCode"
	RoleVendorEmail  = "vendorEmail"
	RoleShortText    = "shortText"
	RolePRType       = "prType"
	RoleEmailGroup   = "emailGroup"
	RolePRRecipients = "prRecipients"
	RoleCompanyCode  = "companyCode"
	RoleCostCentre   = "costCentre"
	RoleGLCode       = "glCode"
)

const (
	prLineWords = "PR line"
	roleLock    = "SP&A One reads this attribute for purchase requests, so its level and type stay and it can't be deleted."
)

func AllLevels() []Level {
	return append(append([]Level{}, Levels...), LevelPurchaseRequest, LevelPurchaseRequestLine)
}

// splits reports whether attributes of level may be multiple choice with % split (D-029): activity and line item.
func splits(level Level) bool {
	return level == LevelActivity || level == LevelLineItem
}

func nounsOf(level Level) [2]string {
	switch level {
	case LevelPurchaseRequest:
		return [2]string{"purchase request", "purchase requests"}
	case LevelPurchaseRequestLine:
		return [2]string{prLineWords, prLineWords + "s"}
	case LevelPlan, LevelCampaign, LevelInitiative, LevelActivity, LevelLineItem:
	}
	return levelNouns[level]
}
