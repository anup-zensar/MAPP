import { z } from 'zod';
import type { ApprovalDelegation, DelegationCreate, DelegationUpdate } from './delegations-api';

const pageTabSchema = z.enum(['waiting', 'open', 'decided', 'monitoring']);

export type PageTab = z.infer<typeof pageTabSchema>;

/** What the approvals page's address says: ?tab=, ?activity= and ?panel=delegations. */
export interface ApprovalsAddress {
  /** The tab the address names; null when it names none, or one that does not exist. */
  tab: PageTab | null;
  previewId: string | null;
  delegationsOpen: boolean;
}

export function approvalsAddressOf(params: URLSearchParams): ApprovalsAddress {
  const tab = pageTabSchema.safeParse(params.get('tab'));
  return {
    tab: tab.success ? tab.data : null,
    previewId: params.get('activity'),
    delegationsOpen: params.get('panel') === 'delegations',
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
