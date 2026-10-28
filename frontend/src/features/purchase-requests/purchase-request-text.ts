import type { StatusTone } from '@/ui/StatusChip/StatusChip';
import type {
  GsapSource,
  LinesOnApproved,
  PurchaseRequestRoutingType,
  PurchaseRequestStatus,
} from './api';

export const PURCHASE_REQUESTS = 'Purchase requests';
export const PURCHASE_REQUEST = 'Purchase request';
export const NEW_PURCHASE_REQUEST = 'New purchase request';
export const EDIT_PURCHASE_REQUEST = 'Edit purchase request';

export const STATUS_WORDS: Record<PurchaseRequestStatus, string> = {
  created: 'Created',
  submitted: 'Submitted',
  inReview: 'In review',
  declined: 'Declined',
  approved: 'Approved',
  closed: 'Closed',
};

export const STATUS_TONES: Record<PurchaseRequestStatus, StatusTone> = {
  created: 'neutral',
  submitted: 'review',
  inReview: 'review',
  declined: 'problem',
  approved: 'positive',
  closed: 'done',
};

export const CHANGE_PENDING = 'Change in review';

export const ROUTING_TYPE_WORDS: Record<PurchaseRequestRoutingType, string> = {
  original: 'Original',
  modification: 'Modification',
  resend: 'Resend',
  close: 'Close',
};

export const GSAP_SOURCE_WORDS: Record<GsapSource, string> = {
  '': '',
  simulatedGsap: 'Simulated GSAP',
  gsap: 'GSAP',
};

export const LINES_ON_APPROVED_WORDS: Record<LinesOnApproved, string> = {
  notAllowed: 'Not allowed',
  withModificationApproval: 'Allowed with modification approval',
  withoutApproval: 'Allowed without approval',
};

export const SWITCHED_OFF = 'Purchase requests are switched off for your organisation.';
export const FILLED_BY_GSAP = 'Filled by GSAP';
export const HX_LABEL = 'Generated from the line-item ID';
export const SWBS_LABEL = 'Filled by GSAP through the line item';
export const NO_FORM_YET = 'The PR form is created when the purchase request is approved.';
export const AWAITING_NOTICE = 'Awaiting approval — saving changes will restart approval.';
export const CLOSED_NO_CHANGE = 'Closed purchase requests cannot be changed.';
export const SIMULATED_GSAP = 'Simulated GSAP';

export const FIELD_LABELS = {
  name: 'Name',
  description: 'Description',
  owner: 'Owner',
  moaApprover: 'MoA approver',
  approvalWorkflow: 'Approval workflow',
  modificationWorkflow: 'Approval workflow for modification',
  status: 'Status',
  amount: 'Total amount',
  companyCode: 'Company code',
  costCentres: 'Cost centre',
  hxKeys: 'HX keys',
  vendor: 'Vendor',
  invoiced: 'Total invoiced',
  gsapPrNumber: 'GSAP PR number',
  gsapPrDate: 'GSAP PR date',
  gsapSource: 'Source',
  lineItem: 'Line item',
  hxKey: 'HX key',
  currency: 'Currency',
  lineAmount: 'Amount',
  deliveryDate: 'Delivery date',
  overallLimit: 'Overall limit',
  swbsCode: 'SWBS code',
  sentOn: 'Sent date',
  poNumber: 'PO number',
  poLine: 'PO line',
  poLineStatus: 'PO line status',
  poDate: 'PO date',
  poValue: 'PO value',
  lineInvoiced: 'Invoiced',
} as const;

/** "Created", or "Approved · Change in review" while a change of an approved PR waits. */
export function statusText(status: PurchaseRequestStatus, changePending: boolean): string {
  return changePending ? `${STATUS_WORDS[status]} · ${CHANGE_PENDING}` : STATUS_WORDS[status];
}

/** "1 line", "3 lines". */
export function linesText(count: number): string {
  return count === 1 ? '1 line' : `${count} lines`;
}

/** "PR for Stand build (PR-1002)". */
export function nameWithId(name: string, id: string): string {
  return `${name} (${id})`;
}
