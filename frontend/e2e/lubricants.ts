import type { Page } from '@playwright/test';
import { callApi, uniqueName } from './fixtures';
import { idOf, type Named } from './planning';

// A Lubricants plan of next year with an activity ready to submit, prepared through the API.

export const STANDARD_WORKFLOW = 'Standard marketing approval';

interface FormAttribute {
  id: string;
  label: string;
  options: { id: string; label: string }[];
}

/** Every needed-to-submit activity attribute of a Lubricants plan, filled with a demo option. */
async function lubricantsAttributes(page: Page): Promise<Record<string, unknown>> {
  const form = await callApi<{ sections: { attributes: FormAttribute[] }[] }>(
    page,
    'GET',
    '/api/v1/shell-attribute-forms/activity',
  );
  const all = form.sections.flatMap((section) => section.attributes);
  const attribute = (label: string) => {
    const found = all.find((a) => a.label === label);
    if (!found) {
      throw new Error(`There is no activity attribute '${label}'`);
    }
    return found;
  };
  const option = (label: string, optionLabel?: string) => {
    const options = attribute(label).options;
    const found = optionLabel ? options.find((o) => o.label === optionLabel) : options[0];
    if (!found) {
      throw new Error(`'${label}' has no option '${optionLabel}'`);
    }
    return found.id;
  };
  return {
    [attribute('Channel / audience').id]: [option('Channel / audience', 'External')],
    [attribute('Country / OU').id]: [
      { optionId: option('Country / OU', 'China'), share: '100.00' },
    ],
    [attribute('Company code').id]: option('Company code', 'CN04'),
    [attribute('Cost centre').id]: option('Cost centre', 'CN04000220'),
    [attribute('Global campaign').id]: [option('Global campaign', 'Shell Recharge launch')],
    [attribute('Growth ambition').id]: [option('Growth ambition')],
    [attribute('PIR').id]: option('PIR', 'No'),
    [attribute('FIT4 initiative').id]: option('FIT4 initiative', 'Not FIT4'),
    [attribute('FIT4 ID').id]: 'FIT4-0999',
    [attribute('Brands').id]: [{ optionId: option('Brands', 'Helix'), share: '100.00' }],
    [attribute('Global key accounts').id]: [option('Global key accounts')],
    [attribute('Sector').id]: [option('Sector', 'Automotive')],
    [attribute('Sub-sector').id]: [option('Sub-sector', 'Passenger cars')],
    [attribute('Product launch').id]: option('Product launch', 'No'),
  };
}

export interface LubricantsActivity extends Named {
  version: number;
  path: string;
}

/**
 * As the signed-in person: a Lubricants plan of next year (team Lubricants Europe) with one campaign
 * and one initiative, and a Created activity of `amount` USD with every needed field filled, owned by
 * Priya Raman with the Standard marketing approval (017 seed), in USD, unless `options` say otherwise.
 */
export async function lubricantsActivity(
  page: Page,
  label: string,
  amount: string,
  options: { owner?: string; workflow?: string; currency?: string } = {},
): Promise<LubricantsActivity> {
  const workflowName = options.workflow ?? STANDARD_WORKFLOW;
  const currency = options.currency ?? 'USD';
  const year = new Date().getUTCFullYear() + 1;
  const [modelId, teamId, ownerId, planForm] = await Promise.all([
    idOf(page, '/api/v1/models', 'Lubricants 1.0'),
    idOf(page, '/api/v1/teams', 'Lubricants Europe'),
    idOf(page, '/api/v1/people', options.owner ?? 'Priya Raman'),
    callApi<{
      businessUnitAttributeId: string;
      businessUnitOptions: { id: string; label: string }[];
    }>(page, 'GET', '/api/v1/shell-attribute-forms/plan'),
  ]);
  const lubricants = planForm.businessUnitOptions.find((o) => o.label === 'Lubricants');
  if (!lubricants) {
    throw new Error('The plan form has no business unit Lubricants');
  }
  const plan = await callApi<Named>(page, 'POST', '/api/v1/plans', {
    name: uniqueName(`${label} plan`),
    year,
    modelId,
    teamIds: [teamId],
    ownerId,
    aop: false,
    description: '',
    budget: { amount: '100000.00', currency },
    currency,
    defaultFinanceBudgetId: null,
    defaultSpread: 'evenly',
    bcpir: false,
    goalStage: 'inq',
    goalAmount: null,
    attributes: { [planForm.businessUnitAttributeId]: [lubricants.id] },
  });
  const wholeYear = { ownerId, startDate: `${year}-01-01`, endDate: `${year}-12-31` };
  const campaign = await callApi<Named>(page, 'POST', `/api/v1/plans/${plan.id}/campaigns`, {
    name: uniqueName(`${label} campaign`),
    description: '',
    ...wholeYear,
    budget: { amount: '50000.00', currency },
    attributes: {},
  });
  const initiative = await callApi<Named>(
    page,
    'POST',
    `/api/v1/campaigns/${campaign.id}/initiatives`,
    {
      name: uniqueName(`${label} initiative`),
      description: '',
      ...wholeYear,
      budget: { amount: '40000.00', currency },
      attributes: {},
    },
  );
  const form = await callApi<{ activityTypes: Named[]; approvalWorkflows: Named[] }>(
    page,
    'GET',
    `/api/v1/initiatives/${initiative.id}/activity-form`,
  );
  const type = form.activityTypes.find((t) => t.name === 'Content production');
  const workflow = form.approvalWorkflows.find((w) => w.name === workflowName);
  if (!type || !workflow) {
    throw new Error(`The activity form offers no 'Content production' or '${workflowName}'`);
  }
  const activity = await callApi<Named & { version: number }>(
    page,
    'POST',
    `/api/v1/initiatives/${initiative.id}/activities`,
    {
      name: uniqueName(`${label} activity`),
      externalName: '',
      activityTypeId: type.id,
      ownerId,
      startDate: `${year}-03-01`,
      endDate: `${year}-05-31`,
      ongoing: false,
      plannedCost: { amount, currency },
      approvalWorkflowId: workflow.id,
      description: '',
      attributes: await lubricantsAttributes(page),
      goals: { inq: null, setLaterStagesMyself: false, mql: null, revenue: null },
    },
  );
  return { ...activity, path: `/plans/${plan.id}/activities/${activity.id}` };
}
