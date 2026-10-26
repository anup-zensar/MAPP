import { ClipboardList, SearchX } from 'lucide-react';
import type { ReactNode } from 'react';
import { ApiError } from '@/api/errors';
import { Button, LinkButton } from '@/ui/Button/Button';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import styles from './AllWorkPage.module.css';
import type { AllWorkResult, ViewState } from './all-work-types';
import { GridSkeleton } from './GridSkeleton';

export interface ViewContentProps {
  state: ViewState;
  result: AllWorkResult | undefined;
  error: unknown;
  /** Whether the year has plans at all (the filter bar's choices). */
  hasPlans: boolean;
  onRetry: () => void;
  onClearFilters: () => void;
  /** The view itself, shown when there are rows. */
  children: ReactNode;
}

/** The states of a view (plan "States"): loading, error, no plans, nothing matches, the rows. */
export function ViewContent({
  state,
  result,
  error,
  hasPlans,
  onRetry,
  onClearFilters,
  children,
}: ViewContentProps) {
  if (!result) {
    return error && !(error instanceof ApiError && error.kind === 'validation') ? (
      <div className={styles.state}>
        <ErrorState error={error} onRetry={onRetry} />
      </div>
    ) : (
      <GridSkeleton />
    );
  }
  if (result.rows.length === 0) {
    return (
      <div className={styles.state}>
        {!hasPlans ? (
          <EmptyState
            icon={<ClipboardList />}
            title={`No plans in ${state.year} yet`}
            text={`Plans of ${state.year} appear here once they're created.`}
            action={<LinkButton to="/plans">Go to Plans</LinkButton>}
          />
        ) : (
          <EmptyState
            icon={<SearchX />}
            title="Nothing matches these filters"
            text="Try removing a filter or changing the search."
            action={<Button onClick={onClearFilters}>Clear filters</Button>}
          />
        )}
      </div>
    );
  }
  return <>{children}</>;
}
