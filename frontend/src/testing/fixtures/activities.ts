/**
 * Activities for the tests of feature 006 (contract shapes, as the API answers them): the plan
 * "FY2026 Lubricants Europe", the initiative "Digital & social" (Priya Raman, 1 Jan – 30 Jun 2026)
 * and the activity "Launch film" (A-1042).
 */
import type { components } from '@/api/generated/schema';
import { attribute, LUBRICANTS, MOBILITY, option } from './attribute-forms';
import type {
  Activity,
  ActivityDestination,
  ActivityFormData,
  ActivitySummary,
} from '@/features/activities/activitiesApi';

type Person = components['schemas']['Person'];
type AttributeForm = components['schemas']['AttributeForm'];

export const PRIYA = {
  id: 'U-1002',
  name: 'Priya Raman',
  initials: 'PR',
  jobTitle: 'Marketer',
  active: true,
};
export const ALEX = {
  id: 'U-1001',
  name: 'Alex Morgan',
  initials: 'AM',
  jobTitle: 'Head of marketing',
  active: true,
};

const META = {
  version: 1,
  createdAt: '2026-03-01T09:00:00Z',
  createdBy: { id: 'U-1001', name: 'Alex Morgan' },
  updatedAt: '2026-03-01T09:00:00Z',
  updatedBy: { id: 'U-1001', name: 'Alex Morgan' },
};
const EUROPE = { id: 'T-1001', name: 'Lubricants Europe', colour: 'blue', active: true } as const;

/** GET /api/v1/people, for the Owner field. */
export const PEOPLE: { items: Person[]; total: number } = {
  total: 2,
  items: [ALEX, PRIYA].map((person) => ({
    ...person,
    email: `${person.name.toLowerCase().replace(' ', '.')}@shell.com`,
    teams: [EUROPE],
    ...META,
  })),
};

/** The model "Lubricants 1.0": 2 %, 5 %, 10,000 USD per deal. */
export const MODEL = {
  id: 'MOD-1001',
  name: 'Lubricants 1.0',
  inqToMqlRate: '2.00',
  mqlToDealRate: '5.00',
  averageDealSize: { amount: '10000.00', currency: 'USD' },
  active: true,
};

/** The new-activity form of "Digital & social"; pass what a test needs to differ. */
export function newForm(overrides: Partial<ActivityFormData> = {}): ActivityFormData {
  return {
    context: {
      plan: { id: 'P-1001', name: 'FY2026 Lubricants Europe', year: 2026, status: 'draft' },
      campaign: { type: 'campaign', id: 'C-1001', name: 'Helix launch' },
      initiative: {
        id: 'I-1001',
        name: 'Digital & social',
        owner: PRIYA,
        startDate: '2026-01-01',
        endDate: '2026-06-30',
      },
      model: MODEL,
      businessUnits: [{ id: 'SO-BU-LUB', label: 'Lubricants' }],
      latestEndDate: '2036-12-31',
    },
    activityTypes: [
      {
        id: 'AT-1002',
        name: 'Digital campaign',
        defaultPlannedCost: { amount: '10000.00', currency: 'USD' },
        active: true,
        allowed: true,
      },
      {
        id: 'AT-1001',
        name: 'Event',
        defaultPlannedCost: { amount: '25000.00', currency: 'USD' },
        active: true,
        allowed: true,
      },
      { id: 'AT-1003', name: 'Sponsorship', defaultPlannedCost: null, active: true, allowed: true },
    ],
    approvalWorkflows: [
      { id: 'WF-1001', name: 'Lubricants EU approval', active: true, levels: [] },
    ],
    attributes: [],
    existingNames: ['Launch film'],
    defaults: {
      owner: PRIYA,
      startDate: '2026-01-01',
      endDate: '2026-06-30',
      approvalWorkflowId: 'WF-1001',
    },
    ...overrides,
  };
}

/** The edit form of "Launch film": no defaults, its own name left out of the names in use. */
export function editForm(overrides: Partial<ActivityFormData> = {}): ActivityFormData {
  return newForm({ existingNames: ['Print ads'], defaults: null, ...overrides });
}

const NO_PROBLEMS = { ready: true, problemCount: 0, activity: [], lineItems: [], advice: [] };

