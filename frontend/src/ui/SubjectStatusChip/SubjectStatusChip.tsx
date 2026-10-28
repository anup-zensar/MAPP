import type { components } from '@/api/generated/schema';
import { StatusChip } from '../StatusChip/StatusChip';
import type { StatusTone } from '../StatusChip/StatusChip';

export type SubjectStatus = components['schemas']['ApprovalSubjectStatus'];

const STATUS: Record<SubjectStatus, { word: string; tone: StatusTone }> = {
  created: { word: 'Created', tone: 'neutral' },
  submitted: { word: 'Submitted', tone: 'review' },
  inReview: { word: 'In review', tone: 'review' },
  declined: { word: 'Declined', tone: 'problem' },
  approved: { word: 'Approved', tone: 'positive' },
  inProgress: { word: 'In progress', tone: 'active' },
  complete: { word: 'Complete', tone: 'done' },
  closed: { word: 'Closed', tone: 'done' },
};

export interface SubjectStatusChipProps {
  status: SubjectStatus;
}

/** The status chip of any record an approval workflow approves: an activity or a purchase request. */
export function SubjectStatusChip({ status }: SubjectStatusChipProps) {
  return <StatusChip tone={STATUS[status].tone}>{STATUS[status].word}</StatusChip>;
}
