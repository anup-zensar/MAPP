package settings

// The Field help catalogue (016 FR-045, data-model §6, D-271): every built-in field and calculated value of
// the planning levels and pages, with a stable key, its screen label and the help text MAPP supplies.
// Administrators may edit a text in Settings › Field help; only edited texts are stored. A new field gets
// its default here, without any data change. Shell attributes keep their help on each attribute.

type HelpLevel string

const (
	HelpPlan       HelpLevel = "plan"
	HelpCampaign   HelpLevel = "campaign"
	HelpInitiative HelpLevel = "initiative"
	HelpActivity   HelpLevel = "activity"
	HelpLineItem   HelpLevel = "lineItem"
	HelpPhasing    HelpLevel = "phasing"
	HelpCommon     HelpLevel = "common"
	HelpAllWork    HelpLevel = "allWork"
)

// helpLevelNames name each level at the start of an entry's name in the activity log: "Plan › Plan budget".
var helpLevelNames = map[HelpLevel]string{
	HelpPlan: "Plan", HelpCampaign: "Campaign", HelpInitiative: "Initiative", HelpActivity: "Activity",
	HelpLineItem: "Line item", HelpPhasing: "Budget by month", HelpCommon: "Every record", HelpAllWork: "All work",
}

type helpDefinition struct {
	Key, Label, Default string
	Level               HelpLevel
}

