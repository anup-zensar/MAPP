import type { UseQueryResult } from '@tanstack/react-query';
import { CircleAlert } from 'lucide-react';
import { useId } from 'react';
import type { ReactNode } from 'react';
import { ApiError } from '@/api/errors';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { LockedNote } from '@/ui/LockedNote/LockedNote';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { useFormatters } from '@/format/useFormatters';
import type { ActivityDestination } from './activitiesApi';
import { datesText, THIS_INITIATIVE } from './activityText';
import styles from './DestinationList.module.css';

type DestinationPage = { items: ActivityDestination[]; total: number };

export interface DestinationsStateProps {
  query: UseQueryResult<DestinationPage>;
  /** The dialog's content once the initiatives are there. */
  children: (items: ActivityDestination[]) => ReactNode;
}

/**
 * The move and copy dialogs' states: grey lines while the initiatives load, the error with **Try
 * again**, or — when the activity is gone — the API's message alone (Cancel closes the dialog).
 */
export function DestinationsState({ query, children }: DestinationsStateProps) {
  if (query.data) {
    return children(query.data.items);
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

export interface DestinationListProps {
  /** Names the whole list; `legendHidden` keeps it for screen readers only. */
  legend: string;
  legendHidden?: boolean;
  destinations: ActivityDestination[];
  /** Which reason makes an initiative unavailable: the move's or the copy's. */
  purpose: 'move' | 'copy';
  /** The chosen initiative's id; "" for none. */
  value: string;
  onChange: (initiativeId: string) => void;
  error?: string;
}

/**
 * The initiatives of the plan as radios, grouped by campaign (FR-029, FR-030): each with its
 * dates, the activity's own one marked "(this initiative)", and one that can't take the activity
 * listed with the reason and what to do.
 */
export function DestinationList({
  legend,
  legendHidden = false,
  destinations,
  purpose,
  value,
  onChange,
  error,
}: DestinationListProps) {
  const groupId = useId();
  const format = useFormatters();
  const errorId = `${groupId}-error`;

  return (
    <fieldset className={styles.group} aria-describedby={error ? errorId : undefined}>
      <legend className={legendHidden ? 'visually-hidden' : styles.legend}>{legend}</legend>
      {byCampaign(destinations).map(({ campaign, items }) => (
        <fieldset key={campaign.id} className={styles.campaign}>
          <legend className={styles.campaignName}>{campaign.name}</legend>
          <ul className={styles.list}>
            {items.map((item) => {
              const inputId = `${groupId}-${item.initiative.id}`;
              const datesId = `${inputId}-dates`;
              const reasonId = `${inputId}-reason`;
              const problem = purpose === 'move' ? item.moveProblem : item.copyProblem;
              return (
                <li key={item.initiative.id} className={styles.option}>
                  <input
                    id={inputId}
                    type="radio"
                    name={groupId}
                    value={item.initiative.id}
                    checked={value === item.initiative.id}
                    disabled={problem !== null}
                    aria-describedby={problem === null ? datesId : `${datesId} ${reasonId}`}
                    onChange={() => onChange(item.initiative.id)}
                  />
                  <div className={styles.text}>
                    <label htmlFor={inputId} className={styles.name}>
                      {item.initiative.name}
                      {item.current && (
                        <>
                          {' '}
                          <span className={styles.current}>{THIS_INITIATIVE}</span>
                        </>
                      )}
                    </label>
                    <span id={datesId} className={styles.dates}>
                      {datesText(item.startDate, item.endDate, false, format)}
                    </span>
                    {problem !== null && <LockedNote id={reasonId}>{problem}</LockedNote>}
                  </div>
                </li>
              );
            })}
          </ul>
        </fieldset>
      ))}
      {error && (
        <p id={errorId} className={styles.error}>
          <CircleAlert aria-hidden="true" />
          {error}
        </p>
      )}
    </fieldset>
  );
}

/** The initiatives grouped by campaign, in the API's order (campaigns, then their initiatives). */
function byCampaign(destinations: ActivityDestination[]) {
  const groups: { campaign: ActivityDestination['campaign']; items: ActivityDestination[] }[] = [];
  for (const item of destinations) {
    const last = groups.at(-1);
    if (last && last.campaign.id === item.campaign.id) {
      last.items.push(item);
    } else {
      groups.push({ campaign: item.campaign, items: [item] });
    }
  }
  return groups;
}
