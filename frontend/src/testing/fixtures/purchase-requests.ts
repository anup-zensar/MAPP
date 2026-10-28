import type { components } from '@/api/generated/schema';
import {
  DANIEL_REF,
  HIDDEN,
  money,
  no,
  PATH,
  PLAN_REF,
  PR_CHANGE_WORKFLOW,
  PR_WORKFLOW,
  PRIYA_REF,
  STAND_BUILD,
  YES,
} from './purchase-request-refs';

// The demo data of 032/033: PR-1002 "PR for Stand build" with one line on L-1006 (9,200.00 EUR).

type S = components['schemas'];

export const LINE_ACCESS: S['PurchaseRequestLineAccess'] = { remove: YES };

export function prLine(
  overrides: Partial<S['PurchaseRequestLine']> = {},
): S['PurchaseRequestLine'] {
  return {
    id: 'PRL-1002',
    position: 1,
    lineItem: STAND_BUILD,
    hxKey: 'HX1006',
    currency: 'EUR',
    amount: money('9200.00'),
    deliveryDate: '2026-11-30',
    overallLimit: money('9200.00'),
    description: '',
    attributes: {},
    sentOn: null,
    poNumber: '',
    poLine: '',
    poLineStatus: '',
    poDate: null,
    poValue: null,
    invoiced: money('0.00'),
    changed: false,
    missing: [],
    access: LINE_ACCESS,
    gsapPending: false,
    version: 1,
    ...overrides,
  };
}

export const PR_ACCESS: S['PurchaseRequestAccess'] = {
  edit: YES,
  delete: YES,
  submit: YES,
  resubmit: HIDDEN,
  submitChanges: HIDDEN,
  recall: HIDDEN,
  close: no('Only approved purchase requests can be closed.'),
  resend: no('The PR form is created when the purchase request is approved.'),
  addLine: YES,
  gsapOrder: HIDDEN,
};

export const ROUTING: S['PurchaseRequestRouting'] = {
  id: 'PRR-1001',
  type: 'original',
  routedAt: '2026-10-27T10:15:00Z',
  routedBy: { id: 'U-1003', name: 'Daniel Okoro' },
  recipients: ['DE_LUBES'],
  lineCount: 1,
};

export function purchaseRequest(
  overrides: Partial<S['PurchaseRequest']> = {},
): S['PurchaseRequest'] {
  return {
    id: 'PR-1002',
    name: 'PR for Stand build',
    description: '',
    owner: PRIYA_REF,
    moaApprover: DANIEL_REF,
    approvalWorkflow: PR_WORKFLOW,
    modificationWorkflow: PR_CHANGE_WORKFLOW,
    status: 'created',
    changePending: false,
    plan: PLAN_REF,
    path: PATH,
    companyCode: 'GB01',
    costCentres: ['GB01000105'],
    hxKeys: ['HX1006'],
    attributes: {},
    vendor: 'Northlight Events GmbH',
    amountDisplay: money('10000.00', 'USD'),
    invoicedDisplay: money('0.00', 'USD'),
    gsapPrNumber: '',
    gsapPrDate: null,
    gsapSource: '',
    waitingForGsap: false,
    lines: [prLine()],
    approvedVersion: null,
    routings: [],
    readiness: { ready: true, problems: [] },
    missing: [],
    notice: '',
    access: PR_ACCESS,
    version: 3,
    createdAt: '2026-10-27T09:00:00Z',
    createdBy: { id: 'U-1002', name: 'Priya Raman' },
    updatedAt: '2026-10-27T09:30:00Z',
    updatedBy: { id: 'U-1002', name: 'Priya Raman' },
    ...overrides,
  };
}

/** PR-1002 approved and routed to DE_LUBES, waiting for GSAP; Lea may run the simulated GSAP. */
export function routedPurchaseRequest(
  overrides: Partial<S['PurchaseRequest']> = {},
): S['PurchaseRequest'] {
  return purchaseRequest({
    status: 'approved',
    waitingForGsap: true,
    routings: [ROUTING],
    lines: [
      prLine({
        sentOn: '2026-10-27',
        gsapPending: true,
        access: {
          remove: no(
            'Lines of approved purchase requests cannot be removed — close the purchase request instead.',
          ),
        },
      }),
    ],
    access: {
      ...PR_ACCESS,
      edit: YES,
      delete: no('Approved purchase requests cannot be deleted — close them instead.'),
      submit: HIDDEN,
      submitChanges: YES,
      close: YES,
      resend: YES,
      gsapOrder: YES,
    },
    ...overrides,
  });
}

/** PR-1002 after the simulated GSAP answered (033). */
export function orderedPurchaseRequest(
  overrides: Partial<S['PurchaseRequest']> = {},
): S['PurchaseRequest'] {
  const routed = routedPurchaseRequest();
  return {
    ...routed,
    waitingForGsap: false,
    gsapPrNumber: 'SIM-0010001001',
    gsapPrDate: '2026-10-27',
    gsapSource: 'simulatedGsap',
    lines: routed.lines.map((line) => ({
      ...line,
      poNumber: 'SIM-4500001001',
      poLine: '10',
      poLineStatus: 'Released',
      poDate: '2026-10-27',
      poValue: money('9200.00'),
      gsapPending: false,
    })),
    access: {
      ...routed.access,
      gsapOrder: no(
        'The simulated GSAP already created the PO for this purchase request. Run it again after an approved change of its amounts or lines.',
      ),
    },
    ...overrides,
  };
}

export * from './purchase-request-refs';
export * from './purchase-request-approvals';
export * from './purchase-request-pages';
