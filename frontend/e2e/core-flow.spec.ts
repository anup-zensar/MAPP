import { expect, test, type Locator, type Page } from '@playwright/test';
import {
  action,
  callApi,
  chooseOption,
  escapeRegExp,
  expectNoAxeViolations,
  expectToast,
  signInAs,
  uniqueName,
} from './fixtures';

// End-to-end journeys through the core flow of the Plan module (features 002–008), against the real API
// with the demo data: plan → campaign → initiative → activity → line item, approval level by level,
// Settings, the owner rules of the plan form, delete with Undo, and axe on the main pages.
//
// Every text and label comes from the "User-visible messages" tables of specs/002–008 plan.md. Where a
// plan leaves a selector open, the most likely one is used and marked "// CHECK: <what to verify>".
//
// Tests create their own records (names from uniqueName) and never assert global counts. The demo data
// they read (backend/seed): people and teams (Mei Lin and Carlos Mendes in Mobility China; Priya Raman,
// Daniel Okoro and Sara Novak in Lubricants Europe; Alex Morgan in Global Marketing Ops), the models
// Lubricants 1.0 and Mobility 1.0, the Shell attributes and options, the currencies (EUR 0.92, INR 83.50),
// the approval workflow "China BC approval workflow" (level 1 Carlos Mendes, level 2 Alex Morgan), and
// for the accessibility test FY2026 Lubricants Europe (P-1001) with Launch film (A-1001).

const ALEX = 'alex.morgan@northstar.demo';
const PRIYA = 'priya.raman@northstar.demo';
const DANIEL = 'daniel.okoro@northstar.demo';
const MEI = 'mei.lin@northstar.demo';
const CARLOS = 'carlos.mendes@northstar.demo';

const CHINA_WORKFLOW = 'China BC approval workflow';

// ---------------------------------------------------------------------------------------------------
// Records prepared through the API (contract: backend/api/openapi.yaml), for journeys that start from
// an existing activity. What a journey proves is always done on screen.

interface Named {
  id: string;
  name: string;
}

