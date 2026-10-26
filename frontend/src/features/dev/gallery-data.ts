/**
 * Example data for the Plan-module cards of the component gallery: people, teams and a Shell
 * attribute form shaped like the demo Settings.
 */
import type { PickerPerson } from '@/ui/PersonPicker/PersonPicker';
import type { AttributeForm } from '@/ui/ShellAttributeFields/rules';
import type { TeamRef } from '@/ui/TeamPill/TeamPill';
import type { TreeNode } from '@/ui/TreeTable/TreeTable';

const LUBRICANTS_EUROPE: TeamRef = {
  id: 'T-1001',
  name: 'Lubricants Europe',
  colour: 'blue',
  active: true,
};
const MOBILITY_CHINA: TeamRef = {
  id: 'T-1002',
  name: 'Mobility China',
  colour: 'teal',
  active: true,
};

export const TEAMS: TeamRef[] = [
  LUBRICANTS_EUROPE,
  MOBILITY_CHINA,
  { id: 'T-1003', name: 'Fleet Americas', colour: 'gold', active: false },
];

export const ALEX: PickerPerson = {
  id: 'U-1001',
  name: 'Alex Morgan',
  email: 'alex.morgan@shell.com',
  jobTitle: 'Head of marketing',
  initials: 'AM',
  active: true,
  teams: [LUBRICANTS_EUROPE],
};

export const PRIYA: PickerPerson = {
  id: 'U-1002',
  name: 'Priya Raman',
  email: 'priya.raman@shell.com',
  jobTitle: 'Marketer',
  initials: 'PR',
  active: true,
  teams: [LUBRICANTS_EUROPE],
};

export const PEOPLE: PickerPerson[] = [
  ALEX,
  PRIYA,
  {
    id: 'U-1004',
    name: 'Sara Novak',
    email: 'sara.novak@shell.com',
    jobTitle: 'Finance partner',
    initials: 'SN',
    active: true,
    teams: [MOBILITY_CHINA],
  },
];

/** 1,217 cost centres, to show the searchable choice with a capped list (SC-007). */
export const COST_CENTRES = Array.from({ length: 1217 }, (_, index) => ({
  id: `SO-${index + 1}`,
  label: `CC-${String(index).padStart(4, '0')}`,
  active: true,
}));

export const CAMPAIGN_FORM: AttributeForm = {
  level: 'campaign',
  businessUnitAttributeId: 'SA-BU',
  businessUnitOptions: [
    { id: 'SO-LUB', label: 'Lubricants', active: true, parentOptionIds: [] },
    { id: 'SO-MOB', label: 'Mobility', active: true, parentOptionIds: [] },
  ],
  sections: [
    {
      name: 'Shell attributes',
      conditional: false,
      attributes: [
        {
          id: 'SA-CLUSTER',
          label: 'Cluster',
          helpText: 'The market clusters the campaign runs in.',
          type: 'multipleChoice',
          marking: 'neededToPublish',
          active: true,
          parentAttributeId: null,
          conditionOptionIds: [],
          options: [
            { id: 'SO-RETAIL', label: 'Retail', active: true, parentOptionIds: [] },
            { id: 'SO-B2B', label: 'B2B', active: true, parentOptionIds: [] },
          ],
        },
        {
          id: 'SA-REGION',
          label: 'Region',
          helpText: '',
          type: 'multipleChoice',
          marking: 'neededToPublish',
          active: true,
          parentAttributeId: 'SA-CLUSTER',
          conditionOptionIds: [],
          options: [
            {
              id: 'SO-CHINA',
              label: 'RETAIL – CHINA',
              active: true,
              parentOptionIds: ['SO-RETAIL'],
            },
            { id: 'SO-EU', label: 'B2B – EUROPE', active: true, parentOptionIds: ['SO-B2B'] },
            { id: 'SO-ASIA', label: 'B2B – ASIA', active: true, parentOptionIds: ['SO-B2B'] },
          ],
        },
        {
          id: 'SA-SMP',
          label: 'SMP',
          helpText: 'Part of a strategic marketing programme.',
          type: 'yesNo',
          marking: 'optional',
          active: true,
          parentAttributeId: null,
          conditionOptionIds: [],
          options: [],
        },
      ],
    },
    {
      name: 'Lubricants',
      conditional: true,
      attributes: [
        {
          id: 'SA-BRANDS',
          label: 'Brands',
          helpText: '',
          type: 'multipleChoice',
          marking: 'neededToPublish',
          active: true,
          parentAttributeId: null,
          conditionOptionIds: ['SO-LUB'],
          options: [
            { id: 'SO-HELIX', label: 'Shell Helix', active: true, parentOptionIds: [] },
            { id: 'SO-RIMULA', label: 'Shell Rimula', active: true, parentOptionIds: [] },
          ],
        },
      ],
    },
  ],
};

export const PLAN_TREE: TreeNode[] = [
  {
    id: 'P-1001',
    level: 'plan',
    name: 'FY2026 Lubricants Europe',
    cells: { owner: 'Priya Raman', budget: '$1,200,000.00' },
    children: [
      {
        id: 'C-1001',
        level: 'campaign',
        name: 'Helix launch',
        href: '/dev/components',
        cells: { name: 'C-1001 · 1 initiative', owner: 'Alex Morgan', budget: '$500,000.00' },
        children: [
          {
            id: 'I-1001',
            level: 'initiative',
            name: 'Digital & social',
            cells: { owner: 'Alex Morgan', budget: '$200,000.00' },
            children: [
              {
                id: 'A-1001',
                level: 'activity',
                name: 'Launch film',
                cells: { owner: 'Priya Raman', budget: '—' },
                children: [],
              },
            ],
          },
        ],
      },
    ],
  },
];
