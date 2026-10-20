/**
 * Field help entries for the tests of this folder and of the pages that show ⓘ tips, shaped as
 * GET /api/field-help answers them (backend/api/openapi.yaml, feature 016 data-model §6).
 */
import type { components } from '@/api/schema';
import type { FieldHelpEntry, FieldHelpPage } from './fieldHelpApi';

type ActorRef = components['schemas']['ActorRef'];

const PRIYA_REF: ActorRef = { id: 'U-1002', name: 'Priya Raman' };

function entry(
  overrides: Partial<FieldHelpEntry> & Pick<FieldHelpEntry, 'key' | 'label' | 'defaultText'>,
): FieldHelpEntry {
  return {
    level: 'plan',
    text: overrides.defaultText,
    isDefault: true,
    version: 0,
    updatedAt: null,
    updatedBy: null,
    ...overrides,
  };
}

/** Plan budget, with its default text. */
export const PLAN_BUDGET = entry({
  key: 'plan.budget',
  label: 'Plan budget',
  defaultText:
    'The money this plan may spend in its year, in the plan currency. Campaign budgets are allocated from it.',
});

/** Planned cost, edited by Priya. */
export const PLAN_PLANNED_COST = entry({
  key: 'plan.plannedCost',
  label: 'Planned cost',
  text: 'Everything the plan expects to spend: the planned costs of all its activities added up.',
  defaultText: 'The sum of the planned costs of every activity in the plan.',
  isDefault: false,
  version: 2,
  updatedAt: '2026-10-19T14:05:00Z',
  updatedBy: PRIYA_REF,
});

/** BCPIR plan, emptied by Priya: no ⓘ is shown. */
export const PLAN_BCPIR = entry({
  key: 'plan.bcpir',
  label: 'BCPIR plan',
  text: '',
  defaultText: 'Marks the plan as a BCPIR plan.',
  isDefault: false,
  version: 1,
  updatedAt: '2026-10-18T09:00:00Z',
  updatedBy: PRIYA_REF,
});

/** Campaign budget, default. */
export const CAMPAIGN_BUDGET = entry({
  key: 'campaign.budget',
  level: 'campaign',
  label: 'Campaign budget',
  defaultText:
    'The part of the plan budget given to this campaign, in the plan currency. Initiative budgets are allocated from it.',
});

/** Left to plan, default. */
export const CAMPAIGN_LEFT_TO_PLAN = entry({
  key: 'campaign.leftToPlan',
  level: 'campaign',
  label: 'Left to plan',
  defaultText: 'Campaign budget minus planned cost. Below zero means the campaign is over budget.',
});

export const HELP_ENTRIES: FieldHelpEntry[] = [
  PLAN_BUDGET,
  PLAN_PLANNED_COST,
  PLAN_BCPIR,
  CAMPAIGN_BUDGET,
  CAMPAIGN_LEFT_TO_PLAN,
];

/** The answer of GET /api/field-help, with 3 Shell attributes lacking help. */
export const FIELD_HELP_PAGE: FieldHelpPage = {
  items: HELP_ENTRIES,
  total: HELP_ENTRIES.length,
  shellAttributesWithoutHelp: 3,
};

/** An answer for pages that only need the request answered. */
export const NO_FIELD_HELP: FieldHelpPage = { items: [], total: 0, shellAttributesWithoutHelp: 0 };
