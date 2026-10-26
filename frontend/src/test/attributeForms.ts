import type { components } from '@/api/schema';

type AttributeForm = components['schemas']['AttributeForm'];
type FormAttribute = components['schemas']['FormAttribute'];
type FormOption = components['schemas']['FormOption'];

/**
 * Shell attribute forms for tests (contract AttributeForm, as GET /api/v1/shell-attribute-forms/{level}
 * answers), shaped like the demo Settings: Business unit (Lubricants, Mobility), Cluster → Region,
 * and a Lubricants-only section.
 */

export const LUBRICANTS = 'SO-BU-LUB';
export const MOBILITY = 'SO-BU-MOB';

const BUSINESS_UNIT_OPTIONS: FormOption[] = [
  option(LUBRICANTS, 'Lubricants'),
  option(MOBILITY, 'Mobility'),
];

/** A form option; `parents` are the parent attribute's option ids it is linked to. */
export function option(
  id: string,
  label: string,
  parents: string[] = [],
  active = true,
): FormOption {
  return { id, label, active, parentOptionIds: parents };
}

/** A form attribute with sensible defaults: an active optional single choice without options. */
export function attribute(overrides: Partial<FormAttribute> & { id: string; label: string }) {
  return {
    helpText: '',
    type: 'singleChoice',
    marking: 'optional',
    active: true,
    parentAttributeId: null,
    conditionOptionIds: [],
    options: [],
    ...overrides,
  } satisfies FormAttribute;
}

/**
 * The campaign form: "Shell attributes" — Cluster (needed to publish), Region (depends on Cluster),
 * SMP (yes/no), Campaign code (text), Notes (long text), Target accounts (number) — and
 * "Lubricants" — Brands, shown only for the Lubricants business unit.
 */
export function campaignForm(): AttributeForm {
  return {
    level: 'campaign',
    businessUnitAttributeId: 'SA-BU',
    businessUnitOptions: BUSINESS_UNIT_OPTIONS,
    sections: [
      {
        name: 'Shell attributes',
        conditional: false,
        attributes: [
          attribute({
            id: 'SA-CLUSTER',
            label: 'Cluster',
            type: 'multipleChoice',
            marking: 'neededToPublish',
            helpText: 'The market clusters the campaign runs in.',
            options: [
              option('SO-RETAIL', 'Retail'),
              option('SO-B2B', 'B2B'),
              option('SO-FLEET', 'Fleet'),
            ],
          }),
          attribute({
            id: 'SA-REGION',
            label: 'Region',
            type: 'multipleChoice',
            marking: 'neededToPublish',
            parentAttributeId: 'SA-CLUSTER',
            options: [
              option('SO-CHINA', 'RETAIL – CHINA', ['SO-RETAIL']),
              option('SO-EU', 'B2B – EUROPE', ['SO-B2B']),
              option('SO-ASIA', 'B2B – ASIA', ['SO-B2B']),
              option('SO-OLD', 'B2B – OLD', ['SO-B2B'], false),
            ],
          }),
          attribute({ id: 'SA-SMP', label: 'SMP', type: 'yesNo' }),
          attribute({ id: 'SA-CODE', label: 'Campaign code', type: 'text' }),
          attribute({ id: 'SA-NOTES', label: 'Notes', type: 'longText' }),
          attribute({ id: 'SA-ACCOUNTS', label: 'Target accounts', type: 'number' }),
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
            marking: 'neededToPublish',
            conditionOptionIds: [LUBRICANTS],
            options: [option('SO-HELIX', 'Shell Helix'), option('SO-RIMULA', 'Shell Rimula')],
          }),
        ],
      },
    ],
  };
}

/** The plan form: Business unit first, then a Mobility-only Fleet size. */
export function planForm(): AttributeForm {
  return {
    level: 'plan',
    businessUnitAttributeId: 'SA-BU',
    businessUnitOptions: BUSINESS_UNIT_OPTIONS,
    sections: [
      {
        name: 'Shell attributes',
        conditional: false,
        attributes: [
          attribute({
            id: 'SA-BU',
            label: 'Business unit',
            type: 'multipleChoice',
            marking: 'neededToPublish',
            options: BUSINESS_UNIT_OPTIONS,
          }),
          attribute({
            id: 'SA-FLEET',
            label: 'Fleet size',
            type: 'number',
            conditionOptionIds: [MOBILITY],
          }),
        ],
      },
    ],
  };
}

/** An activity form with a % split attribute (only activities and line items have them). */
export function activityForm(): AttributeForm {
  return {
    level: 'activity',
    businessUnitAttributeId: 'SA-BU',
    businessUnitOptions: BUSINESS_UNIT_OPTIONS,
    sections: [
      {
        name: 'Shell attributes',
        conditional: false,
        attributes: [
          attribute({
            id: 'SA-SPLIT',
            label: 'Brand split',
            type: 'multipleChoiceSplit',
            marking: 'neededToSubmit',
            options: [option('SO-HELIX', 'Shell Helix'), option('SO-RIMULA', 'Shell Rimula')],
          }),
          attribute({
            id: 'SA-RETIRED',
            label: 'Retired field',
            type: 'text',
            marking: 'neededToSubmit',
            active: false,
          }),
        ],
      },
    ],
  };
}