/** "Launch film" (A-1042), Created, in "Digital & social"; pass what a test needs to differ. */
export function launchFilm(overrides: Partial<Activity> = {}): Activity {
  const activity: Omit<Activity, 'display'> = {
    id: 'A-1042',
    name: 'Launch film',
    externalName: '',
    plan: { type: 'plan', id: 'P-1001', name: 'FY2026 Lubricants Europe' },
    campaign: { type: 'campaign', id: 'C-1001', name: 'Helix launch' },
    initiative: { type: 'initiative', id: 'I-1001', name: 'Digital & social' },
    activityType: { id: 'AT-1001', name: 'Event', active: true, allowed: true },
    owner: PRIYA,
    startDate: '2026-03-01',
    endDate: '2026-05-31',
    ongoing: false,
    plannedCost: { amount: '120000.00', currency: 'USD' },
    approvalWorkflow: { id: 'WF-1001', name: 'Lubricants EU approval', active: true },
    description: '',
    attributes: {},
    goals: {
      inq: null,
      setLaterStagesMyself: false,
      mql: null,
      deals: null,
      revenue: null,
      calculated: true,
    },
    status: 'created',
    displayStatus: 'created',
    unassignedPlannedCost: { amount: '20000.00', currency: 'USD' },
    lineItems: {
      count: 0,
      plannedCost: { amount: '0.00', currency: 'USD' },
      autoSpread: 0,
      customSpread: 0,
    },
    completeness: { missingFields: 0, lineItemsNeedingAttention: 0 },
    readiness: NO_PROBLEMS,
    version: 3,
    createdAt: '2026-03-01T09:00:00Z',
    createdBy: { id: 'U-1001', name: 'Alex Morgan' },
    updatedAt: '2026-03-12T10:42:00Z',
    updatedBy: { id: 'U-1002', name: 'Priya Raman' },
    ...overrides,
  };
  // A reader in USD gets the same amounts again (D-220); tests of another currency pass `display`.
  const display = overrides.display ?? {
    plannedCost: activity.plannedCost,
    unassignedPlannedCost: activity.unassignedPlannedCost,
    revenue: activity.goals.revenue,
  };
  return { ...activity, display };
}

function problem(
  code: string,
  field: string,
  message: string,
  lineItem: { id: string; name: string } | null = null,
) {
  return {
    code,
    field,
    message,
    lineItemId: lineItem?.id ?? null,
    lineItemName: lineItem?.name ?? null,
  };
}

/** A check with five problems: three of the activity, two of its line items (FR-023). */
export const FIVE_PROBLEMS: Activity['readiness'] = {
  ready: false,
  problemCount: 5,
  activity: [
    problem('missingField', 'ownerId', 'Owner — needed to submit'),
    problem('missingField', 'attributes.SA-1015', 'Cost centre — needed to submit'),
    problem(
      'splitIncomplete',
      'attributes.SA-1013',
      'Country / OU shares add up to 90.00 % — they must add up to 100.00 %',
    ),
  ],
  lineItems: [
    // The API's message names the line item already (006 plan, "Ready to submit panel").
    problem(
      'missingField',
      'attributes.SA-1033',
      "Line item 'Film shoot' (L-1001): GL code — needed to submit",
      { id: 'L-1001', name: 'Film shoot' },
    ),
    problem(
      'negativeUnassigned',
      'lineItems',
      "Line items exceed the activity's planned cost by 500.00 USD",
    ),
  ],
  advice: [],
};

/** "Launch film" with the five problems: 3 fields needed, 1 line item needing attention. */
export function incompleteLaunchFilm(overrides: Partial<Activity> = {}): Activity {
  return launchFilm({
    owner: null,
    readiness: FIVE_PROBLEMS,
    completeness: { missingFields: 3, lineItemsNeedingAttention: 1 },
    ...overrides,
  });
}

/** "Launch film" as a plan-tree row. */
export function launchFilmRow(overrides: Partial<ActivitySummary> = {}): ActivitySummary {
  const activity = launchFilm();
  const row: Omit<ActivitySummary, 'display'> = {
    id: activity.id,
    name: activity.name,
    initiativeId: 'I-1001',
    activityType: activity.activityType,
    owner: activity.owner,
    startDate: activity.startDate,
    endDate: activity.endDate,
    ongoing: activity.ongoing,
    plannedCost: activity.plannedCost,
    unassignedPlannedCost: activity.unassignedPlannedCost,
    status: activity.status,
    displayStatus: activity.displayStatus,
    completeness: activity.completeness,
    version: activity.version,
    ...overrides,
  };
  // A reader in USD gets the same amounts again (D-220); tests of another currency pass `display`.
  return {
    ...row,
    display: overrides.display ?? {
      plannedCost: row.plannedCost,
      unassignedPlannedCost: row.unassignedPlannedCost,
      revenue: null,
    },
  };
}

