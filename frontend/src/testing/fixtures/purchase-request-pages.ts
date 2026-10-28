import type { components } from '@/api/generated/schema';
import {
  GIFT_BOXES,
  money,
  PATH,
  PLAN_REF,
  PR_WORKFLOW,
  PRIYA_REF,
  STAND_BUILD,
  YES,
} from './purchase-request-refs';

type S = components['schemas'];

export function defaults(
  overrides: Partial<S['PurchaseRequestDefaults']> = {},
): S['PurchaseRequestDefaults'] {
  return {
    name: 'PR for Stand build',
    owner: PRIYA_REF,
    plan: PLAN_REF,
    companyCode: 'GB01',
    line: {
      lineItem: STAND_BUILD,
      hxKey: 'HX1006',
      currency: 'EUR',
      amount: money('9200.00'),
      overallLimit: money('9200.00'),
    },
    ...overrides,
  };
}

export function lineItemOption(
  overrides: Partial<S['PurchaseRequestLineItemOption']> = {},
): S['PurchaseRequestLineItemOption'] {
  return {
    lineItem: GIFT_BOXES,
    plan: PLAN_REF,
    path: PATH,
    hxKey: 'HX1011',
    availableToRequest: money('10000.00', 'USD'),
    available: true,
    reason: '',
    ...overrides,
  };
}

export function prRow(overrides: Partial<S['PurchaseRequestRow']> = {}): S['PurchaseRequestRow'] {
  return {
    id: 'PR-1002',
    name: 'PR for Stand build',
    vendor: 'Northlight Events GmbH',
    status: 'approved',
    changePending: false,
    amountDisplay: money('10000.00', 'USD'),
    invoicedDisplay: money('0.00', 'USD'),
    owner: PRIYA_REF,
    createdAt: '2026-10-27T09:00:00Z',
    approvalWorkflow: PR_WORKFLOW,
    gsapPrNumber: '',
    waitingForGsap: true,
    plan: PLAN_REF,
    lines: [
      {
        id: 'PRL-1002',
        lineItem: STAND_BUILD,
        lineItemOwner: PRIYA_REF,
        hxKey: 'HX1006',
        amount: money('9200.00'),
        deliveryDate: '2026-11-30',
        sentOn: '2026-10-27',
        poNumber: '',
        poLine: '',
        poLineStatus: '',
        invoiced: money('0.00'),
      },
    ],
    ...overrides,
  };
}

export const BRIGHTSIDE_ROW = prRow({
  id: 'PR-1001',
  name: 'PR for Media booking',
  vendor: 'Brightside Media BV',
  status: 'created',
  waitingForGsap: false,
  amountDisplay: money('30000.00', 'USD'),
  lines: [],
});

export function listPage(
  overrides: Partial<S['PurchaseRequestListPage']> = {},
): S['PurchaseRequestListPage'] {
  return {
    items: [BRIGHTSIDE_ROW, prRow()],
    total: 2,
    page: 1,
    pageSize: 50,
    vendors: [
      {
        vendor: 'Brightside Media BV',
        label: 'Brightside Media BV',
        purchaseRequests: 1,
        amountDisplay: money('30000.00', 'USD'),
        invoicedDisplay: money('0.00', 'USD'),
      },
      {
        vendor: 'Northlight Events GmbH',
        label: 'Northlight Events GmbH',
        purchaseRequests: 1,
        amountDisplay: money('10000.00', 'USD'),
        invoicedDisplay: money('0.00', 'USD'),
      },
    ],
    purchaseSupport: true,
    ...overrides,
  };
}

export function lineItemSection(
  overrides: Partial<S['LineItemPurchaseRequests']> = {},
): S['LineItemPurchaseRequests'] {
  return {
    lineItemId: 'L-1006',
    hxKey: 'HX1006',
    plannedCost: money('9200.00'),
    requested: money('6000.00'),
    availableToRequest: money('3200.00'),
    purchaseRequests: [
      {
        purchaseRequestId: 'PR-1002',
        name: 'PR for Stand build',
        status: 'created',
        changePending: false,
        lineId: 'PRL-1002',
        amount: money('6000.00'),
        amountInLineItemCurrency: money('6000.00'),
        sentOn: null,
        poNumber: '',
        owner: PRIYA_REF,
      },
    ],
    create: YES,
    notice: '',
    ...overrides,
  };
}

export function activitySection(
  overrides: Partial<S['ActivityPurchaseRequests']> = {},
): S['ActivityPurchaseRequests'] {
  return {
    activityId: 'A-1003',
    requested: money('6521.74', 'USD'),
    lineItems: [
      {
        lineItemId: 'L-1006',
        name: 'Stand build',
        requested: money('6000.00'),
        availableToRequest: money('3200.00'),
        purchaseRequests: 1,
      },
    ],
    ...overrides,
  };
}

export function configuration(
  overrides: Partial<S['PurchaseRequestConfiguration']> = {},
): S['PurchaseRequestConfiguration'] {
  return {
    purchaseSupport: true,
    linesOnApprovedPurchaseRequests: 'withModificationApproval',
    version: 1,
    updatedAt: '2026-10-20T08:00:00Z',
    updatedBy: { id: 'U-1001', name: 'Alex Morgan' },
    canChange: YES,
    ...overrides,
  };
}

export function closePreview(
  overrides: Partial<S['PurchaseRequestClosePreview']> = {},
): S['PurchaseRequestClosePreview'] {
  return {
    purchaseRequestId: 'PR-1002',
    version: 3,
    available: true,
    reason: '',
    lines: [
      {
        lineId: 'PRL-1002',
        lineItemId: 'L-1006',
        hxKey: 'HX1006',
        amount: money('9200.00'),
        invoiced: money('7500.00'),
        closedAmount: money('7500.00'),
        released: money('1700.00'),
      },
    ],
    releasedDisplay: money('1847.83', 'USD'),
    ...overrides,
  };
}

export const BIN_ENTRY: S['RecycleBinEntry'] = {
  id: 'BIN-1101',
  recordType: 'purchaseRequest',
  recordTypeLabel: 'purchase request',
  recordId: 'PR-1002',
  recordName: 'PR for Stand build',
  path: [PLAN_REF],
  parent: null,
  deletedAt: '2026-10-27T11:00:00Z',
  deletedBy: { id: 'U-1002', name: 'Priya Raman' },
  contents: [],
};