interface ListOf<T> {
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

interface PlanningRecords {
  planId: string;
  planName: string;
  campaignName: string;
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

type ActivityAttribute = (typeof ACTIVITY_ATTRIBUTES)[number];

/** Finds the record called `name` in a list answer of the API. */
async function idOf(page: Page, url: string, name: string): Promise<string> {
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
 * As the signed-in person (Mei Lin): a Draft plan of next year (Mobility 1.0, team Mobility China,
 * business unit Mobility, budget 100,000.00 USD) with one campaign (50,000.00), one initiative
 * (40,000.00) and one Created activity "Content production" of 20,000.00 USD from 1 Mar to 31 May.
 *
 * Next year, so the activity starts after today: once approved it shows "Approved", not "In progress"
 * (006 FR-021). The plan form accepts years up to the current year + 5.
 */
async function createPlanningRecords(
  page: Page,
  label: string,
  activity: { withWorkflow: boolean; leaveEmpty: readonly ActivityAttribute[] },
): Promise<PlanningRecords> {
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

  const planName = uniqueName(`${label} plan`);
  const plan = await callApi<Named>(page, 'POST', '/api/plans', {
    name: planName,
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

  const wholeYear = { ownerId, startDate: `${year}-01-01`, endDate: `${year}-12-31` };
  const campaignName = uniqueName(`${label} campaign`);
  const campaign = await callApi<Named>(page, 'POST', `/api/plans/${plan.id}/campaigns`, {
    name: campaignName,
    description: '',
    ...wholeYear,
    budget: { amount: '50000.00', currency: 'USD' },
    attributes: {},
  });
  const initiativeName = uniqueName(`${label} initiative`);
  const initiative = await callApi<Named>(
    page,
    'POST',
    `/api/campaigns/${campaign.id}/initiatives`,
    {
      name: initiativeName,
      description: '',
      ...wholeYear,
      budget: { amount: '40000.00', currency: 'USD' },
      attributes: {},
    },
  );

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
    planId: plan.id,
    planName,
    campaignName,
    initiativeName,
    activityId: created.id,
    activityName,
    activityVersion: created.version,
    activityPath: `/plans/${plan.id}/activities/${created.id}`,
  };
}

/** The readable ID in `url` captured by the first group of `pattern`. */
function idIn(url: string, pattern: RegExp): string {
  const match = pattern.exec(url);
  if (!match) {
    throw new Error(`No ID in ${url}`);
  }
  return match[1];
}

/** The plan page's tree (004's TreeTable, a treegrid). */
function planTree(page: Page): Locator {
  // CHECK: the TreeTable `label` given by PlanTree is "Plan tree" (the card's title).
  return page.getByRole('treegrid', { name: 'Plan tree' });
}

/** The tree row of the plan, campaign, initiative or activity called `name`. */
function treeRow(page: Page, name: string): Locator {
  return planTree(page).getByRole('row').filter({ hasText: name });
}

/** Opens the Approvals page as `email`, finds the activity and opens its preview. */
async function openApprovalPreview(
  page: Page,
  email: string,
  activityName: string,
): Promise<Locator> {
  await signInAs(page, email);
  await page.goto('/approvals');
  await expect(page.getByRole('heading', { name: 'Approvals', level: 1 })).toBeVisible();
  // "Waiting for me" is the default tab (008 plan, Routes).
  // CHECK: 002's SearchField is an <input type="search"> named by its label.
  await page.getByRole('searchbox', { name: 'Search activity name or ID' }).fill(activityName);
  // The Activity column's name is a button that opens the preview (008 plan, Approvals page).
  await page.getByRole('button', { name: activityName, exact: true }).click();
  // CHECK: the preview SidePanel's title is the activity name.
  return page.getByRole('dialog', { name: activityName });
}

// ---------------------------------------------------------------------------------------------------

test('A: a planner builds a plan down to a line item in euros and sees the cost roll up', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, MEI);
  const planName = uniqueName('Core plan');
  const campaignName = uniqueName('Core campaign');
  const initiativeName = uniqueName('Core initiative');
  const activityName = uniqueName('Core activity');
  const lineItemName = uniqueName('Core line item');

  // The plan (004 US1): Mei's team and Mei as owner are the defaults; model and business unit are chosen.
  await page.goto('/plans');
  await action(page, 'New plan').click();
  await expect(page).toHaveURL(/\/plans\/new$/);
  await expect(page.getByRole('heading', { name: 'New plan', level: 1 })).toBeVisible();
  await expect(page.getByText('Default: your teams')).toBeVisible();
  await expect(page.getByText('Default: you', { exact: true })).toBeVisible();
  await page.getByRole('textbox', { name: /^Plan name/ }).fill(planName);
  // CHECK: Model is 002's ChoiceField, a native <select> for two models.
  await chooseOption(page.getByLabel(/^Model/), 'Mobility 1.0');
  // CHECK: Business unit (multiple choice, 2 options) is a check-box group; "Mobility" is unique here.
  await page.getByRole('checkbox', { name: 'Mobility', exact: true }).check();
  // CHECK: 002's MoneyField is a textbox named "Plan budget".
  await page.getByRole('textbox', { name: /^Plan budget/ }).fill('100000');
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Plan '${planName}' created — add campaigns next.`);
  await expect(page).toHaveURL(/\/plans\/P-\d+$/);
  const planId = idIn(page.url(), /\/plans\/(P-\d+)$/);

  // Open it again from the Plans list (004 US2).
  await page.goto('/plans');
  await page.getByRole('searchbox', { name: 'Search name or ID' }).fill(planName);
  await page.getByRole('link', { name: planName, exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`/plans/${planId}$`));
  await expect(page.getByRole('heading', { name: planName, level: 1 })).toBeVisible();
  await expect(page.getByText('Draft', { exact: true }).first()).toBeVisible();
  await expect(page.getByText('All filled in').first()).toBeVisible();
  await expect(
    page.getByText('No campaigns yet — a campaign is a theme that runs for most of the year.'),
  ).toBeVisible();

  // A campaign (005 US1): owner and dates come from the plan; one cluster fills its only region.
  await action(page, 'Add campaign').click();
  const campaignPanel = page.getByRole('dialog', { name: 'New campaign' });
  await expect(campaignPanel.getByText(`Inside plan '${planName}'`)).toBeVisible();
  await expect(campaignPanel.getByText('From the plan')).toBeVisible();
  await expect(campaignPanel.getByText("The plan's year")).toBeVisible();
  await campaignPanel.getByRole('textbox', { name: /^Campaign name/ }).fill(campaignName);
  await campaignPanel.getByRole('textbox', { name: /^Campaign budget/ }).fill('50000');
  // CHECK: Cluster (5 options) is a check-box group; Region is filled in because it has one option here.
  await campaignPanel.getByRole('checkbox', { name: 'Retail – China', exact: true }).check();
  // CHECK: 002's auto-fill message puts the labels mid-sentence in lower case ("region", "cluster").
  await expect(
    campaignPanel.getByText("Region set to 'RETAIL – CHINA', the only region for this cluster."),
  ).toBeVisible();
  await campaignPanel.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Campaign '${campaignName}' created`);
  await expect(treeRow(page, campaignName)).toBeVisible();

  // An initiative (005 US2), from the campaign row's ⋯ menu.
  await page.getByRole('button', { name: `Actions for ${campaignName}` }).click();
  await page.getByRole('menuitem', { name: 'Add initiative' }).click();
  const initiativePanel = page.getByRole('dialog', { name: 'New initiative' });
  await expect(initiativePanel.getByText(`Inside campaign '${campaignName}'`)).toBeVisible();
  await expect(initiativePanel.getByText('From the campaign')).toBeVisible();
  await initiativePanel.getByRole('textbox', { name: /^Initiative name/ }).fill(initiativeName);
  await initiativePanel.getByRole('textbox', { name: /^Initiative budget/ }).fill('40000');
  await initiativePanel.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Initiative '${initiativeName}' created`);
  await expect(treeRow(page, initiativeName)).toBeVisible();

  // An activity (006 US1), from the initiative row's ⋯ menu (006 adds "Add activity" there).
  await page.getByRole('button', { name: `Actions for ${initiativeName}` }).click();
  await page.getByRole('menuitem', { name: 'Add activity' }).click();
  await expect(page.getByRole('heading', { name: 'New activity', level: 1 })).toBeVisible();
  await expect(
    page.getByText(`In initiative '${initiativeName}' of campaign '${campaignName}' · ${planName}`),
  ).toBeVisible();
  await expect(page.getByText('From the initiative').first()).toBeVisible();
  await page.getByRole('textbox', { name: /^Activity name/ }).fill(activityName);
  // CHECK: Activity type is a native <select> whose options read "<type>" or "<type> · default <amount>".
  await chooseOption(page.getByLabel(/^Activity type/), 'Content production');
  await page.getByRole('textbox', { name: /^Planned cost/ }).fill('20000');
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Activity '${activityName}' created`);
  await expect(page).toHaveURL(new RegExp(`/plans/${planId}/activities/A-\\d+$`));
  const activityId = idIn(page.url(), /\/activities\/(A-\d+)$/);
  await expect(page.getByRole('heading', { name: activityName, level: 1 })).toBeVisible();
  await expect(page.getByText('Created', { exact: true }).first()).toBeVisible();
  await expect(
    page.getByText(
      "No line items yet — split this activity's planned cost into the costs you will pay",
    ),
  ).toBeVisible();

