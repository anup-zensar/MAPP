import { StatusChip } from '@/ui/StatusChip/StatusChip';
import type { PurchaseRequestStatus } from './api';
import { STATUS_TONES, statusText } from './purchase-request-text';

export interface PurchaseRequestStatusChipProps {
  status: PurchaseRequestStatus;
  changePending?: boolean;
}

/** A purchase request's status as a word on its family's tint (032 FR-052, Status Model). */
export function PurchaseRequestStatusChip({
  status,
  changePending = false,
}: PurchaseRequestStatusChipProps) {
  return (
    <StatusChip tone={changePending ? 'review' : STATUS_TONES[status]}>
      {statusText(status, changePending)}
    </StatusChip>
  );
}
