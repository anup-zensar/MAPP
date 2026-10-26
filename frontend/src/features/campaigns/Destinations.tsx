import type { UseQueryResult } from '@tanstack/react-query';
import { CircleAlert, TriangleAlert } from 'lucide-react';
import { useId } from 'react';
import type { ReactNode } from 'react';
import { ApiError } from '@/api/errors';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { LockedNote } from '@/ui/LockedNote/LockedNote';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { STATUS } from '@/features/plans/planText';
import { useFormatters } from '@/format/useFormatters';
import type { Destination, DestinationList, MoveConsequences } from './campaignApi';
import styles from './Destinations.module.css';
import {
  CURRENT,
  goalsRecalculated,
  hiddenThere,
  otherModelOrUnit,
  overAllocateWarning,
  shownThere,
  typesNotAllowedText,
} from './texts';

export interface DestinationsLoadProps {
  query: UseQueryResult<DestinationList>;
  /** The dialog's content once the destinations are there. */
  children: (list: DestinationList) => ReactNode;
}

/**
 * The move and copy dialogs' states (plan "States"): grey lines while the destinations load, the
 * error with **Try again**, or — when the record is gone — the API's message alone (Cancel closes).
 */
export function DestinationsLoad({ query, children }: DestinationsLoadProps) {
  if (query.data) {
    return children(query.data);
  }
  if (query.isError) {
    return query.error instanceof ApiError && query.error.kind === 'notFound' ? (
      <p role="alert" className={styles.error}>
        <CircleAlert aria-hidden="true" />
        {query.error.message}
      </p>
    ) : (
      <ErrorState error={query.error} onRetry={() => void query.refetch()} />
    );
  }
  return <Skeleton lines={3} />;
}

export interface DestinationChoiceProps {
  /** Names the group ("Copy into"); `legendHidden` keeps it for screen readers only. */
  legend: string;
  legendHidden?: boolean;
  items: Destination[];
  /** The chosen destination's id; "" for none. */
  value: string;
  onChange: (id: string) => void;
  error?: string;
}

/**
 * The destinations as radios (FR-023, FR-026): each plan with its status, or each campaign with
 * its dates, "Current" on the record's own parent; one that can't be chosen stays listed with its
 * reason and the fix under it.
 */
export function DestinationChoice({
  legend,
  legendHidden = false,
  items,
  value,
  onChange,
  error,
}: DestinationChoiceProps) {
  const groupId = useId();
  const { dateRange } = useFormatters();
  const errorId = `${groupId}-error`;

  return (
    <fieldset className={styles.group} aria-describedby={error ? errorId : undefined}>
      <legend className={legendHidden ? 'visually-hidden' : styles.legend}>{legend}</legend>
      <ul className={styles.list}>
        {items.map((item, index) => {
          const inputId = `${groupId}-${index}`;
          const detailsId = `${inputId}-details`;
          const reasonId = `${inputId}-reason`;
          const status = item.planStatus ? STATUS[item.planStatus] : null;
          return (
            <li key={item.id} className={styles.option}>
              <input
                id={inputId}
                type="radio"
                name={groupId}
                value={item.id}
                checked={value === item.id}
                disabled={!item.available}
                aria-describedby={item.available ? detailsId : `${detailsId} ${reasonId}`}
                onChange={() => onChange(item.id)}
              />
              <div className={styles.text}>
                <label htmlFor={inputId} className={styles.name}>
                  {item.name}
                </label>
                <span id={detailsId} className={styles.details}>
                  {item.current && (
                    <>
                      <StatusChip tone="neutral">{CURRENT}</StatusChip>
                      {/* Screen readers hear "Current Published", not "CurrentPublished". */}{' '}
                    </>
                  )}
                  {status ? (
                    <StatusChip tone={status.tone}>{status.word}</StatusChip>
                  ) : (
                    <span>{dateRange(item.startDate, item.endDate)}</span>
                  )}
                </span>
                {!item.available && <LockedNote id={reasonId}>{item.unavailableReason}</LockedNote>}
              </div>
            </li>
          );
        })}
      </ul>
      {error && (
        <p id={errorId} className={styles.error}>
          <CircleAlert aria-hidden="true" />
          {error}
        </p>
      )}
    </fieldset>
  );
}

/**
 * What choosing a destination means (FR-024, FR-026): the budget it would over-allocate and, for a
 * campaign going to a plan with another model or business unit, the consequences. Announced when
 * the choice changes.
 */
export function DestinationNotes({ destination }: { destination: Destination | undefined }) {
  const { money } = useFormatters();
  return (
    <div aria-live="polite" className={styles.notes}>
      {destination?.overAllocatedBy && (
        <p className={styles.warning}>
          <TriangleAlert aria-hidden="true" />
          {overAllocateWarning(destination.name, money(destination.overAllocatedBy))}
        </p>
      )}
      {destination?.consequences && (
        <Consequences plan={destination.name} consequences={destination.consequences} />
      )}
    </div>
  );
}

function Consequences({ plan, consequences }: { plan: string; consequences: MoveConsequences }) {
  const { modelName, modelChanges, typesNotAllowed, groupsHidden, groupsShown } = consequences;
  return (
    <div className={styles.consequences}>
      <p className={styles.consequencesTitle}>{otherModelOrUnit(plan)}</p>
      <ul className={styles.consequenceList}>
        {typesNotAllowed.length > 0 && (
          <li>
            {typesNotAllowedText(
              modelName,
              typesNotAllowed.map((activity) => `${activity.name} (${activity.id})`).join(', '),
            )}
          </li>
        )}
        {modelChanges && <li>{goalsRecalculated(modelName)}</li>}
        {groupsHidden.length > 0 && <li>{hiddenThere(groupsHidden.join(', '))}</li>}
        {groupsShown.length > 0 && <li>{shownThere(groupsShown.join(', '))}</li>}
      </ul>
    </div>
  );
}
