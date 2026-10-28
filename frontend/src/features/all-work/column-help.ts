// The Field help entry of each All work column and of the Breakdown's Share, for their ⓘ tips.

const SAME_KEY = new Set([
  'plan.year',
  'plan.model',
  'plan.teams',
  'plan.description',
  'plan.defaultFinanceBudget',
  'plan.defaultSpread',
  'plan.bcpir',
  'plan.goalStage',
  'plan.goalAmount',
  'campaign.description',
  'initiative.description',
  'activity.externalName',
  'activity.ongoing',
  'activity.approvalWorkflow',
  'activity.description',
  'lineItem.description',
  'lineItem.currency',
  'lineItem.spread',
  'lineItem.contractedAmount',
  'lineItem.requestSwbs',
  'lineItem.financeBudgetYear',
  'lineItem.financeBudgetItem',
]);

const OTHER_KEY: Readonly<Record<string, string>> = {
  id: 'common.id',
  completeness: 'common.completeness',
  createdAt: 'common.created',
  updatedAt: 'common.lastChanged',
  planBudget: 'allWork.planBudget',
  plannedCost: 'allWork.plannedCost',
  unassigned: 'allWork.unassignedPlannedCost',
  activities: 'allWork.activities',
  remainingBudget: 'plan.remainingBudget',
  budgetUsed: 'plan.budgetUsed',
  timing: 'campaign.timing',
  year: 'plan.year',
  planType: 'plan.planType',
  planCurrency: 'plan.currency',
  planBudgetPlan: 'plan.budget',
  plannedCostPlan: 'lineItem.plannedCostPlan',
  inq: 'activity.inq',
  mql: 'activity.mql',
  revenue: 'activity.revenue',
  'plan.strategic': 'plan.aop',
  'activity.type': 'activity.activityType',
  'activity.setLaterStages': 'activity.setLaterStagesMyself',
  'lineItem.localPlannedCost': 'lineItem.plannedCost',
  'lineItem.swbsStatus': 'lineItem.swbsCode',
};

/** The Breakdown table's Share column. */
export const BREAKDOWN_SHARE_HELP_KEY = 'allWork.breakdownShare';

/** The Field help key explaining an All work column; undefined for a Shell attribute or a plain one. */
export function columnHelpKey(columnKey: string): string | undefined {
  return SAME_KEY.has(columnKey) ? columnKey : OTHER_KEY[columnKey];
}
