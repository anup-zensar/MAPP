import { z } from 'zod';
import type { components } from '@/api/generated/schema';
import type { ApprovalDelegation, DelegationCreate, DelegationUpdate } from './delegations-api';

type SubjectKind = components['schemas']['ApprovalSubjectKind'];

const pageTabSchema = z.enum(['waiting', 'open', 'decided', 'monitoring']);
const subjectKindSchema = z.enum([
  'activity',
  'purchaseRequest',
  'modifiedPurchaseRequest',
] satisfies SubjectKind[]);

export type PageTab = z.infer<typeof pageTabSchema>;

export interface PreviewSubject {
  id: string;
  kind: SubjectKind;
}

/** What the approvals page's address says: ?tab=, ?activity= with ?kind= and ?panel=delegations. */
export interface ApprovalsAddress {
  /** The tab the address names; null when it names none, or one that does not exist. */
  tab: PageTab | null;
  previewId: string | null;
  previewKind: SubjectKind;
  delegationsOpen: boolean;
}

export function approvalsAddressOf(params: URLSearchParams): ApprovalsAddress {
  const tab = pageTabSchema.safeParse(params.get('tab'));
  const previewId = params.get('activity');
  return {
    tab: tab.success ? tab.data : null,
    previewId,
    previewKind: previewKindOf(params.get('kind'), previewId),
    delegationsOpen: params.get('panel') === 'delegations',
  };
}

// Links made before ?kind= existed name a purchase request by its PR- number alone.
function previewKindOf(raw: string | null, previewId: string | null): SubjectKind {
  const kind = subjectKindSchema.safeParse(raw);
  if (kind.success) {
    return kind.data;
  }
  return raw === null && previewId?.startsWith('PR-') ? 'purchaseRequest' : 'activity';
}

export function previewParams(subject: PreviewSubject | null): Record<string, string | null> {
  return {
    activity: subject?.id ?? null,
    kind: subject && subject.kind !== 'activity' ? subject.kind : null,
  };
}

/** The API checks every field of a delegation, so the form checks none. */
export const delegationSchema = z.object({
  personId: z.string(),
  delegateId: z.string(),
  startDate: z.string(),
  endDate: z.string(),
  note: z.string(),
});

export type DelegationValues = z.infer<typeof delegationSchema>;

export function delegationValuesOf(
  delegation: ApprovalDelegation | undefined,
  userId: string,
): DelegationValues {
  return {
    personId: delegation?.person.id ?? userId,
    delegateId: delegation?.delegate.id ?? '',
    startDate: delegation?.startDate ?? '',
    endDate: delegation?.endDate ?? '',
    note: delegation?.note ?? '',
  };
}

/** The new delegation; the person is sent only when someone who manages others chose another. */
export function newDelegation(
  values: DelegationValues,
  userId: string,
  canManageOthers: boolean,
): DelegationCreate {
  return {
    ...(canManageOthers && values.personId !== userId ? { personId: values.personId } : {}),
    delegateId: values.delegateId,
    startDate: values.startDate,
    endDate: values.endDate,
    ...(values.note.trim() ? { note: values.note.trim() } : {}),
  };
}

export function delegationChanges(
  values: DelegationValues,
  start: DelegationValues,
  version: number,
): DelegationUpdate {
  const changes: DelegationUpdate = { version };
  if (values.delegateId !== start.delegateId) {
    changes.delegateId = values.delegateId;
  }
  if (values.startDate !== start.startDate) {
    changes.startDate = values.startDate;
  }
  if (values.endDate !== start.endDate) {
    changes.endDate = values.endDate;
  }
  if (values.note !== start.note) {
    changes.note = values.note.trim();
  }
  return changes;
}

export const commentSchema = z.object({ comment: z.string() });

export type CommentValues = z.infer<typeof commentSchema>;

export const recallSchema = z.object({ reason: z.string() });

/** Declining needs a comment, checked before anything is sent. */
export const declineSchema = z.object({
  comment: z
    .string()
    .refine(
      (comment) => comment.trim() !== '',
      'Add a comment explaining why the activity is declined.',
    ),
});