  // A line item in euros (007 US2): the planned cost starts as the unassigned 20,000.00 USD and follows
  // the currency; 9,200.00 EUR is 10,000.00 USD at the demo rate 0.92.
  await action(page, 'Add line item').click();
  await expect(page).toHaveURL(new RegExp(`/activities/${activityId}/line-items/new$`));
  await expect(page.getByRole('heading', { name: 'New line item', level: 1 })).toBeVisible();
  await expect(page.getByText(`In activity '${activityName}' (${activityId})`)).toBeVisible();
  await page.getByRole('textbox', { name: /^Line item name/ }).fill(lineItemName);
  // CHECK: Currency (5 active currencies) is a native <select> with options "EUR – Euro" …
  await chooseOption(page.getByLabel(/^Currency/), 'EUR');
  await expect(page.getByText('Changed to 18,400.00 EUR = 20,000.00 USD')).toBeVisible();
  // CHECK: the line item's Planned cost (002 MoneyField with the currency inside) is named "Planned cost".
  await page.getByRole('textbox', { name: /^Planned cost/ }).fill('9200');
  await expect(page.getByText('= 10,000.00 USD at 0.92 EUR per USD')).toBeVisible();
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Line item '${lineItemName}' created`);
  await expect(page).toHaveURL(new RegExp(`/activities/${activityId}#line-items$`));

  // The line item's USD cost counts against the activity's planned cost (007 FR-002).
  // The section's table comes before the spread grid, which repeats the name.
  const lineItemRow = page.getByRole('row').filter({ hasText: lineItemName }).first();
  await expect(lineItemRow).toContainText('9,200.00 EUR');
  // CHECK: the USD column reads "10,000.00 USD" (007-P4) and not "$10,000.00" (003's money helper).
  await expect(lineItemRow).toContainText('10,000.00 USD');
  await expect(page.getByText('10,000.00 of 20,000.00 USD assigned')).toBeVisible();
  // CHECK: the balance message uses the same "<amount> USD" form.
  await expect(
    page.getByText('10,000.00 USD not yet assigned to a line item').first(),
  ).toBeVisible();

  // What the activity still needs (006 FR-022 – FR-024): the approval workflow and the nine always-shown
  // needed-to-submit attributes (Mobility's own group is optional); its line item has no finance coding.
  await expect(page.getByText('Created', { exact: true }).first()).toBeVisible();
  // CHECK: 10 = Approval workflow + 9 attributes; owner, dates and planned cost are filled by defaults.
  await expect(page.getByText('10 fields needed').first()).toBeVisible();
  // CHECK: the line item lacks Spend type and Finance budget item (and GL code unless D-034 skips a
  // dependent attribute whose parent is empty), so the chip adds "1 line item needs attention".
  await expect(page.getByText('1 line item needs attention').first()).toBeVisible();
  await expect(
    page.getByRole('link', { name: 'Approval workflow — needed to submit' }),
  ).toBeVisible();
  await expect(lineItemRow).toContainText(/\d+ fields? needed/);

