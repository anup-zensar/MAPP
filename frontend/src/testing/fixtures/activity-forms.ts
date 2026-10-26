/** The activity form's answers (feature 006): its data, the Shell attributes, a refused save. */
import type { components } from '@/api/generated/schema';
import type { ActivityFormData } from '@/features/activities';
import { MODEL, PRIYA } from './activities';
import { attribute, LUBRICANTS, MOBILITY, option } from './attribute-forms';

type AttributeForm = components['schemas']['AttributeForm'];
type ErrorBody = components['schemas']['Error'];

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
export function validationFailed(fields: { field: string; message: string }[]): {
  status: number;
  body: ErrorBody;
} {
  return {
    status: 400,
    body: {
      error: { code: 'validation_failed', message: 'Check the highlighted fields.', fields },
    },
  };
}

/** Cluster → Region, the Country / OU split, Lubricants' Brands and Mobility's Fleet card. */
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
