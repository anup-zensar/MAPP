import { ChartPie } from 'lucide-react';
import { ApiError } from '@/api/errors';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import pageStyles from './AllWorkPage.module.css';
import type { BreakdownChoice, FilterableAttribute, ViewState } from './all-work-types';
import { useAllWorkBreakdown } from './breakdown-api';
import { breakdownChoice, breakdownQuery, gridOfGroup } from './breakdown-state';
import type { GroupRef } from './breakdown-state';
import { BREAKDOWN, NO_PLANNED_COST, NO_PLANNED_COST_TEXT } from './breakdown-text';
import { BreakdownTable } from './BreakdownTable';
import styles from './BreakdownView.module.css';
import { GridSkeleton } from './GridSkeleton';
import { GroupingPickers } from './GroupingPickers';

export interface BreakdownViewProps {
  state: ViewState;
  /** The filter bar's Shell attributes; undefined while they load (the first grouping needs them). */
  attributes: FilterableAttribute[] | undefined;
  onChange: (next: ViewState) => void;
}

/** The filtered planned cost by one grouping and then by a second; a group's name opens the Grid. */
export function BreakdownView({ state, attributes, onChange }: BreakdownViewProps) {
  const choice = breakdownChoice(state, attributes);
  const known = state.breakdown !== undefined && state.breakdown !== null;
  const query = useAllWorkBreakdown(known || attributes ? breakdownQuery(state, choice) : null);
  const choose = (next: BreakdownChoice) => onChange({ ...state, breakdown: next });
  const answered = query.data && {
    by: query.data.by.key,
    thenBy: query.data.thenBy?.key ?? null,
  };
  const openGroup = (groups: GroupRef[]) => {
    const next = gridOfGroup(state, answered ?? choice, groups);
    return next ? () => onChange(next) : null;
  };

  return (
    <div className={styles.view}>
      {query.data && (
        <div className={styles.toolbar}>
          <GroupingPickers groupings={query.data.groupings} choice={choice} onChange={choose} />
        </div>
      )}
      <Content
        query={query}
        onReset={() => onChange({ ...state, breakdown: null })}
        openGroup={openGroup}
      />
    </div>
  );
}

interface ContentProps {
  query: ReturnType<typeof useAllWorkBreakdown>;
  onReset: () => void;
  openGroup: (groups: GroupRef[]) => (() => void) | null;
}

function Content({ query, onReset, openGroup }: ContentProps) {
  if (query.isError && !query.isFetching) {
    // A grouping the API refused (one of a saved view that no longer exists) goes back to the first.
    const refused = query.error instanceof ApiError && query.error.kind === 'validation';
    return (
      <div className={pageStyles.state}>
        <ErrorState error={query.error} onRetry={refused ? onReset : () => void query.refetch()} />
      </div>
    );
  }
  if (!query.data) {
    return <GridSkeleton />;
  }
  if (query.data.groups.length === 0) {
    return (
      <div className={pageStyles.state}>
        <EmptyState icon={<ChartPie />} title={NO_PLANNED_COST} text={NO_PLANNED_COST_TEXT} />
      </div>
    );
  }
  return <BreakdownTable breakdown={query.data} caption={BREAKDOWN} openGroup={openGroup} />;
}