  // The activity's planned cost rolls up to the initiative, the campaign and the plan (006 FR-005,
  // 005 FR-011, 004 FR-012); budget used = 20,000 of 40,000, 50,000 and 100,000.
  await page
    .getByRole('navigation', { name: 'Breadcrumb' })
    .getByRole('link', { name: planName })
    .click();
  await expect(page).toHaveURL(new RegExp(`/plans/${planId}$`));
  await page.getByRole('button', { name: 'Expand all' }).click();
  // CHECK: tree amounts use the full money format ("$20,000.00"), not the compact one ("$20K").
  await expect(treeRow(page, activityName)).toContainText('20,000.00');
  await expect(treeRow(page, activityName)).toContainText('Created');
  await expect(treeRow(page, activityName)).toContainText('10 fields needed');
  await expect(treeRow(page, initiativeName)).toContainText('20,000.00');
  await expect(treeRow(page, initiativeName).getByRole('progressbar')).toHaveAttribute(
    'aria-valuenow',
    '50',
  );
  await expect(treeRow(page, campaignName)).toContainText('20,000.00');
  await expect(treeRow(page, campaignName).getByRole('progressbar')).toHaveAttribute(
    'aria-valuenow',
    '40',
  );
  // The campaign's own fields are complete (cluster and region set); something below is not.
  await expect(treeRow(page, campaignName)).toContainText('All filled in');
  await expect(treeRow(page, planName)).toContainText('20,000.00');
  await expect(treeRow(page, planName).getByRole('progressbar')).toHaveAttribute(
    'aria-valuenow',
    '20',
  );
  await expect(treeRow(page, planName)).toContainText('Draft');
  // CHECK: the plan header's CompletenessChip itemsBelow counts the activity and its line item.
  await expect(page.getByText(/\d+ items? below needs? attention/).first()).toBeVisible();
});

