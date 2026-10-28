import type { components } from '@/api/generated/schema';
import { recordHref } from './record-links';

type SubjectKind = components['schemas']['ApprovalSubjectKind'];

const LABEL: Record<SubjectKind, string> = {
  activity: 'Activity',
  purchaseRequest: 'Purchase request',
  modifiedPurchaseRequest: 'Modified purchase request',
};

export function subjectKindLabel(kind: SubjectKind): string {
  return LABEL[kind];
}

/** The page of a record that is approved; a purchase request's page is feature 032's /purchase-requests/<id>. */
export function subjectHref(kind: SubjectKind, id: string): string {
  switch (kind) {
    case 'activity':
      return recordHref('activity', id);
    case 'purchaseRequest':
    case 'modifiedPurchaseRequest':
      return `/purchase-requests/${encodeURIComponent(id)}`;
  }
}
