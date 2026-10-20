import type { Locator, Page } from '@playwright/test';
import { callApi, signInAs, uniqueName } from './fixtures';

// Records prepared through the API (contract: backend/api/openapi.yaml), for journeys that start from an
// existing plan or activity. What a journey proves is always done on screen; these only set the stage.

export const CHINA_WORKFLOW = 'China BC approval workflow';

export interface Named {
  id: string;
  name: string;
}

export interface ListOf<T> {
  items: T[];
  total: number;
}

interface FormOption {
  id: string;
  label: string;
}

interface AttributeForm {
  sections: { attributes: { id: string; label: string; options: FormOption[] }[] }[];
  businessUnitAttributeId: string | null;
  businessUnitOptions: FormOption[];
}

interface ActivityForm {
  activityTypes: Named[];
  approvalWorkflows: Named[];
}

export interface PlanningRecords {
  /** The plan's year: next year, so a plan copy into the following year is still allowed. */
  year: number;
  planId: string;
  planName: string;
  campaignId: string;
  campaignName: string;
  initiativeId: string;
  initiativeName: string;
  activityId: string;
  activityName: string;
  activityVersion: number;
  /** The activity page: /plans/<planId>/activities/<activityId>. */
  activityPath: string;
}

/** The needed-to-submit activity attributes (always shown) and the demo option each test picks. */
const ACTIVITY_ATTRIBUTES = [
  'Channel / audience',
  'Country / OU',
  'Company code',
  'Cost centre',
  'Global campaign',
  'Growth ambition',
  'PIR',
  'FIT4 initiative',
  'FIT4 ID',
] as const;

export type ActivityAttribute = (typeof ACTIVITY_ATTRIBUTES)[number];

/** Finds the record called `name` in a list answer of the API. */
export async function idOf(page: Page, url: string, name: string): Promise<string> {
  const list = await callApi<ListOf<Named>>(page, 'GET', url);
  const found = list.items.find((item) => item.name === name);
  if (!found) {
    throw new Error(`${url} has no record called '${name}'`);
  }
  return found.id;
}

/**
 * The values of the needed-to-submit activity attributes of a Mobility plan, every one filled except
 * `leaveEmpty` (Mobility's own attribute group is optional, so these are all that is needed).
 */
async function activityAttributeValues(
  page: Page,
  leaveEmpty: readonly ActivityAttribute[],
): Promise<Record<string, unknown>> {
  const form = await callApi<AttributeForm>(page, 'GET', '/api/shell-attribute-forms/activity');
  const attributes = form.sections.flatMap((section) => section.attributes);
  const attribute = (label: string) => {
    const found = attributes.find((a) => a.label === label);
    if (!found) {
      throw new Error(`There is no activity attribute '${label}'`);
    }
    return found;
  };
  const option = (label: string, optionLabel: string) => {
    const found = attribute(label).options.find((o) => o.label === optionLabel);
    if (!found) {
      throw new Error(`'${label}' has no option '${optionLabel}'`);
    }
    return found.id;
  };
  const values: Record<ActivityAttribute, () => unknown> = {
    'Channel / audience': () => [option('Channel / audience', 'External')],
    'Country / OU': () => [{ optionId: option('Country / OU', 'China'), share: '100.00' }],
    'Company code': () => option('Company code', 'CN04'),
    'Cost centre': () => option('Cost centre', 'CN04000220'),
    'Global campaign': () => [option('Global campaign', 'Shell Recharge launch')],
    'Growth ambition': () => [option('Growth ambition', 'MOBILITY – Integrated offer')],
    PIR: () => option('PIR', 'No'),
    'FIT4 initiative': () => option('FIT4 initiative', 'Not FIT4'),
    'FIT4 ID': () => 'FIT4-0999',
  };
  return Object.fromEntries(
    ACTIVITY_ATTRIBUTES.filter((label) => !leaveEmpty.includes(label)).map((label) => [
      attribute(label).id,
      values[label](),
    ]),
  );
}

/**
 * As the signed-in person (Mei Lin): a Draft plan of next year called "<label> plan <unique>" (Mobility
 * 1.0, team Mobility China, owner Mei Lin, business unit Mobility, budget 100,000.00 USD), without
 * campaigns.
 *
 * Next year, so an activity in it starts after today: once approved it shows "Approved", not "In
 * progress" (006 FR-021). The plan form accepts years up to the current year + 5.
 */
