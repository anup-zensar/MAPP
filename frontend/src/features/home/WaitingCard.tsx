import { Hourglass } from 'lucide-react';
import { Link } from 'react-router';
import { LinkButton } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { recordHref } from '@/ui/record-links';
import { useFormatters } from '@/format/useFormatters';
import { useHomeWaiting } from './api';
import { waitingForMeHref } from './home-links';
import {
  NOTHING_WAITING,
  WAITING_SUBTITLE,
  levelText,
  viewAllText,
  waitingText,
} from './home-text';
import { SectionError } from './SectionError';
import styles from './WaitingCard.module.css';

/**
 * Waiting for my approval: the activities of 008's Waiting for me tab, oldest first, at most 5,
 * each with Review opening its preview on the Approvals page, and "View all (n)" that tab.
 */
export function WaitingCard() {
  const waiting = useHomeWaiting();
  const { money } = useFormatters();

  return (
    <Card title="Waiting for my approval" subtitle={WAITING_SUBTITLE}>
      {waiting.isPending ? (
        <Skeleton lines={4} height="18px" />
      ) : waiting.isError ? (
        <SectionError onRetry={() => void waiting.refetch()} />
      ) : waiting.data.items.length === 0 ? (
        <EmptyState icon={<Hourglass />} title={NOTHING_WAITING} />
      ) : (
        <>
          <ul className={styles.rows}>
            {waiting.data.items.map((item) => (
              <li key={item.id} className={styles.row}>
                <div className={styles.main}>
                  <div className={styles.head}>
                    <Link
                      to={recordHref('activity', item.id, item.plan.id)}
                      className={styles.name}
                    >
                      {item.name}
                    </Link>
                    <span className={styles.id}>{item.id}</span>
                  </div>
                  <p className={styles.meta}>
                    {[item.plan.name, item.owner?.name, money(item.plannedCost)]
                      .filter(Boolean)
                      .join(' · ')}
                  </p>
                  <p className={styles.meta}>
                    {levelText(item.level, item.levels)} · {waitingText(item.waitingSince)}
                  </p>
                </div>
                <LinkButton
                  size="sm"
                  to={waitingForMeHref(item.id)}
                  aria-label={`Review ${item.name}`}
                >
                  Review
                </LinkButton>
              </li>
            ))}
          </ul>
          {waiting.data.total > waiting.data.items.length && (
            <p className={styles.more}>
              <Link to={waitingForMeHref()}>{viewAllText(waiting.data.total)}</Link>
            </p>
          )}
        </>
      )}
    </Card>
  );
}
