import type { ApprovalWorkflow } from './api';

type Kind = ApprovalWorkflow['objectType'];
type Usage = Pick<ApprovalWorkflow, 'usedByCount' | 'waitingCount' | 'objectType'>;

export const WORKFLOW_GONE =
  'This approval workflow no longer exists — someone may have deleted it.';

const WORDS: Record<Kind, { one: string; many: string }> = {
  activity: { one: 'activity', many: 'activities' },
  purchaseRequest: { one: 'purchase request', many: 'purchase requests' },
  modifiedPurchaseRequest: { one: 'purchase request', many: 'purchase requests' },
};

/** "1 activity", "3 purchase requests": the records a workflow of this kind approves. */
export function recordsCount(count: number, kind: Kind): string {
  return count === 1 ? `1 ${WORDS[kind].one}` : `${count} ${WORDS[kind].many}`;
}

export function recordsWord(kind: Kind): string {
  return WORDS[kind].many;
}

export function usedByText({ usedByCount, waitingCount, objectType }: Usage): string {
  if (usedByCount === 0) {
    return 'Not used yet';
  }
  const waiting = waitingCount > 0 ? ` (${waitingCount} waiting for approval)` : '';
  return `Used by ${recordsCount(usedByCount, objectType)}${waiting}`;
}

export function openRoundsNote({ waitingCount, objectType }: Usage): string {
  const { one, many } = WORDS[objectType];
  const keep =
    waitingCount === 1
      ? `1 ${one} waiting for approval keeps the levels it was submitted with.`
      : `${waitingCount} ${many} waiting for approval keep the levels they were submitted with.`;
  return `Changes apply to ${many} submitted from now on. ${keep}`;
}

export function setInactiveEffects({ usedByCount, waitingCount, objectType }: Usage): string[] {
  const { one, many } = WORDS[objectType];
  const effects = [`It will no longer be offered on ${many}.`];
  if (usedByCount === 1) {
    effects.push(`The 1 ${one} that uses it keeps it, shown as '(inactive)'.`);
  } else if (usedByCount > 1) {
    effects.push(`The ${usedByCount} ${many} that use it keep it, shown as '(inactive)'.`);
  }
  if (waitingCount === 1) {
    effects.push(`1 ${one} waiting for approval finishes with the levels it was submitted with.`);
  } else if (waitingCount > 1) {
    effects.push(
      `${waitingCount} ${many} waiting for approval finish with the levels they were submitted with.`,
    );
  }
  return effects;
}

/** The API's in_use message, word for word. */
export function cantDeleteText({ usedByCount, objectType }: Usage): string {
  return `Used by ${recordsCount(usedByCount, objectType)} — set it inactive instead.`;
}

const KIND_LABELS: Record<Kind, { label: string; plural: string }> = {
  activity: { label: 'Activities', plural: 'activities' },
  purchaseRequest: { label: 'Purchase requests', plural: 'purchase requests' },
  modifiedPurchaseRequest: {
    label: 'Modified purchase requests',
    plural: 'modified purchase requests',
  },
};

export const OBJECT_TYPES = Object.entries(KIND_LABELS).map(([id, { label }]) => ({
  id,
  label,
  active: true,
}));

export function isObjectType(value: string): value is Kind {
  return Object.hasOwn(KIND_LABELS, value);
}

export function objectTypeLabel(kind: Kind): string {
  return KIND_LABELS[kind].label;
}

/** The API's message when a used workflow's object type is changed, word for word. */
export function objectTypeFixedText(kind: Kind): string {
  return `It approves ${KIND_LABELS[kind].plural} and is in use, so this can't change. Create a new workflow instead.`;
}
