import type { components } from '@/api/generated/schema';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import type { StatusTone } from '@/ui/StatusChip/StatusChip';
import { SWBS_TEXT } from './lineItemText';

type SwbsStatus = components['schemas']['SwbsStatus'];

const TONE: Record<SwbsStatus, StatusTone> = {
  notRequested: 'neutral',
  waitingForApproval: 'review',
  readyToSend: 'positive',
};

/**
 * The SWBS request status of a line item (FR-024, data model §3.5): "Not requested" (grey),
 * "Waiting for activity approval" (amber), "Ready to send to GSAP" (green).
 */
export function SwbsChip({ status }: { status: SwbsStatus }) {
  return <StatusChip tone={TONE[status]}>{SWBS_TEXT[status]}</StatusChip>;
}
