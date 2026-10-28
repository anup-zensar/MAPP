import type { HttpHandler } from 'msw';
import type { components } from '@/api/generated/schema';
import { reply } from '../replies';
import { workflowOf } from './approval-workflows';
import { attribute } from './attribute-forms';
import { PEOPLE } from './people';
import { PR_CHANGE_WORKFLOW, PR_WORKFLOW, configuration } from './purchase-requests';
import { CURRENCIES } from './settings';

type AttributeForm = components['schemas']['AttributeForm'];

/** The answers every purchase-request form needs besides the PR or its defaults (032). */

export function purchaseRequestAttributeForm(): AttributeForm {
  return {
    level: 'purchaseRequest',
    businessUnitAttributeId: null,
    businessUnitOptions: [],
    sections: [
      {
        name: 'Purchase request',
        conditional: false,
        attributes: [
          attribute({ id: 'SA-1101', label: 'Vendor', type: 'text', marking: 'neededToSubmit' }),
        ],
      },
    ],
  };
}

export function lineAttributeForm(): AttributeForm {
  return {
    level: 'purchaseRequestLine',
    businessUnitAttributeId: null,
    businessUnitOptions: [],
    sections: [
      {
        name: 'PR line',
        conditional: false,
        attributes: [attribute({ id: 'SA-1201', label: 'Material group', type: 'text' })],
      },
    ],
  };
}

export const PR_WORKFLOWS = {
  items: [
    workflowOf({ id: PR_WORKFLOW.id, name: PR_WORKFLOW.name, objectType: 'purchaseRequest' }),
    workflowOf({
      id: PR_CHANGE_WORKFLOW.id,
      name: PR_CHANGE_WORKFLOW.name,
      objectType: 'modifiedPurchaseRequest',
    }),
    workflowOf(),
  ],
  total: 3,
};

export function formAnswers(): HttpHandler[] {
  return [
    reply(
      'GET /api/v1/shell-attribute-forms/{level}',
      { body: purchaseRequestAttributeForm() },
      { level: 'purchaseRequest' },
    ),
    reply(
      'GET /api/v1/shell-attribute-forms/{level}',
      { body: lineAttributeForm() },
      { level: 'purchaseRequestLine' },
    ),
    reply('GET /api/v1/approval-workflows', { body: PR_WORKFLOWS }),
    reply('GET /api/v1/currencies', { body: CURRENCIES }),
    reply('GET /api/v1/people', { body: PEOPLE }),
    reply('GET /api/v1/purchase-request-configuration', { body: configuration() }),
  ];
}