export async function createPlan(page: Page, label: string): Promise<Named & { year: number }> {
  const year = new Date().getUTCFullYear() + 1;
  const [modelId, teamId, ownerId, planForm] = await Promise.all([
    idOf(page, '/api/models', 'Mobility 1.0'),
    idOf(page, '/api/teams', 'Mobility China'),
    idOf(page, '/api/people', 'Mei Lin'),
    callApi<AttributeForm>(page, 'GET', '/api/shell-attribute-forms/plan'),
  ]);
  const mobility = planForm.businessUnitOptions.find((o) => o.label === 'Mobility');
  if (!planForm.businessUnitAttributeId || !mobility) {
    throw new Error('The plan form has no business unit Mobility');
  }
  const plan = await callApi<Named>(page, 'POST', '/api/plans', {
    name: uniqueName(`${label} plan`),
    year,
    modelId,
    teamIds: [teamId],
    ownerId,
    aop: false,
    description: '',
    budget: { amount: '100000.00', currency: 'USD' },
    defaultFinanceBudgetId: null,
    defaultSpread: 'evenly',
    bcpir: false,
    goalStage: 'inq',
    goalAmount: null,
    attributes: { [planForm.businessUnitAttributeId]: [mobility.id] },
  });
  return { ...plan, year };
}

/**
 * As the signed-in person (Mei Lin): an initiative "<label> initiative <unique>" in the campaign, for
 * the whole of `year`, owned by Mei Lin, with the budget in USD (such as '40000.00').
 */
export async function createInitiative(
  page: Page,
  campaignId: string,
  label: string,
  year: number,
  budget: string,
): Promise<Named> {
  const ownerId = await idOf(page, '/api/people', 'Mei Lin');
  return callApi<Named>(page, 'POST', `/api/campaigns/${campaignId}/initiatives`, {
    name: uniqueName(`${label} initiative`),
    description: '',
    ownerId,
    startDate: `${year}-01-01`,
    endDate: `${year}-12-31`,
    budget: { amount: budget, currency: 'USD' },
    attributes: {},
  });
}

/**
 * As the signed-in person (Mei Lin): a plan of next year (see createPlan) with one campaign (50,000.00),
 * one initiative (40,000.00) and one Created activity "Content production" of 20,000.00 USD from 1 Mar
 * to 31 May.
 */
export async function createPlanningRecords(
  page: Page,
  label: string,
  activity: { withWorkflow: boolean; leaveEmpty: readonly ActivityAttribute[] },
): Promise<PlanningRecords> {
  const plan = await createPlan(page, label);
  const { year } = plan;
  const ownerId = await idOf(page, '/api/people', 'Mei Lin');

  const wholeYear = { ownerId, startDate: `${year}-01-01`, endDate: `${year}-12-31` };
  const campaignName = uniqueName(`${label} campaign`);
  const campaign = await callApi<Named>(page, 'POST', `/api/plans/${plan.id}/campaigns`, {
    name: campaignName,
    description: '',
    ...wholeYear,
    budget: { amount: '50000.00', currency: 'USD' },
    attributes: {},
  });
  const initiative = await createInitiative(page, campaign.id, label, year, '40000.00');

  const form = await callApi<ActivityForm>(
    page,
    'GET',
    `/api/initiatives/${initiative.id}/activity-form`,
  );
  const type = form.activityTypes.find((t) => t.name === 'Content production');
  const workflow = form.approvalWorkflows.find((w) => w.name === CHINA_WORKFLOW);
  if (!type || !workflow) {
    throw new Error(`The activity form offers no 'Content production' or '${CHINA_WORKFLOW}'`);
  }
  const activityName = uniqueName(`${label} activity`);
  const created = await callApi<Named & { version: number }>(
    page,
    'POST',
    `/api/initiatives/${initiative.id}/activities`,
    {
      name: activityName,
      externalName: '',
      activityTypeId: type.id,
      ownerId,
      startDate: `${year}-03-01`,
      endDate: `${year}-05-31`,
      ongoing: false,
      plannedCost: { amount: '20000.00', currency: 'USD' },
      approvalWorkflowId: activity.withWorkflow ? workflow.id : null,
      description: '',
      attributes: await activityAttributeValues(page, activity.leaveEmpty),
      goals: { inq: null, setLaterStagesMyself: false, mql: null, revenue: null },
    },
  );

  return {
    year,
    planId: plan.id,
    planName: plan.name,
    campaignId: campaign.id,
    campaignName,
    initiativeId: initiative.id,
    initiativeName: initiative.name,
    activityId: created.id,
    activityName,
    activityVersion: created.version,
    activityPath: `/plans/${plan.id}/activities/${created.id}`,
  };
}

interface ApprovalRound {
  id: string;
  currentLevel: number;
}

/**
 * Signs in as `email` and approves or declines the activity's current level through the API (the screen
 * journeys of deciding are in core-flow.spec.ts).
 */
