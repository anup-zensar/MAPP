import { formatAmount, parseMoneyInput } from '@/format/decimal';
import type {
  LinesOnApproved,
  PurchaseRequest,
  PurchaseRequestDefaults,
  PurchaseRequestInput,
  PurchaseRequestLine,
  PurchaseRequestLineInput,
  PurchaseRequestLineItemOption,
} from './api';
import type { LineValues, PurchaseRequestFormValues } from './form-schema';
import { SAVE, SAVE_AND_RESUBMIT, SUBMIT_CHANGES } from './form-text';

export type Then = PurchaseRequestInput['then'];

export interface SaveMode {
  then: Then;
  label: string;
}

export function fromDefaults(defaults: PurchaseRequestDefaults): PurchaseRequestFormValues {
  const { line } = defaults;
  return {
    name: defaults.name,
    description: '',
    ownerId: defaults.owner.id,
    moaApproverId: '',
    approvalWorkflowId: '',
    modificationWorkflowId: '',
    attributes: {},
    lines: [
      {
        id: '',
        lineItem: line.lineItem,
        hxKey: line.hxKey,
        currency: line.currency,
        amount: formatAmount(line.amount.amount),
        deliveryDate: '',
        overallLimit: formatAmount(line.overallLimit.amount),
        description: '',
        attributes: {},
      },
    ],
  };
}

function lineOf(line: PurchaseRequestLine): LineValues {
  return {
    id: line.id,
    lineItem: line.lineItem,
    hxKey: line.hxKey,
    currency: line.currency,
    amount: formatAmount(line.amount.amount),
    deliveryDate: line.deliveryDate,
    overallLimit: line.overallLimit ? formatAmount(line.overallLimit.amount) : '',
    description: line.description,
    attributes: line.attributes,
  };
}

export function fromPurchaseRequest(pr: PurchaseRequest): PurchaseRequestFormValues {
  return {
    name: pr.name,
    description: pr.description,
    ownerId: pr.owner?.id ?? '',
    moaApproverId: pr.moaApprover?.id ?? '',
    approvalWorkflowId: pr.approvalWorkflow?.id ?? '',
    modificationWorkflowId: pr.modificationWorkflow?.id ?? '',
    attributes: pr.attributes,
    lines: pr.lines.map(lineOf),
  };
}

/** A new line for a line item chosen in the picker: its whole available amount (032 FR-003). */
export function lineFromOption(option: PurchaseRequestLineItemOption): LineValues {
  const available = formatAmount(option.availableToRequest.amount);
  return {
    id: '',
    lineItem: option.lineItem,
    hxKey: option.hxKey,
    currency: option.availableToRequest.currency,
    amount: available,
    deliveryDate: '',
    overallLimit: available,
    description: '',
    attributes: {},
  };
}

function amountOf(text: string): string {
  return parseMoneyInput(text)?.amount ?? text.trim();
}

function lineInput(line: LineValues): PurchaseRequestLineInput {
  const limit = line.overallLimit.trim();
  return {
    ...(line.id ? { id: line.id } : {}),
    lineItemId: line.lineItem.id,
    currency: line.currency,
    amount: amountOf(line.amount),
    deliveryDate: line.deliveryDate,
    overallLimit: limit === '' ? null : amountOf(limit),
    description: line.description,
    attributes: line.attributes,
  };
}

export function toInput(
  values: PurchaseRequestFormValues,
  then: Then,
  version?: number,
): PurchaseRequestInput {
  return {
    ...(version === undefined ? {} : { version }),
    name: values.name.trim(),
    description: values.description,
    ownerId: values.ownerId,
    moaApproverId: values.moaApproverId || null,
    approvalWorkflowId: values.approvalWorkflowId,
    modificationWorkflowId: values.modificationWorkflowId,
    attributes: values.attributes,
    lines: values.lines.map(lineInput),
    then,
  };
}

/** Whether the only change since `start` is lines added (032 FR-039 "Allowed without approval"). */
export function onlyAddsLines(start: PurchaseRequestFormValues, now: PurchaseRequestFormValues) {
  const stored = { ...now, lines: now.lines.filter((line) => line.id !== '') };
  const added = now.lines.length > stored.lines.length;
  return (
    added && JSON.stringify(toInput(stored, 'save')) === JSON.stringify(toInput(start, 'save'))
  );
}

/** The footer's primary action by status (032 FR-023, FR-036, FR-039, Record Operations). */
export function saveModeOf(
  pr: PurchaseRequest | null,
  onlyAdded: boolean,
  linesOnApproved: LinesOnApproved | undefined,
): SaveMode {
  if (!pr || pr.status === 'created' || pr.status === 'declined') {
    return { then: 'save', label: SAVE };
  }
  if (pr.status === 'approved' && !pr.changePending) {
    return onlyAdded && linesOnApproved === 'withoutApproval'
      ? { then: 'save', label: SAVE }
      : { then: 'submitChanges', label: SUBMIT_CHANGES };
  }
  return { then: 'resubmit', label: SAVE_AND_RESUBMIT };
}
