/**
 * Activities of feature 006 as the API answers them: "Launch film" (A-1042) in the initiative
 * "Digital & social" (Priya Raman, 1 Jan – 30 Jun 2026) of the plan "FY2026 Lubricants Europe".
 */
import type { components } from '@/api/generated/schema';
import type { Activity, ActivitySummary } from '@/features/activities';

type Person = components['schemas']['Person'];

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

export const DELETED = {
  recycleBinEntry: {
    id: 'BIN-1001',
    recordType: 'activity',
    recordTypeLabel: 'activity',
    recordId: 'A-1042',
    recordName: 'Launch film',
    path: [],
    parent: { type: 'initiative', id: 'I-1001', name: 'Digital & social' },
    deletedAt: '2026-03-12T09:00:00Z',
    deletedBy: { id: 'U-1002', name: 'Priya Raman' },
    contents: [],
  },
};

export const RESTORED = {
  recordType: 'activity',
  recordTypeLabel: 'activity',
  recordId: 'A-1042',
  recordName: 'Launch film',
  details: '',
};