export async function decideAs(
  page: Page,
  email: string,
  activityId: string,
  decision: 'approve' | 'decline',
  comment = '',
): Promise<void> {
  await signInAs(page, email);
  const approval = await callApi<{ rounds: ApprovalRound[] }>(
    page,
    'GET',
    `/api/activities/${activityId}/approval`,
  );
  const [round] = approval.rounds;
  await callApi(page, 'POST', `/api/activities/${activityId}/approval/${decision}`, {
    roundId: round.id,
    level: round.currentLevel,
    comment,
  });
}

/**
 * As Mei Lin: submits the activity (made with the China BC approval workflow) and has Carlos Mendes and
 * Alex Morgan approve both levels. Ends signed in as Alex Morgan.
 */
export async function approveThroughApi(page: Page, records: PlanningRecords): Promise<void> {
  await callApi(page, 'POST', `/api/activities/${records.activityId}/approval/submit`, {
    version: records.activityVersion,
    comment: '',
  });
  await decideAs(page, 'carlos.mendes@northstar.demo', records.activityId, 'approve');
  await decideAs(page, 'alex.morgan@northstar.demo', records.activityId, 'approve');
}

/** The readable ID in `url` captured by the first group of `pattern`. */
export function idIn(url: string, pattern: RegExp): string {
  const match = pattern.exec(url);
  if (!match) {
    throw new Error(`No ID in ${url}`);
  }
  return match[1];
}

/** The plan page's tree (004's TreeTable, a treegrid). */
export function planTree(page: Page): Locator {
  // PlanTree names its TreeTable after the card's title.
  return page.getByRole('treegrid', { name: 'Plan tree' });
}

/** The tree row of the plan, campaign, initiative or activity called `name`. */
export function treeRow(page: Page, name: string): Locator {
  return planTree(page).getByRole('row').filter({ hasText: name });
}

interface ActivityAnswer {
  name: string;
  externalName: string;
  activityType: { id: string };
  owner: { id: string } | null;
  startDate: string | null;
  endDate: string | null;
  ongoing: boolean;
  approvalWorkflow: { id: string } | null;
  description: string;
  attributes: Record<string, unknown>;
  goals: {
    inq: number | null;
    setLaterStagesMyself: boolean;
    mql: number | null;
    revenue: unknown;
  };
  version: number;
}

/**
 * As the signed-in person: saves the activity with a new planned cost in USD (such as '25000.00') and
 * everything else as it is, as the edit form would (006 updateActivity).
 */
export async function changeActivityPlannedCost(
  page: Page,
  activityId: string,
  amount: string,
): Promise<void> {
  const activity = await callApi<ActivityAnswer>(page, 'GET', `/api/activities/${activityId}`);
  const ownStages = activity.goals.setLaterStagesMyself;
  await callApi(page, 'PATCH', `/api/activities/${activityId}`, {
    name: activity.name,
    externalName: activity.externalName,
    activityTypeId: activity.activityType.id,
    ownerId: activity.owner?.id ?? null,
    startDate: activity.startDate,
    endDate: activity.endDate,
    ongoing: activity.ongoing,
    plannedCost: { amount, currency: 'USD' },
    approvalWorkflowId: activity.approvalWorkflow?.id ?? null,
    description: activity.description,
    attributes: activity.attributes,
    goals: {
      inq: activity.goals.inq,
      setLaterStagesMyself: ownStages,
      mql: ownStages ? activity.goals.mql : null,
      revenue: ownStages ? activity.goals.revenue : null,
    },
    version: activity.version,
    restartApproval: false,
  });
}

/** A line item's fields that a test sets; everything else stays empty (007 FR-009). */
function lineItemFields(name: string, plannedCost: string) {
  return {
    name,
    ownerId: null,
    description: '',
    currency: 'USD',
    plannedCost,
    spread: { method: 'evenly', months: [] },
    attributes: {},
    contractedAmount: null,
    requestSwbs: false,
    financeBudgetYear: null,
    financeBudgetItemId: null,
  };
}

/** As the signed-in person: a USD line item "<label> line item <unique>" of `plannedCost` in the activity. */
export async function createLineItem(
  page: Page,
  activityId: string,
  label: string,
  plannedCost: string,
): Promise<Named & { version: number }> {
  return callApi<Named & { version: number }>(
    page,
    'POST',
    `/api/activities/${activityId}/line-items`,
    lineItemFields(uniqueName(`${label} line item`), plannedCost),
  );
}

/** As the signed-in person: saves the line item with a new planned cost and everything else empty. */
export async function changeLineItemPlannedCost(
  page: Page,
  lineItem: Named & { version: number },
  plannedCost: string,
): Promise<void> {
  await callApi(page, 'PATCH', `/api/line-items/${lineItem.id}`, {
    ...lineItemFields(lineItem.name, plannedCost),
    version: lineItem.version,
  });
}
