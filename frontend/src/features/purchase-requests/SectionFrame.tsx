import { useId } from 'react';
import type { ReactNode } from 'react';
import { ApiError } from '@/api/errors';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { InfoTip } from '@/ui/InfoTip/InfoTip';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { PURCHASE_REQUESTS } from './purchase-request-text';
import { NOT_VISIBLE } from './section-text';
import styles from './PurchaseRequestSections.module.css';

export interface SectionFrameProps {
  tip: string;
  action?: ReactNode;
  children: ReactNode;
}

/** The Purchase requests card of a record page: open, never collapsed, with its ⓘ. */
export function SectionFrame({ tip, action, children }: SectionFrameProps) {
  const titleId = useId();
  return (
    <section id="purchase-requests" className={styles.card} aria-labelledby={titleId}>
      <div className={styles.header}>
        <h2 id={titleId} className={styles.title}>
          {PURCHASE_REQUESTS}
        </h2>
        <InfoTip label={PURCHASE_REQUESTS} text={tip} />
        {action && <div className={styles.action}>{action}</div>}
      </div>
      <div className={styles.body}>{children}</div>
    </section>
  );
}

export function SectionNotLoaded({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  if (error instanceof ApiError && (error.kind === 'forbidden' || error.kind === 'notFound')) {
    return <p className={styles.note}>{NOT_VISIBLE}</p>;
  }
  return error ? <ErrorState error={error} onRetry={onRetry} /> : <Skeleton lines={3} />;
}
