import { CircleAlert, CircleCheckBig } from 'lucide-react';
import { useState } from 'react';
import { Link } from 'react-router';
import { Button, Card, EmptyState, LevelTag, Skeleton } from '@/components';
import { recordHref } from '@/lib/recordLinks';
import { useFormatters } from '@/lib/useFormatters';
import styles from './AttentionCard.module.css';
import { useHomeAttention } from './homeApi';
import type { HomeAttentionItem } from './homeApi';
import type { ReasonFormatters } from './homeText';
import {
  NOTHING_NEEDS_ATTENTION,
  REASON_LINK,
  reasonLinkName,
  reasonText,
  showAllText,
} from './homeText';
import { SectionError } from './SectionError';

/** How many items show before "Show all <n>" (FR-012). */
const SHOWN = 8;

/**
 * Needs your attention (FR-011, FR-012): the plans, campaigns, initiatives and activities the
 * person owns, in every plan, that have a problem — each once, with all its reasons and a link to
 * where each is fixed. The API sends them in order: declined, balance, budget, missing fields.
 */
export function AttentionCard() {
  const attention = useHomeAttention();
  const { money, date } = useFormatters();
  const [showAll, setShowAll] = useState(false);

  return (
    <Card title="Needs your attention" subtitle="Things you own that have a problem">
      {attention.isPending ? (
        <Skeleton lines={4} height="18px" />
      ) : attention.isError ? (
        <SectionError onRetry={() => void attention.refetch()} />
      ) : attention.data.items.length === 0 ? (
        <EmptyState icon={<CircleCheckBig />} title={NOTHING_NEEDS_ATTENTION} />
      ) : (
        <>
          <ul className={styles.items}>
            {(showAll ? attention.data.items : attention.data.items.slice(0, SHOWN)).map((item) => (
              <AttentionRow
                key={`${item.recordType}-${item.id}`}
                item={item}
                format={{ money, date }}
              />
            ))}
          </ul>
          {attention.data.items.length > SHOWN && (
            <Button
              variant="tertiary"
              size="sm"
              className={styles.more}
              aria-expanded={showAll}
              onClick={() => setShowAll(!showAll)}
            >
              {showAll ? 'Show fewer' : showAllText(attention.data.total)}
            </Button>
          )}
        </>
      )}
    </Card>
  );
}

function AttentionRow({ item, format }: { item: HomeAttentionItem; format: ReasonFormatters }) {
  const href = recordHref(item.recordType, item.id, item.plan.id);
  return (
    <li className={styles.item}>
      <div className={styles.head}>
        <LevelTag level={item.recordType} />
        <Link to={href} className={styles.name}>
          {item.name}
        </Link>
        <span className={styles.id}>{item.id}</span>
      </div>
      {item.recordType !== 'plan' && <p className={styles.where}>in {item.plan.name}</p>}
      <ul className={styles.reasons}>
        {item.reasons.map((reason) => {
          const link = REASON_LINK[reason.kind];
          return (
            <li key={reason.kind} className={styles.reason}>
              <CircleAlert aria-hidden="true" className={styles.icon} />
              <span>
                {reasonText(reason, format)}{' '}
                <Link
                  to={`${href}${link.hash}`}
                  aria-label={reasonLinkName(link.label, item.recordTypeLabel, item.name)}
                  className={styles.fix}
                >
                  {link.label}
                </Link>
              </span>
            </li>
          );
        })}
      </ul>
    </li>
  );
}
