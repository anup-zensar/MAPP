import type { components } from '@/api/generated/schema';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import type { StatusTone } from '@/ui/StatusChip/StatusChip';
import { SWBS_TEXT } from './swbs-text';

type SwbsStatus = components['schemas']['SwbsStatus'];

const TONE: Record<SwbsStatus, StatusTone> = {
  notRequested: 'neutral',
  waitingForApproval: 'review',
  readyToSend: 'positive',
  received: 'done',
};

/** The SWBS request status as a word and a colour (domain model §2.4, 030 FR-017). */
export function SwbsChip({ status }: { status: SwbsStatus }) {
  return <StatusChip tone={TONE[status]}>{SWBS_TEXT[status]}</StatusChip>;
}
