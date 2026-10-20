import { ChevronRight, ListChecks } from 'lucide-react';
import { useId, useState } from 'react';
import { Link, useNavigate } from 'react-router';
import { Button, Card, CompletenessChip, EmptyState, LinkButton, Skeleton } from '@/components';
import { ActivityStatusChip } from '@/features/activities/ActivityStatusChip';
import { cx } from '@/lib/cx';
import { formatWhole } from '@/lib/decimal';
import { fieldHref, recordHref } from '@/lib/recordLinks';
import { useFormatters } from '@/lib/useFormatters';
import { useHomeMyActivities } from './homeApi';
import type { MyActivityGroup, MyActivityRow } from './homeApi';
import { NO_ACTIVITIES, datesText, lineItemsNeedAttentionText, showAllText } from './homeText';
import styles from './MyActivitiesCard.module.css';
import { SectionError } from './SectionError';

/** How many activities a group shows before "Show all <n>" (FR-014). */
const SHOWN = 5;

/**
 * My activities (FR-014): the activities the person owns, in every plan, grouped by the status
 * people see — Declined, Created, Submitted, Approved, In progress, Complete — with Complete
 * folded. "Activities I own" leads here while All work is not built (D-174).
 */
export function MyActivitiesCard() {
  const activities = useHomeMyActivities();

  return (
    // The card's place on Home, for the "Activities I own" card's link.
    <div id="my-activities" className={styles.anchor}>
      <Card title="My activities" subtitle="Activities you own, by status">
        {activities.isPending ? (
          <Skeleton lines={4} height="18px" />
        ) : activities.isError ? (
          <SectionError onRetry={() => void activities.refetch()} />
        ) : activities.data.groups.length === 0 ? (
          <EmptyState
            icon={<ListChecks />}
            title={NO_ACTIVITIES}
            action={<LinkButton to="/plans">Go to Plans</LinkButton>}
          />
        ) : (
          <div className={styles.groups}>
            {activities.data.groups
              .filter((group) => group.count > 0)
              .map((group) => (
                <ActivityGroup key={group.status} group={group} />
              ))}
          </div>
        )}
      </Card>
    </div>
  );
}

/** One status: a button that folds or unfolds it, then its first five activities. */
function ActivityGroup({ group }: { group: MyActivityGroup }) {
  const listId = useId();
  const [open, setOpen] = useState(group.status !== 'complete');
  const [showAll, setShowAll] = useState(false);
  const rows = showAll ? group.items : group.items.slice(0, SHOWN);

  return (
    <section className={styles.group}>
      <h3 className={styles.heading}>
        <button
          type="button"
          className={styles.toggle}
          aria-expanded={open}
          aria-controls={listId}
          onClick={() => setOpen(!open)}
        >
          <ChevronRight aria-hidden="true" className={cx(styles.chevron, open && styles.open)} />
          <ActivityStatusChip status={group.status} />{' '}
          <span className={styles.count}>{formatWhole(group.count)}</span>
        </button>
      </h3>
      <div id={listId} hidden={!open}>
        <ul className={styles.rows}>
          {rows.map((row) => (
            <ActivityRow key={row.id} row={row} />
          ))}
        </ul>
        {group.count > SHOWN && (
          <Button
            variant="tertiary"
            size="sm"
            className={styles.more}
            aria-expanded={showAll}
            onClick={() => setShowAll(!showAll)}
          >
            {showAll ? 'Show fewer' : showAllText(group.count)}
          </Button>
        )}
      </div>
    </section>
  );
}

function ActivityRow({ row }: { row: MyActivityRow }) {
  const navigate = useNavigate();
  const format = useFormatters();
  const { money } = format;

  return (
    <li className={styles.row}>
      <div className={styles.main}>
        <div className={styles.head}>
          <Link to={recordHref('activity', row.id, row.plan.id)} className={styles.name}>
            {row.name}
          </Link>
          <span className={styles.id}>{row.id}</span>
        </div>
        <p className={styles.meta}>
          {row.plan.name} › {row.campaign.name}
        </p>
        <p className={styles.meta}>{datesText(row, format)}</p>
      </div>
      <div className={styles.side}>
        <span className={styles.money}>{money(row.plannedCost)}</span>
        <CompletenessChip
          missing={row.missing.map((field) => ({ id: field.field, label: field.label }))}
          onSelectField={(field) => navigate(fieldHref('activity', row.id, field, row.plan.id))}
          itemsBelow={row.itemsBelow}
          itemsBelowText={lineItemsNeedAttentionText}
        />
      </div>
    </li>
  );
}
