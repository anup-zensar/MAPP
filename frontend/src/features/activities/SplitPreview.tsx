import { ActivityStatusChip } from '@/ui/ActivityStatusChip/ActivityStatusChip';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { moneyText } from '@/features/line-items';
import type { ActivitySplitSide } from './split-api';
import styles from './SplitActivityDialog.module.css';

/** Both activities as they will be (FR-027), side by side from 600 px. */
export function SplitPreview({
  original,
  created,
  waiting,
}: {
  original: ActivitySplitSide | null;
  created: ActivitySplitSide | null;
  waiting: 'nothing' | 'loading' | null;
}) {
  return (
    <section aria-label="After the split" className={styles.preview}>
      <h3 className={styles.legend}>After the split</h3>
      {waiting === 'nothing' ? (
        <p className={styles.hint}>Choose line items or an amount to see both activities.</p>
      ) : waiting === 'loading' || !original || !created ? (
        <Skeleton lines={3} />
      ) : (
        <div className={styles.sides} aria-live="polite">
          <Side side={original} />
          <Side side={created} isNew />
        </div>
      )}
    </section>
  );
}

function Side({ side, isNew = false }: { side: ActivitySplitSide; isNew?: boolean }) {
  const negative = side.unassignedState === 'exceeded';
  return (
    <div className={styles.side}>
      <p className={styles.sideName}>
        {isNew ? `New: ${side.name}` : side.name}
        <span className={styles.sideInitiative}>{side.initiativeName}</span>
      </p>
      <dl className={styles.facts}>
        <div>
          <dt>Planned cost</dt>
          <dd>{side.plannedCost ? moneyText(side.plannedCost) : '—'}</dd>
        </div>
        <div>
          <dt>Unassigned planned cost</dt>
          <dd className={negative ? styles.negative : undefined}>
            {side.unassignedPlannedCost ? moneyText(side.unassignedPlannedCost) : '—'}
            {negative && ' — line items exceed the planned cost'}
          </dd>
        </div>
        <div>
          <dt>Line items</dt>
          <dd>{side.lineItemNames.length > 0 ? side.lineItemNames.join(', ') : 'None'}</dd>
        </div>
        <div>
          <dt>Status</dt>
          <dd>
            <ActivityStatusChip status={side.status} />
          </dd>
        </div>
      </dl>
    </div>
  );
}
