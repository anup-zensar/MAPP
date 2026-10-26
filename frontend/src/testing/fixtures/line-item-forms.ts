/** The line-item forms' answers (feature 007): form options, attributes, people, rates, refusals. */
import type { HttpHandler } from 'msw';
import type { components } from '@/api/generated/schema';
import { attribute, option } from './attribute-forms';
import {
  activity,
  DIGITAL,
  DIGITAL_2027,
  EUR,
  EVENTS,
  GBP,
  launchFilmSection,
  MOBILITY_RETAIL,
  PRIYA_REF,
  USD,
  usd,
} from './line-items';
import { reply } from '../replies';

type LineItemFormOptions = components['schemas']['LineItemFormOptions'];
type AttributeForm = components['schemas']['AttributeForm'];
type Person = components['schemas']['Person'];
type ErrorBody = components['schemas']['Error'];

const ACTOR = { id: 'U-1002', name: 'Priya Raman' };

/** The form options of "Launch film": 20,000.00 USD unassigned (US2 scenario 1). */
export function formOptions(overrides: Partial<LineItemFormOptions> = {}): LineItemFormOptions {
  return {
    activity: activity(),
    balance: launchFilmSection().balance,
    defaults: {
      name: 'Launch film – line item',
      owner: PRIYA_REF,
      currency: 'USD',
      plannedCost: usd('20000.00'),
      plannedCostSource: 'unassignedBalance',
      spreadMethod: 'evenly',
      financeBudgetYear: 2026,
      financeBudgetItemId: null,
      financeBudgetItemPreselected: false,
      requestSwbs: true,
    },
    currencies: [USD, EUR, GBP],
    financeBudgetYears: [2026, 2027],
    financeBudgetItems: [DIGITAL, EVENTS, MOBILITY_RETAIL, DIGITAL_2027],
    ...overrides,
  };
}

/** The line-item Shell attributes: Spend type → GL code (R11), both needed to submit. */
export function lineItemForm(): AttributeForm {
  return {
    level: 'lineItem',
    businessUnitAttributeId: 'SA-BU',
    businessUnitOptions: [option('SO-BU-LUB', 'Lubricants'), option('SO-BU-MOB', 'Mobility')],
    sections: [
      {
        name: 'Finance coding',
        conditional: false,
        attributes: [
          attribute({
            id: 'SA-ST',
            label: 'Spend type',
            marking: 'neededToSubmit',
            options: [option('SO-ST-ATL', 'ATL – Media'), option('SO-ST-BTL', 'BTL – Event')],
          }),
          attribute({
            id: 'SA-GL',
            label: 'GL code',
            marking: 'neededToSubmit',
            parentAttributeId: 'SA-ST',
            options: [
              option('SO-GL-DIG', '7250080 – Advertising digital', ['SO-ST-ATL']),
              option('SO-GL-PRT', '7250090 – Advertising print', ['SO-ST-ATL']),
              option('SO-GL-EVT', '7260010 – Event costs', ['SO-ST-BTL']),
            ],
          }),
        ],
      },
    ],
  };
}

const PRIYA: Person = {
  ...PRIYA_REF,
  email: 'priya.raman@shell.com',
  teams: [{ id: 'T-1001', name: 'Lubricants Europe', colour: 'blue', active: true }],
  version: 1,
  createdAt: '2026-01-01T00:00:00Z',
  createdBy: ACTOR,
  updatedAt: '2026-01-01T00:00:00Z',
  updatedBy: ACTOR,
};

/** People, attributes and rates; the Business unit options come with the activity (D-212). */
export function formAnswers(): HttpHandler[] {
  return [
    reply('GET /api/v1/people', { body: { items: [PRIYA], total: 1 } }),
    reply(
      'GET /api/v1/shell-attribute-forms/{level}',
      { body: lineItemForm() },
      { level: 'lineItem' },
    ),
    reply('GET /api/v1/exchange-rates/current', { body: currentRates() }),
  ];
}

export function validationError(fields: { field: string; message: string }[]): {
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

/** The 400 that says the person must confirm "This activity will need approval again" (D-189). */
export function restartRefusal() {
  return validationError([
    {
      field: 'restartApproval',
      message:
        'This activity will need approval again. Confirm the change to send it back for approval.',
    },
  ]);
}

export function notFound(message: string): { status: number; body: ErrorBody } {
  return { status: 404, body: { error: { code: 'not_found', message, fields: [] } } };
}

/** The answer to a delete: the recycle-bin entry for Undo. */
export function deleteResult(name = 'Agency fee', id = 'L-1001') {
  return {
    recycleBinEntry: {
      id: 'BIN-1001',
      recordType: 'lineItem',
      recordTypeLabel: 'line item',
      recordId: id,
      recordName: name,
      path: activity().path,
      parent: { type: 'activity', id: 'A-1001', name: 'Launch film' },
      deletedAt: '2026-03-14T10:30:00Z',
      deletedBy: ACTOR,
      contents: [],
    },
  };
}

/** Today's rates, for the ⓘ that names the rates used (016 FR-009). */
export function currentRates() {
  return {
    today: '2026-03-12',
    total: 3,
    items: [
      {
        currency: 'USD',
        rate: '1',
        inverseRate: '1',
        rateDate: null,
        source: null,
        importFileName: null,
      },
      {
        currency: 'EUR',
        rate: '0.92',
        inverseRate: '1.086957',
        rateDate: '2026-03-01',
        source: 'manual' as const,
        importFileName: null,
      },
      {
        currency: 'GBP',
        rate: '0.79',
        inverseRate: '1.265823',
        rateDate: '2026-03-01',
        source: 'gsapImport' as const,
        importFileName: 'rates-2026-03-01.csv',
      },
    ],
  };
}
