import { Plus } from 'lucide-react';
import { useState } from 'react';
import { ApiError } from '@/api/errors';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { Button } from '@/ui/Button/Button';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { useToast } from '@/ui/Toast/useToast';
import { formatRate } from '@/format/decimal';
import { useFormatters } from '@/format/useFormatters';
import { useUndoableDelete } from '@/ui/useUndoableDelete';
import settingsStyles from '@/ui/settings-table.module.css';
import { inverseLine } from './currency-actions';
import { RateDialog } from './RateDialog';
import { RATE_LISTS, useDeleteRate, useRateHistory } from './rates-api';
import type { ExchangeRateRecord } from './rates-api';
import { lastRateReason, sourceOf } from './rate-words';
import styles from './rates.module.css';

export interface RateHistorySectionProps {
  /** The currency, such as "EUR"; USD has no history (it is always 1). */
  code: string;
}

/** The dialog that is open: a new rate, or one being changed. */
type OpenDialog = { kind: 'add' } | { kind: 'edit'; rate: ExchangeRateRecord };

/**
 * A currency's **Rate history** in its details (016 FR-001, FR-003): newest first, each rate with
 * its date, its source (with the file of an import) and who recorded it when, and the rate both
 * ways — three columns, so it fits the side panel without wrapping a name; "Current" on the rate
 * conversions use today and "From <date>" on future rates. **Add rate**, and per row **Edit** (the
 * rate only) and **Delete** (to the recycle bin, with Undo). The last rate dated today or earlier
 * can't be deleted, and the menu says why.
 */
export function RateHistorySection({ code }: RateHistorySectionProps) {
  const history = useRateHistory(code);
  const [open, setOpen] = useState<OpenDialog | null>(null);
  const deleteRate = useDeleteRate();
  const afterDelete = useUndoableDelete({ invalidate: RATE_LISTS });
  const { show } = useToast();
  const { date, dateTime } = useFormatters();
  const headingId = `rate-history-${code}`;

  async function remove(rate: ExchangeRateRecord) {
    try {
      afterDelete(await deleteRate.mutateAsync(rate.id));
    } catch (error) {
      const failure = error instanceof ApiError ? error : new ApiError('other');
      if (failure.kind !== 'signInRequired' && failure.kind !== 'sessionExpired') {
        show({ message: failure.message });
      }
    }
  }

  const items = history.data?.items ?? [];
  // A currency always keeps one rate dated today or earlier (FR-003).
  const inForce = items.filter((rate) => !rate.future);

  return (
    <section aria-labelledby={headingId}>
      <div className={styles.sectionHead}>
        <h3 id={headingId}>Rate history</h3>
        <Button size="sm" icon={<Plus />} onClick={() => setOpen({ kind: 'add' })}>
          Add rate
        </Button>
      </div>
      {!history.data ? (
        history.isError ? (
          <ErrorState error={history.error} onRetry={() => void history.refetch()} />
        ) : (
          <Skeleton lines={3} height="18px" />
        )
      ) : (
        <div className={settingsStyles.tableScroll}>
          <table className={settingsStyles.table}>
            <thead>
              <tr>
                <th scope="col">Date and source</th>
                <th scope="col" className={settingsStyles.number}>
                  Rate per 1 USD
                </th>
                <th scope="col">
                  <span className="visually-hidden">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {items.map((rate) => {
                const locked =
                  !rate.future && inForce.length === 1 ? lastRateReason(code) : undefined;
                return (
                  <tr key={rate.id}>
                    <th scope="row" className={styles.dateCell}>
                      {/* A future rate counts from its date; the current one is marked. */}
                      <span className={styles.dateLine}>
                        {rate.future ? `From ${date(rate.date)}` : date(rate.date)}
                        {rate.current && <StatusChip tone="positive">Current</StatusChip>}
                      </span>
                      <span className={settingsStyles.secondLine}>
                        {sourceOf(rate.source, rate.importFileName)}
                      </span>
                      <span className={settingsStyles.secondLine}>
                        {`${rate.updatedBy.name}, ${dateTime(rate.updatedAt)}`}
                      </span>
                    </th>
                    <td className={settingsStyles.number}>
                      {formatRate(rate.rate)}
                      <span className={settingsStyles.secondLine}>
                        {inverseLine(code, rate.inverseRate)}
                      </span>
                    </td>
                    <td className={settingsStyles.actionsCell}>
                      <ActionsMenu
                        label={`the rate of ${date(rate.date)}`}
                        items={[
                          { label: 'Edit', onSelect: () => setOpen({ kind: 'edit', rate }) },
                          {
                            label: 'Delete',
                            tone: 'danger',
                            separatorBefore: true,
                            hint: locked,
                            onSelect: () =>
                              locked ? show({ message: locked }) : void remove(rate),
                          },
                        ]}
                      />
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
      {open && (
        <RateDialog
          code={code}
          rate={open.kind === 'edit' ? open.rate : undefined}
          onClose={() => setOpen(null)}
        />
      )}
    </section>
  );
}
