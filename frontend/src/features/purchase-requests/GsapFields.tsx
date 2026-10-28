import { Card } from '@/ui/Card/Card';
import { useFormatters } from '@/format/useFormatters';
import type { PurchaseRequest } from './api';
import { Detail, DetailsList } from './Details';
import { GSAP_EMPTY, GSAP_SUBTITLE, GSAP_TITLE } from './page-text';
import { FIELD_LABELS, FILLED_BY_GSAP, GSAP_SOURCE_WORDS } from './purchase-request-text';
import styles from './PurchaseRequestPage.module.css';

/** GSAP's answer on the header (032 FR-009; 033 FR-004): read-only, "Simulated GSAP" for now. */
export function GsapFields({ purchaseRequest }: { purchaseRequest: PurchaseRequest }) {
  const { date } = useFormatters();
  const { gsapPrNumber, gsapPrDate, gsapSource } = purchaseRequest;
  return (
    <Card title={GSAP_TITLE} subtitle={GSAP_SUBTITLE}>
      <div className={styles.stack}>
        {gsapSource === '' && <p className={styles.note}>{GSAP_EMPTY}</p>}
        <DetailsList>
          <Detail label={FIELD_LABELS.gsapPrNumber} tip={FILLED_BY_GSAP} numeric>
            {gsapPrNumber}
          </Detail>
          <Detail label={FIELD_LABELS.gsapPrDate} tip={FILLED_BY_GSAP}>
            {gsapPrDate ? date(gsapPrDate) : null}
          </Detail>
          <Detail label={FIELD_LABELS.gsapSource} tip={FILLED_BY_GSAP}>
            {GSAP_SOURCE_WORDS[gsapSource]}
          </Detail>
        </DetailsList>
      </div>
    </Card>
  );
}