test('B: the planner completes and submits an activity; Carlos Mendes and Alex Morgan approve it level by level', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, MEI);
  // Everything but the approval workflow and three attributes is filled in already.
  const records = await createPlanningRecords(page, 'Approve', {
    withWorkflow: false,
    leaveEmpty: ['Channel / audience', 'Country / OU', 'Cost centre'],
  });
  const name = records.activityName;

  // What is missing (006 US2).
  await page.goto(records.activityPath);
  await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible();
  await expect(page.getByText('Created', { exact: true }).first()).toBeVisible();
  await expect(page.getByText('4 fields needed').first()).toBeVisible();
  for (const label of ['Approval workflow', 'Channel / audience', 'Country / OU', 'Cost centre']) {
    await expect(page.getByRole('link', { name: `${label} — needed to submit` })).toBeVisible();
  }
  // The header's button comes first; the Approval section repeats it.
  await page.getByRole('button', { name: 'Submit for approval' }).first().click();
  // CHECK: 008 adds no second workflow problem when 006 already listed the field (008 plan, Submission).
  await expect(page.getByText('Submitting needs 4 things — view list')).toBeVisible();

  // The planner completes it: a panel link opens the form at that field (006 FR-023).
  await page.getByRole('link', { name: 'Approval workflow — needed to submit' }).click();
  await expect(page).toHaveURL(
    new RegExp(`/activities/${records.activityId}/edit\\?field=approvalWorkflowId$`),
  );
  await expect(
    page.getByRole('heading', { name: `Edit activity '${name}'`, level: 1 }),
  ).toBeVisible();
  // CHECK: Approval workflow is a native <select> (two active workflows, no default).
  await chooseOption(page.getByLabel(/^Approval workflow/), CHINA_WORKFLOW);
  // CHECK: multiple-choice attributes with up to 10 options are check-box groups; these option names
  // are unique on the activity form ("External" of Channel / audience, "China" of Country / OU).
  await page.getByRole('checkbox', { name: 'External', exact: true }).check();
  await page.getByRole('checkbox', { name: 'China', exact: true }).check();
  // One country takes the whole split (002 SplitEditor: adding a value splits evenly again).
  await expect(page.getByText('100.00 % — complete')).toBeVisible();
  // CHECK: Cost centre (8 options) is a native <select>.
  await chooseOption(page.getByLabel(/^Cost centre/), 'CN04000220');
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Activity '${name}' saved`);
  await expect(page).toHaveURL(new RegExp(`${escapeRegExp(records.activityPath)}$`));
  await expect(page.getByText('All filled in').first()).toBeVisible();

  // Submit (008 US1): the dialog names the workflow and both levels.
  await page.getByRole('button', { name: 'Submit for approval' }).first().click();
  const submit = page.getByRole('alertdialog', { name: `Submit '${name}' for approval?` });
  await expect(submit.getByText(`Approval workflow: ${CHINA_WORKFLOW}`)).toBeVisible();
  await expect(submit.getByText('Carlos Mendes')).toBeVisible();
  await expect(submit.getByText('Alex Morgan')).toBeVisible();
  // No line items: the whole planned cost is unassigned, which is advice only (Q-008).
  await expect(
    submit.getByText(/of the planned cost is not yet assigned to a line item\.$/),
  ).toBeVisible();
  await submit.getByRole('button', { name: 'Submit for approval' }).click();
  await expectToast(page, `Activity '${name}' submitted for approval`);
  await expect(
    page.getByText(
      'Submitted for approval — changing planned cost or dates will restart approval.',
    ),
  ).toBeVisible();

  // Level 1: Carlos Mendes approves on the Approvals page (008 US3).
  let preview = await openApprovalPreview(page, CARLOS, name);
  await expect(preview.getByText(`${records.activityId} · Waiting at level 1 of 2`)).toBeVisible();
  await preview.getByRole('button', { name: 'Approve', exact: true }).click();
  let approve = page.getByRole('alertdialog', { name: `Approve '${name}'?` });
  await expect(approve.getByText('Level 1 of 2')).toBeVisible();
  // CHECK: "Next: level 2 · Alex Morgan." is one text (the approvers may be drawn as separate chips).
  await expect(approve.getByText('Next: level 2 · Alex Morgan.')).toBeVisible();
  await approve.getByRole('button', { name: 'Approve', exact: true }).click();
  await expectToast(page, `Level 1 approved — '${name}' now waits for Alex Morgan (level 2).`);

  // Level 2: Alex Morgan approves; the activity becomes Approved (008 US2).
  preview = await openApprovalPreview(page, ALEX, name);
  await expect(preview.getByText(`${records.activityId} · Waiting at level 2 of 2`)).toBeVisible();
  await preview.getByRole('button', { name: 'Approve', exact: true }).click();
  approve = page.getByRole('alertdialog', { name: `Approve '${name}'?` });
  await expect(approve.getByText('Level 2 of 2')).toBeVisible();
  await expect(
    approve.getByText('This is the last level — the activity becomes Approved.'),
  ).toBeVisible();
  await approve.getByRole('button', { name: 'Approve', exact: true }).click();
  await expectToast(page, `Activity '${name}' approved`);

  // The planner sees it Approved, with both decisions in its approval chain (008 FR-024).
  await signInAs(page, MEI);
  await page.goto(records.activityPath);
  await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible();
  await expect(page.getByText('Approved', { exact: true }).first()).toBeVisible();
  await expect(
    page.getByText('Approved — changing planned cost or dates will restart approval.'),
  ).toBeVisible();
  // CHECK: 008's ApprovalSection keeps the element id "approval" (the #approval anchor).
  const approval = page.locator('#approval');
  await expect(approval.getByText(/^Approved by Carlos Mendes on /).first()).toBeVisible();
  await expect(approval.getByText(/^Approved by Alex Morgan on /).first()).toBeVisible();
});

test('B: a declined activity shows the approver and the reason to the planner', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, MEI);
  const records = await createPlanningRecords(page, 'Decline', {
    withWorkflow: true,
    leaveEmpty: [],
  });
  const name = records.activityName;
  // Mei submits it (prepared through the API; the screen journey of submitting is test B above).
  await callApi(page, 'POST', `/api/activities/${records.activityId}/approval/submit`, {
    version: records.activityVersion,
    comment: '',
  });

  // Carlos Mendes declines at level 1; a decline needs a comment (008 FR-016).
  const reason = 'Please move half of the cost to the second half of the year.';
  const preview = await openApprovalPreview(page, CARLOS, name);
  await preview.getByRole('button', { name: 'Decline', exact: true }).click();
  const decline = page.getByRole('alertdialog', { name: `Decline '${name}'?` });
  await expect(
    decline.getByText('The activity goes back to Mei Lin, who can fix it and re-submit.'),
  ).toBeVisible();
  await decline.getByRole('button', { name: 'Decline activity' }).click();
  await expect(
    decline.getByText('Add a comment explaining why the activity is declined.'),
  ).toBeVisible();
  // CHECK: the comment field is a textarea named "Why are you declining it?" (+ its "Required" marking).
  await decline.getByRole('textbox', { name: /^Why are you declining it\?/ }).fill(reason);
  await decline.getByRole('button', { name: 'Decline activity' }).click();
  await expectToast(page, `Activity '${name}' declined`);

  // Mei sees Declined, who declined and why, and can re-submit (008 US1, FR-016).
  await signInAs(page, MEI);
  await page.goto(records.activityPath);
  await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible();
  await expect(page.getByText('Declined', { exact: true }).first()).toBeVisible();
  const approval = page.locator('#approval');
  await expect(approval.getByText(/^Declined by Carlos Mendes on /).first()).toBeVisible();
  await expect(approval.getByText(reason).first()).toBeVisible();
  await expect(page.getByRole('button', { name: 'Re-submit for approval' }).first()).toBeVisible();

  // The bell tells her too, quoting the comment (008 FR-035).
  // CHECK: 008 names the bell's trigger "Notifications, <n> unread".
  await page.getByRole('button', { name: /^Notifications/ }).click();
  await expect(
    page.getByText(`Carlos Mendes declined '${name}' (${records.activityId}): "${reason}"`),
  ).toBeVisible();
});

test('C: Settings — an administrator edits a model, a Shell attribute and a currency', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, ALEX);

  // Models (002 US1): a duplicate of Lubricants 1.0, then an edit of its INQ→MQL rate. The demo models
  // stay as they are; other tests use them by name.
  await page.goto('/settings');
  await expect(page.getByRole('heading', { name: 'Settings', level: 1 })).toBeVisible();
  // CHECK: each Settings card is one link whose name starts with the section title.
  await page.getByRole('link', { name: /^Models/ }).click();
  await expect(page).toHaveURL(/\/settings\/models/);
  await expect(page.getByRole('heading', { name: 'Models', level: 1 })).toBeVisible();
  const modelName = uniqueName('Core model');
  await page.getByRole('button', { name: 'Actions for Lubricants 1.0' }).click();
  await page.getByRole('menuitem', { name: 'Duplicate' }).click();
  const newModel = page.getByRole('dialog', { name: 'New model' });
  const modelNameField = newModel.getByRole('textbox', { name: /^Name/ });
  await expect(modelNameField).toHaveValue('Lubricants 1.0 (copy)');
  await modelNameField.fill(modelName);
  await newModel.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Model '${modelName}' created`);

  await page.getByRole('button', { name: `Actions for ${modelName}` }).click();
  await page.getByRole('menuitem', { name: 'Edit' }).click();
  const editModel = page.getByRole('dialog', { name: 'Edit model' });
  // CHECK: 002's DecimalField is a textbox named by its label; the "%" suffix is outside the name.
  await editModel.getByRole('textbox', { name: /^INQ→MQL rate/ }).fill('3');
  // CHECK: the worked example starts from 250,000 inquiries (as 002's own e2e journey reads it).
  await expect(
    editModel.getByText('250,000 INQ → 7,500 MQL → 375 deals → 3,750,000.00 USD revenue'),
  ).toBeVisible();
  // An unused model has no impact to confirm, so it saves straight away.
  await editModel.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Model '${modelName}' saved`);
  await expect(page.getByRole('row').filter({ hasText: modelName })).toContainText(
    '3.00 % → 5.00 %',
  );

  // Shell attributes (002 US2): the help text of the optional line-item attribute Contracted spend.
  await page.goto('/settings');
  await page.getByRole('link', { name: /^Shell attributes/ }).click();
  await expect(page.getByRole('heading', { name: 'Shell attributes', level: 1 })).toBeVisible();
  // CHECK: the level filter is 002's SegmentedFilter: buttons named "<level> <count>".
  await page.getByRole('button', { name: /^Line item/ }).click();
  await page.getByRole('link', { name: 'Contracted spend', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Contracted spend', level: 1 })).toBeVisible();
  await action(page, 'Edit').click();
  await expect(
    page.getByRole('heading', { name: "Edit Shell attribute 'Contracted spend'", level: 1 }),
  ).toBeVisible();
  const helpText = `Whether the spend is under contract (${uniqueName('core flow')}).`;
  await page.getByRole('textbox', { name: /^Help text/ }).fill(helpText);
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, "Shell attribute 'Contracted spend' saved");
  // CHECK: after an edit the editor returns to the attribute page, whose Settings card shows the help text.
  await expect(page.getByRole('heading', { name: 'Contracted spend', level: 1 })).toBeVisible();
  await expect(page.getByText(helpText).first()).toBeVisible();

  // Currencies (002 US4): INR's rate, which no demo line item uses (so no impact to confirm), with the
  // live line both ways; put back to 83.50 at the end for the tests that follow.
  await page.goto('/settings');
  await page.getByRole('link', { name: /^Currencies/ }).click();
  await expect(page.getByRole('heading', { name: 'Currencies', level: 1 })).toBeVisible();
  const inrRow = page.getByRole('row').filter({ hasText: 'INR' });
  await expect(inrRow).toContainText('1 USD = 83.50 INR');
  // CHECK: the row's ⋯ menu is labelled with the code or the name of the currency.
  await inrRow.getByRole('button', { name: /^Actions for / }).click();
  await page.getByRole('menuitem', { name: 'Edit' }).click();
  const editCurrency = page.getByRole('dialog', { name: 'Edit currency' });
  await expect(
    editCurrency.getByText("The code can't change once the currency is saved."),
  ).toBeVisible();
  // CHECK: the rate field is a textbox named "Rate (units per 1 USD)".
  await editCurrency.getByRole('textbox', { name: /^Rate/ }).fill('84');
  await expect(editCurrency.getByText('1 USD = 84.00 INR')).toBeVisible();
  await expect(editCurrency.getByText('1 INR = 0.011905 USD')).toBeVisible();
  await editCurrency.getByRole('button', { name: 'Save', exact: true }).click();
  // CHECK: the toast names the currency by its name ("Indian rupee") or its code.
  await expectToast(page, /^Currency '(Indian rupee|INR)' saved/);
  await expect(inrRow).toContainText('1 USD = 84.00 INR');

  const inr = await callApi<{ version: number }>(page, 'GET', '/api/currencies/INR');
  await callApi(page, 'PATCH', '/api/currencies/INR', { rate: '83.5', version: inr.version });
});

test('D: the plan form offers only members of the chosen teams as owner and refuses an owner outside them', async ({
  page,
}) => {
  await signInAs(page, PRIYA);
  await page.goto('/plans/new');
  await expect(page.getByRole('heading', { name: 'New plan', level: 1 })).toBeVisible();
  // CHECK: 003's TeamPicker and PersonPicker inputs are comboboxes named by their labels.
  const teams = page.getByRole('combobox', { name: /^Teams/ });
  const owner = page.getByRole('combobox', { name: /^Owner/ });

  // Defaults: Priya's team and Priya herself (004 FR-006).
  await expect(page.getByText('Default: your teams')).toBeVisible();
  await expect(page.getByText('Default: you', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Remove Lubricants Europe' })).toBeVisible();

  // The owner list holds only active members of Lubricants Europe, Priya first (003 FR-017, FR-019).
  await owner.click();
  await owner.press('ArrowDown');
  // CHECK: an option's name starts with the person's name ("Priya Raman (you)", then the job title).
  await expect(page.getByRole('option').first()).toHaveAccessibleName(/^Priya Raman \(you\)/);
  await expect(page.getByRole('option', { name: /^Daniel Okoro/ })).toBeVisible();
  await expect(page.getByRole('option', { name: /^Sara Novak/ })).toBeVisible();
  await expect(page.getByRole('option', { name: /^Mei Lin/ })).toHaveCount(0);
  await expect(page.getByText('3 people').first()).toBeVisible();
  await owner.fill('mei');
  await expect(
    page.getByText(
      'No members of the selected teams match. Add people to these teams in Settings › Teams.',
    ),
  ).toBeVisible();
  await owner.press('Escape');

  // Swap the team for Mobility China: Priya is not one of its members (R1, 004 FR-021).
  await teams.fill('Mobility');
  await page.getByRole('option', { name: /^Mobility China/ }).click();
  await teams.press('Escape');
  await page.getByRole('button', { name: 'Remove Lubricants Europe' }).click();
  const notInTeams =
    "Priya Raman isn't in any of the chosen teams — choose another owner or clear the field";
  await expect(page.getByText(notInTeams).first()).toBeVisible();
  // CHECK: the picker also flags the chosen owner with its default notMemberText.
  await expect(page.getByText("Not a member of the plan's teams").first()).toBeVisible();

  // Saving anyway is refused by the API with the same rule, listed in the error summary.
  const planName = uniqueName('Owner rules plan');
  await page.getByRole('textbox', { name: /^Plan name/ }).fill(planName);
  await chooseOption(page.getByLabel(/^Model/), 'Mobility 1.0');
  await page.getByRole('checkbox', { name: 'Mobility', exact: true }).check();
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(
    page.getByRole('heading', { name: /problems? needs? your attention/ }),
  ).toBeVisible();
  await expect(
    page.getByRole('link', { name: new RegExp(`^${escapeRegExp(notInTeams)}`) }),
  ).toBeVisible();
  await expect(page).toHaveURL(/\/plans\/new$/);

  // Without any team the owner field is disabled and says why (004 FR-007).
  await page.getByRole('button', { name: 'Remove Mobility China' }).click();
  await expect(owner).toHaveAttribute('aria-disabled', 'true');
  await expect(
    page.getByText("Choose teams first — the owner must belong to one of the plan's teams").first(),
  ).toBeVisible();

  // Back to Mobility China and choose one of its members; the picker searches job titles too.
  await teams.fill('Mobility');
  await page.getByRole('option', { name: /^Mobility China/ }).click();
  await teams.press('Escape');
  await owner.fill('finance');
  await expect(page.getByRole('option', { name: /^Sara Novak/ })).toBeVisible();
  await owner.fill('Mei');
  await page.getByRole('option', { name: /^Mei Lin/ }).click();
  await page.getByRole('button', { name: 'Save', exact: true }).click();

  await expectToast(page, `Plan '${planName}' created — add campaigns next.`);
  await expect(page).toHaveURL(/\/plans\/P-\d+$/);
  await expect(page.getByRole('heading', { name: planName, level: 1 })).toBeVisible();
  // CHECK: the header's owner fact reads "Owner: <name>" as one text.
  await expect(page.getByText('Owner: Mei Lin').first()).toBeVisible();
  await expect(page.getByText('Mobility China').first()).toBeVisible();
});

test('E: deleting an activity moves it to the recycle bin with its cost, and Undo brings it back', async ({
  page,
}) => {
  await signInAs(page, MEI);
  const records = await createPlanningRecords(page, 'Delete', {
    withWorkflow: false,
    leaveEmpty: [],
  });
  const name = records.activityName;

  await page.goto(records.activityPath);
  await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible();
  // CHECK: the header's More menu is 001's ActionsMenu labelled with the activity name.
  await page.getByRole('button', { name: `Actions for ${name}` }).click();
  await page.getByRole('menuitem', { name: 'Delete' }).click();
  const dialog = page.getByRole('alertdialog', { name: `Delete activity '${name}'?` });
  // CHECK: the amount uses the display-currency format ("$20,000.00").
  await expect(
    dialog.getByText(/^The initiative, campaign and plan planned costs go down by .*20,000\.00\.$/),
  ).toBeVisible();
  await expect(dialog.getByText('It moves to the recycle bin; you can restore it.')).toBeVisible();
  await expect(dialog.getByRole('button', { name: 'Cancel' })).toBeFocused();
  await dialog.getByRole('button', { name: 'Delete activity' }).click();

  // Back on the plan page: the activity and its planned cost are gone from the tree (006 FR-031).
  await expect(page).toHaveURL(new RegExp(`/plans/${records.planId}$`));
  await expectToast(page, `Activity '${name}' moved to the recycle bin`);
  await page.getByRole('button', { name: 'Expand all' }).click();
  await expect(treeRow(page, records.initiativeName)).toBeVisible();
  await expect(treeRow(page, name)).toHaveCount(0);
  await expect(treeRow(page, records.initiativeName)).not.toContainText('20,000.00');

  // Undo within the toast's 10 seconds brings it back with the same ID (001 FR-026).
  await page.getByRole('button', { name: 'Undo' }).click();
  await expectToast(page, `Activity '${name}' restored`);
  await page.getByRole('button', { name: 'Expand all' }).click();
  await expect(planTree(page).getByRole('link', { name, exact: true })).toHaveAttribute(
    'href',
    new RegExp(`/activities/${records.activityId}$`),
  );
  await expect(treeRow(page, records.initiativeName)).toContainText('20,000.00');
});

test('Accessibility: the Plans list, a plan page, an activity page and Approvals pass axe', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, PRIYA);

  await page.goto('/plans');
  await expect(page.getByRole('heading', { name: 'Plans', level: 1 })).toBeVisible();
  await expect(
    page.getByRole('link', { name: 'FY2026 Lubricants Europe', exact: true }),
  ).toBeVisible();
  await expectNoAxeViolations(page);

  await page.goto('/plans/P-1001');
  await expect(
    page.getByRole('heading', { name: 'FY2026 Lubricants Europe', level: 1 }),
  ).toBeVisible();
  await expect(treeRow(page, 'Helix launch')).toBeVisible();
  await expectNoAxeViolations(page);

  // Launch film (A-1001) with its two seeded line items.
  await page.goto('/plans/P-1001/activities/A-1001');
  await expect(page.getByRole('heading', { name: 'Launch film', level: 1 })).toBeVisible();
  await expect(page.getByRole('row').filter({ hasText: 'Agency fee' }).first()).toBeVisible();
  await expectNoAxeViolations(page);

  // Daniel Okoro approves Lubricants Europe's activities; the rows depend on earlier test files.
  await signInAs(page, DANIEL);
  await page.goto('/approvals');
  await expect(page.getByRole('heading', { name: 'Approvals', level: 1 })).toBeVisible();
  await expect(
    page
      .getByRole('table')
      .or(page.getByText("Nothing is waiting for you. You're all caught up."))
      .first(),
  ).toBeVisible();
  await expectNoAxeViolations(page);
});
