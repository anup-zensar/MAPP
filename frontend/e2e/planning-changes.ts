import type { Page } from '@playwright/test';
import { callApi, itemAt, signInAs, uniqueName } from './fixtures';
import type { Named, PlanningRecords } from './planning';

// Changes to prepared records through the API: decisions, planned costs and line items.

interface ApprovalRound {
  id: string;
  currentLevel: number;
}

/** Signs in as `email` and approves or declines the activity's current level. */
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
    `/api/v1/activities/${activityId}/approval`,
  );
  const round = itemAt(approval.rounds, 0);
  await callApi(page, 'POST', `/api/v1/activities/${activityId}/approval/${decision}`, {
    roundId: round.id,
    level: round.currentLevel,
    comment,
  });
}

/** As Mei Lin: submits the China BC activity; Carlos Mendes and Alex Morgan approve. Ends as Alex. */
export async function approveThroughApi(page: Page, records: PlanningRecords): Promise<void> {
  await callApi(page, 'POST', `/api/v1/activities/${records.activityId}/approval/submit`, {
    version: records.activityVersion,
    comment: '',
  });
  await decideAs(page, 'carlos.mendes@shell.com', records.activityId, 'approve');
  await decideAs(page, 'alex.morgan@shell.com', records.activityId, 'approve');
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

/** As the signed-in person: saves the activity with a new USD planned cost, as the edit form would. */
export async function changeActivityPlannedCost(
  page: Page,
  activityId: string,
  amount: string,
): Promise<void> {
  const activity = await callApi<ActivityAnswer>(page, 'GET', `/api/v1/activities/${activityId}`);
  const ownStages = activity.goals.setLaterStagesMyself;
  await callApi(page, 'PATCH', `/api/v1/activities/${activityId}`, {
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

export function lineItemFields(name: string, plannedCost: string, currency = 'USD') {
  return {
    name,
    ownerId: null,
    description: '',
    currency,
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
    `/api/v1/activities/${activityId}/line-items`,
    lineItemFields(uniqueName(`${label} line item`), plannedCost),
  );
}

export async function changeLineItemPlannedCost(
  page: Page,
  lineItem: Named & { version: number },
  plannedCost: string,
): Promise<void> {
  await callApi(page, 'PATCH', `/api/v1/line-items/${lineItem.id}`, {
    ...lineItemFields(lineItem.name, plannedCost),
    version: lineItem.version,
  });
}

/** Signs in as `email` and cancels that person's delegations in force today (017). */
export async function cancelCurrentDelegations(page: Page, email: string): Promise<void> {
  await signInAs(page, email);
  const list = await callApi<{ items: { id: string; state: string; version: number }[] }>(
    page,
    'GET',
    '/api/v1/approval-delegations',
  );
  for (const delegation of list.items.filter((item) => item.state === 'current')) {
    await callApi(page, 'POST', `/api/v1/approval-delegations/${delegation.id}/cancel`, {
      version: delegation.version,
    });
  }
}
