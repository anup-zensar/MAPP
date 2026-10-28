import { SendNowButton } from '@/features/simulated-gsap';
import { useAccess } from '@/session/access';
import { Card } from '@/ui/Card/Card';
import { useFormatters } from '@/format/useFormatters';
import { Detail, DetailsList } from './DetailsList';
import { HxKey } from './HxKey';
import type { LineItem } from './api';
import {
  NOT_RECEIVED,
  SIMULATED_GSAP,
  SWBS_CARD_TITLE,
  SWBS_EXPLANATION,
  SWBS_LABEL,
} from './swbs-text';
import { SwbsChip } from './SwbsChip';
import styles from './SwbsCard.module.css';

/** The line item's SWBS request (030 FR-014): status, code, dates, source, HX key and Send now. */
export function SwbsCard({ lineItem }: { lineItem: LineItem }) {
  const { dateTime } = useFormatters();
  const canSend = useAccess().has('finance.import') && lineItem.swbsStatus === 'readyToSend';
  const received = lineItem.swbsStatus === 'received';
  const timeOf = (timestamp: string | null) => (timestamp ? dateTime(timestamp) : null);

  return (
    <Card title={SWBS_CARD_TITLE}>
      <div className={styles.stack}>
        <p className={styles.status}>
          <SwbsChip status={lineItem.swbsStatus} />
          <span className={styles.note}>{SWBS_EXPLANATION[lineItem.swbsStatus]}</span>
        </p>
        {canSend && <SendNowButton lineItemId={lineItem.id} lineItemName={lineItem.name} />}
        <DetailsList>
          <Detail label={SWBS_LABEL.code}>
            {received ? <span className={styles.code}>{lineItem.swbsCode}</span> : NOT_RECEIVED}
          </Detail>
          {received && <Detail label={SWBS_LABEL.source}>{SIMULATED_GSAP}</Detail>}
          <Detail label={SWBS_LABEL.requested}>{timeOf(lineItem.swbsRequestedAt)}</Detail>
          <Detail label={SWBS_LABEL.received}>{timeOf(lineItem.swbsReceivedAt)}</Detail>
          <Detail label={SWBS_LABEL.hxKey}>
            <HxKey value={lineItem.hxKey} />
          </Detail>
        </DetailsList>
      </div>
    </Card>
  );
}
