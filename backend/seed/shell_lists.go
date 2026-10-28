package seed

// Shell's own option lists, as the Hive9 functional analysis (E3.0.2) records them live, character for
// character: configuration, not demo data (D-325). Spend type holds the 25 of Shell's 57 the analysis names.

func shellZeroBasedCostHierarchy() []demoOption {
	return opts(
		"LUBES - Growth drivers", "LUBES - Legal or regulatory requirements", "LUBES - Long term contractual obligations",
		"LUBES - Other", "LUBES - Protecting the core", "LUBES - Strategic Priorities", "LUBES - Tactical initiatives",
		"RETAIL - Long-Term Contracts", "RETAIL - Strategic Priorities", "RETAIL - Tactical Investments",
	)
}

func shellSpendTypes() []demoOption {
	return opts(
		"ATL - Agency Fees - Creative", "ATL - Agency Fees - PR", "ATL - Digital", "ATL - Event",
		"ATL - Paid Media Campaign Activation", "ATL - Paid Media Strategy and Planning", "ATL - Public Relations",
		"ATL - Sponsorships", "Audits e.g. mystery shopper", "BTL - Agency Fees", "BTL - Branded Shop",
		"BTL - Capability building Training", "BTL - CRM Campaign", "BTL - Customer Meetings/Conference",
		"BTL - Distributor Marketing Budget", "BTL - Event", "BTL - Incentive Scheme", "BTL - Loyalty Operations Costs",
		"BTL - Market research/Insights", "BTL - Merchandise", "BTL - Pipeline", "BTL - Promotions", "BTL - Sponsorship",
		"Contractors or Field Support", "CRM - Agency Fee",
	)
}

func shellGLCodes() []demoOption {
	return opts(
		"6140900 - PMTDR Credit", "7250050 - Advertising - Media Planning/Auditing",
		"7250060 - Advertising - Corp/Agency/Production", "7250070 - Advertising - Media Buying",
		"7250080 - Advertising - Digital", "7250100 - Public Relations", "7250150 - Sponsorship programmes",
		"7250190 - Loyalty Programmes - Materials", "7250200 - Loyalty Programmes - Services", "7250250 - Market Research",
		"7250300 - Promotions & POP - Incentives/Reimbursement", "7250310 - Promotions & POP - Agency/Print/Logistics",
		"7250350 - Brand Development & Coordination",
	)
}

func shellSpendCategories() []demoOption {
	return opts(
		"Not Applicable", "ABM (B2B Media)", "B2B Customer Specific", "Brand", "Canada Customer Specific",
		"Channel - DIFM", "Channel - DIY", "Channel - Indirect", "Consumer Media", "Content, Creative & Assets",
		"Influencer Marketing", "Innovation", "Insights", "MarTech", "Motorsports", "Program", "Promotions",
		"Sustainability", "Tradeshow",
	)
}

func shellContractedSpend() []demoOption {
	return opts("Not Applicable", "No", "Yes - Both", "Yes - LT", "Yes - ST")
}