// helpCatalogue is the catalogue, in the order of the Settings page.
var helpCatalogue = []helpDefinition{
	{"plan.name", "Plan name", `A name everyone recognises, such as "FY2027 Lubricants UK & Ireland". It must be unique in MAPP.`, HelpPlan},
	{"plan.year", "Year", "The fiscal year the plan covers, January to December. It can't change after the plan is saved.", HelpPlan},
	{"plan.planType", "Plan type", "Operational: one business's or market's plan for the year. Strategic: sets direction for several years. Cross-market: campaigns that run across several markets.", HelpPlan},
	{"plan.model", "Model", "The plan's rulebook: which activity types it allows and how inquiries turn into MQLs, deals and revenue.", HelpPlan},
	{"plan.teams", "Teams", "The teams that work on this plan. The owner must belong to one of them.", HelpPlan},
	{"plan.owner", "Owner", "The person responsible for the plan. Choose a member of the plan's teams.", HelpPlan},
	{"plan.aop", "Strategic / AOP plan", `Marks the plan as part of the annual operating plan. AOP plans carry an "AOP" tag in lists.`, HelpPlan},
	{"plan.description", "Description", "What the plan is for, in a few sentences.", HelpPlan},
	{"plan.currency", "Plan currency", "The currency the plan's budgets and planned costs are typed in. Each amount also shows in USD or your display currency. It can't change once amounts are planned in it.", HelpPlan},
	{"plan.budget", "Plan budget", "The money this plan may spend in its year, in the plan currency. Campaign budgets are allocated from it.", HelpPlan},
	{"plan.budgetPhasing", "Budget by month", "How the budget is spread over the months of the year: not phased, evenly or month by month. It is compared with each month's planned cost.", HelpPlan},
	{"plan.defaultFinanceBudget", "Default finance budget", "The finance budget the plan's line items link to by default. Its items are listed first on each line item.", HelpPlan},
	{"plan.defaultSpread", "Default line-item spread", "How new line items spread their cost over months, unless someone changes it on the line item.", HelpPlan},
	{"plan.bcpir", "BCPIR plan", "Marks the plan as a BCPIR plan.", HelpPlan},
	{"plan.goalStage", "Goal stage", "The funnel stage the plan's goal is set at: INQ, MQL or Revenue. The other stages are calculated with the model.", HelpPlan},
	{"plan.goalAmount", "Goal amount", "The plan's goal at the chosen stage: a number of inquiries or MQLs, or revenue in USD.", HelpPlan},
	{"plan.status", "Status", "Draft while the plan is prepared; Published once it is the agreed plan. A published plan can still be edited.", HelpPlan},
	{"plan.plannedCost", "Planned cost", "The sum of the planned costs of every activity in the plan.", HelpPlan},
	{"plan.remainingBudget", "Remaining budget", "Plan budget minus the budgets given to campaigns. Below zero means campaigns have more budget than the plan.", HelpPlan},
	{"plan.budgetUsed", "Budget used", "Planned cost divided by the plan budget. Red above 100 %.", HelpPlan},
	{"plan.activitiesApproved", "Activities approved", "Activities that are Approved, In progress or Complete, out of all activities in the plan.", HelpPlan},
	{"plan.goals", "Goals", "The plan's goal next to the sum of its activities' goals.", HelpPlan},
	{"plan.plannedCostBy", "Planned cost by", "The plan's planned cost added up by a category, such as spend type, brand or country.", HelpPlan},

	{"campaign.name", "Campaign name", "A theme that runs for most of the year. Unique in its plan.", HelpCampaign},
	{"campaign.owner", "Owner", "The person responsible for the campaign. It starts as the plan's owner.", HelpCampaign},
	{"campaign.dates", "Dates", "When the campaign runs. Expected inside the plan's year; initiatives must fit inside these dates.", HelpCampaign},
	{"campaign.description", "Description", "What the campaign is about.", HelpCampaign},
	{"campaign.budget", "Campaign budget", "The part of the plan budget given to this campaign, in the plan currency. Initiative budgets are allocated from it.", HelpCampaign},
	{"campaign.budgetPhasing", "Budget by month", "How the campaign budget is spread over the months of the plan's year.", HelpCampaign},
	{"campaign.remainingPlanBudget", "Remaining plan budget", "Plan budget minus all campaign budgets, including this one. Red when campaigns have more than the plan.", HelpCampaign},
	{"campaign.plannedCost", "Planned cost", "The sum of the planned costs of the activities in this campaign.", HelpCampaign},
	{"campaign.budgetUsed", "Budget used", "Planned cost divided by the campaign budget. Red above 100 %.", HelpCampaign},
	{"campaign.leftToPlan", "Left to plan", "Campaign budget minus planned cost. Below zero means the campaign is over budget.", HelpCampaign},
	{"campaign.approvalProgress", "Approval progress", "How many activities below are approved out of all of them.", HelpCampaign},
	{"campaign.timing", "Timing", "Not started, Running or Finished, compared with today's date.", HelpCampaign},
	{"campaign.goals", "Goals", "The sum of the goals of the activities below.", HelpCampaign},

	{"initiative.name", "Initiative name", "A group of similar work inside a campaign. Unique in its campaign.", HelpInitiative},
	{"initiative.campaign", "Campaign", "The campaign the initiative belongs to. Change it with Move.", HelpInitiative},
	{"initiative.owner", "Owner", "The person responsible for the initiative. It starts as the campaign's owner.", HelpInitiative},
	{"initiative.dates", "Dates", "When the initiative runs. Must be inside the campaign's dates; activities must fit inside these.", HelpInitiative},
	{"initiative.description", "Description", "What the initiative covers.", HelpInitiative},
	{"initiative.budget", "Initiative budget", "The part of the campaign budget given to this initiative, in the plan currency.", HelpInitiative},
	{"initiative.budgetPhasing", "Budget by month", "How the initiative budget is spread over the months of the plan's year.", HelpInitiative},
	{"initiative.remainingCampaignBudget", "Remaining campaign budget", "Campaign budget minus all initiative budgets, including this one. Red when initiatives have more than the campaign.", HelpInitiative},
	{"initiative.plannedCost", "Planned cost", "The sum of the planned costs of the activities in this initiative.", HelpInitiative},
	{"initiative.budgetUsed", "Budget used", "Planned cost divided by the initiative budget. Red above 100 %.", HelpInitiative},
	{"initiative.leftToPlan", "Left to plan", "Initiative budget minus planned cost. Below zero means the initiative is over budget.", HelpInitiative},
	{"initiative.approvalProgress", "Approval progress", "How many activities below are approved out of all of them.", HelpInitiative},
	{"initiative.timing", "Timing", "Not started, Running or Finished, compared with today's date.", HelpInitiative},
	{"initiative.goals", "Goals", "The sum of the goals of the activities below.", HelpInitiative},

	{"activity.name", "Activity name", `One real piece of work, such as "Launch film". Unique in its initiative.`, HelpActivity},
	{"activity.externalName", "External name", "The name connected systems use. Leave it empty to use the activity name.", HelpActivity},
	{"activity.initiative", "Campaign, Initiative", "Where the activity sits in the plan. Change it with Move.", HelpActivity},
	{"activity.activityType", "Activity type", "The kind of work. Only the types the plan's model allows are offered.", HelpActivity},
	{"activity.owner", "Owner", "The person responsible for the activity. Needed to submit.", HelpActivity},
	{"activity.dates", "Dates", "When the work happens. Must be inside the initiative's dates. Changing them on an approved activity restarts approval.", HelpActivity},
	{"activity.ongoing", "Ongoing (no end date)", "For work without an end date. It can't be undone once saved.", HelpActivity},
	{"activity.plannedCost", "Planned cost", "What the activity is expected to cost, in the plan currency. Line items divide it into costs. Changing it on an approved activity restarts approval.", HelpActivity},
	{"activity.approvalWorkflow", "Approval workflow", "The approval levels the activity goes through when submitted.", HelpActivity},
	{"activity.description", "Description", "What the activity is and why it is done.", HelpActivity},
	{"activity.status", "Status", "Created, Submitted, Declined, Approved, In progress or Complete.", HelpActivity},
	{"activity.unassignedPlannedCost", "Unassigned planned cost", "Planned cost not yet given to a line item: the activity's planned cost minus its line items. Below zero blocks submitting.", HelpActivity},
	{"activity.inq", "INQ (inquiries)", "The number of inquiries the activity is expected to bring. MQL and revenue are calculated with the plan's model.", HelpActivity},
	{"activity.setLaterStagesMyself", "Set later stages myself", "Type MQL and revenue yourself instead of calculating them with the model.", HelpActivity},
	{"activity.mql", "MQL", "Marketing-qualified leads: INQ × the model's INQ→MQL rate, unless typed.", HelpActivity},
	{"activity.revenue", "Revenue", "Expected revenue in USD: deals × the model's average deal size, unless typed.", HelpActivity},
	{"activity.splitFrom", "Split from / Split into", "The activity this one was split from, or the activities split off it.", HelpActivity},
	{"activity.fromFinance", "From finance (coming later)", "Values the Finance module and GSAP will fill.", HelpActivity},

	{"lineItem.name", "Line item name", `One cost of the activity, such as "Agency fee". Unique in its activity.`, HelpLineItem},
	{"lineItem.owner", "Owner", "The person responsible for this cost. It starts as the activity's owner.", HelpLineItem},
	{"lineItem.description", "Description", "What the cost is for.", HelpLineItem},
	{"lineItem.currency", "Currency", "The currency the cost is typed in. It is converted to the plan currency and USD at today's rate.", HelpLineItem},
	{"lineItem.plannedCost", "Planned cost", "The cost in the line item's currency. It starts as the activity's unassigned planned cost.", HelpLineItem},
	{"lineItem.plannedCostPlan", "Planned cost (plan currency)", "The planned cost converted into the plan currency at today's rate.", HelpLineItem},
	{"lineItem.plannedCostUsd", "Planned cost in USD", "The planned cost converted into USD at today's rate.", HelpLineItem},
	{"lineItem.spread", "Spread over time", "How the cost is spread over months: evenly, all in the first or last month, or month by month.", HelpLineItem},
	{"lineItem.contractedAmount", "Contracted amount", "The amount already agreed in a contract, in the line item's currency.", HelpLineItem},
	{"lineItem.requestSwbs", "Request SWBS code from GSAP", "Ask GSAP for an SWBS code once the activity is approved.", HelpLineItem},
	{"lineItem.swbsCode", "SWBS code", "Filled by the simulated GSAP once the activity is approved and the request is sent.", HelpLineItem},
	{"lineItem.financeBudgetYear", "Finance budget year", "The year of the finance budget this cost links to.", HelpLineItem},
	{"lineItem.financeBudgetItem", "Finance budget item", "The finance budget item this cost is charged to. Needed to submit.", HelpLineItem},
	{"lineItem.fromFinance", "From finance (coming later)", "Actuals, invoices and purchase requests, filled by Finance and GSAP later.", HelpLineItem},

	{"phasing.budget", "Budget", "The phased budget of each month or quarter.", HelpPhasing},
	{"phasing.plannedCost", "Planned cost", "The planned cost of each month, from the line items' spreads and the unassigned planned cost spread evenly.", HelpPhasing},
	{"phasing.difference", "Difference", "Budget minus planned cost. Red when the planned cost is higher.", HelpPhasing},
	{"phasing.allocated", "Allocated to campaigns / initiatives", "The phased budgets of the records below, month by month.", HelpPhasing},

	{"common.id", "ID", "The record's permanent ID, such as A-1012.", HelpCommon},
	{"common.completeness", "Completeness", `"All filled in", or how many fields are still needed before publishing or submitting.`, HelpCommon},
	{"common.created", "Created", "Who created the record and when.", HelpCommon},
	{"common.lastChanged", "Last changed", "Who changed the record last and when.", HelpCommon},

	{"allWork.campaigns", "Campaigns", "The campaigns that match the filters.", HelpAllWork},
	{"allWork.initiatives", "Initiatives", "The initiatives that match the filters.", HelpAllWork},
	{"allWork.activities", "Activities", "The activities that match the filters.", HelpAllWork},
	{"allWork.plannedCost", "Planned cost", "The sum of the matching activities' planned cost, in your display currency.", HelpAllWork},
	{"allWork.planBudget", "Plan budget", "The sum of the budgets of the plans shown, in your display currency.", HelpAllWork},
	{"allWork.unassignedPlannedCost", "Unassigned planned cost", "The sum of the matching activities' planned cost not yet given to line items.", HelpAllWork},
	{"allWork.breakdownShare", "Share", "The group's planned cost as a share of the total.", HelpAllWork},
}

// helpDefinitionOf returns the catalogue entry of key, and false when the catalogue has none.
func helpDefinitionOf(key string) (helpDefinition, bool) {
	for _, d := range helpCatalogue {
		if d.Key == key {
			return d, true
		}
	}
	return helpDefinition{}, false
}

// fieldHelpName is how the activity log names an entry: "Plan › Plan budget".
func fieldHelpName(key string) string {
	d, found := helpDefinitionOf(key)
	if !found {
		return key
	}
	return helpLevelNames[d.Level] + " › " + d.Label
}