/** The API's answer to a refused save with field problems. */
export function validationFailed(fields: { field: string; message: string }[]) {
  return {
    status: 400,
    body: {
      error: { code: 'validation_failed', message: 'Check the highlighted fields.', fields },
    },
  };
}

/**
 * The activity level's Shell attributes (GET /api/v1/shell-attribute-forms/activity): Cluster →
 * Region, the Country / OU split, a Lubricants group (Brands) and a Mobility group (Fleet card).
 */
export function activityAttributeForm(): AttributeForm {
  return {
    level: 'activity',
    businessUnitAttributeId: 'SA-BU',
    businessUnitOptions: [option(LUBRICANTS, 'Lubricants'), option(MOBILITY, 'Mobility')],
    sections: [
      {
        name: 'Shell attributes',
        conditional: false,
        attributes: [
          attribute({
            id: 'SA-CLUSTER',
            label: 'Cluster',
            marking: 'neededToSubmit',
            options: [option('SO-RETAIL', 'Retail'), option('SO-B2B', 'B2B')],
          }),
          attribute({
            id: 'SA-REGION',
            label: 'Region',
            parentAttributeId: 'SA-CLUSTER',
            options: [
              option('SO-CHINA', 'RETAIL – CHINA', ['SO-RETAIL']),
              option('SO-EU', 'B2B – EUROPE', ['SO-B2B']),
              option('SO-ASIA', 'B2B – ASIA', ['SO-B2B']),
            ],
          }),
          attribute({
            id: 'SA-1013',
            label: 'Country / OU',
            type: 'multipleChoiceSplit',
            marking: 'neededToSubmit',
            options: [
              option('SO-DE', 'Germany'),
              option('SO-FR', 'France'),
              option('SO-NL', 'Netherlands'),
            ],
          }),
        ],
      },
      {
        name: 'Lubricants',
        conditional: true,
        attributes: [
          attribute({
            id: 'SA-BRANDS',
            label: 'Brands',
            type: 'multipleChoice',
            conditionOptionIds: [LUBRICANTS],
            options: [option('SO-HELIX', 'Shell Helix'), option('SO-RIMULA', 'Shell Rimula')],
          }),
        ],
      },
      {
        name: 'Mobility',
        conditional: true,
        attributes: [
          attribute({
            id: 'SA-FLEET',
            label: 'Fleet card',
            type: 'text',
            conditionOptionIds: [MOBILITY],
          }),
        ],
      },
    ],
  };
}

/**
 * An initiative "Launch film" can be moved or copied to (GET /api/v1/activities/{id}/destinations):
 * "Print & outdoor" (1 Jan – 31 Dec 2026) in "Helix launch", available for both.
 */
export function destination(overrides: Partial<ActivityDestination> = {}): ActivityDestination {
  return {
    campaign: { type: 'campaign', id: 'C-1001', name: 'Helix launch' },
    initiative: { type: 'initiative', id: 'I-1002', name: 'Print & outdoor' },
    startDate: '2026-01-01',
    endDate: '2026-12-31',
    current: false,
    moveProblem: null,
    copyProblem: null,
    ...overrides,
  };
}

/** "Digital & social", the activity's own initiative: copies may go there, moves may not. */
export const OWN_INITIATIVE = destination({
  initiative: { type: 'initiative', id: 'I-1001', name: 'Digital & social' },
  endDate: '2026-06-30',
  current: true,
  moveProblem: "It's already in this initiative.",
});

/** "Retail events" in "Rimula fleet" (1 Jun – 31 Dec 2026): Launch film's dates don't fit. */
export const RETAIL_EVENTS = destination({
  campaign: { type: 'campaign', id: 'C-1002', name: 'Rimula fleet' },
  initiative: { type: 'initiative', id: 'I-1003', name: 'Retail events' },
  startDate: '2026-06-01',
  moveProblem:
    "'Launch film' runs 01 Mar 2026 – 31 May 2026, but 'Retail events' runs 01 Jun 2026 – 31 Dec 2026. Change the activity's dates first, or pick another initiative.",
  copyProblem:
    "'Launch film' runs 01 Mar 2026 – 31 May 2026, but 'Retail events' runs 01 Jun 2026 – 31 Dec 2026. Change the activity's dates first, or pick another initiative.",
});

/** The three initiatives of the plan, grouped by campaign as the API answers them. */
export function destinations(
  items: ActivityDestination[] = [OWN_INITIATIVE, destination(), RETAIL_EVENTS],
) {
  return { items, total: items.length };
}
