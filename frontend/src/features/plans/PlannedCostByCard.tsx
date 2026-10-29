import { useState } from 'react';
import { Card } from '@/ui/Card/Card';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import {
  BreakdownTable,
  defaultGrouping,
  GroupingPickers,
  NO_PLANNED_COST,
  PLANNED_COST_BY,
  useAllWorkBreakdown,
  useAllWorkOptions,
} from '@/features/all-work';
import type { BreakdownChoice } from '@/features/all-work';
import type { Plan } from './api';

/** The plan's planned cost in its own currency by Spend type, or by another grouping chosen here. */
export function PlannedCostByCard({ plan }: { plan: Pick<Plan, 'id' | 'name' | 'year'> }) {
  const options = useAllWorkOptions(plan.year);
  const [chosen, setChosen] = useState<BreakdownChoice | null>(null);
  const choice =
    chosen ??
    (options.data ? { by: defaultGrouping(options.data.attributes), thenBy: null } : null);
  const query = useAllWorkBreakdown(
    choice && {
      year: plan.year,
      plan: [plan.id],
      amountsIn: 'plan',
      by: choice.by,
      ...(choice.thenBy && { thenBy: choice.thenBy }),
    },
  );
  const failed = options.isError ? options : query.isError ? query : null;

  function body() {
    if (failed) {
      return <ErrorState error={failed.error} onRetry={() => void failed.refetch()} />;
    }
    if (!query.data) {
      return <Skeleton lines={4} />;
    }
    if (query.data.groups.length === 0) {
      return <EmptyState title={NO_PLANNED_COST} />;
    }
    return <BreakdownTable breakdown={query.data} caption={`${PLANNED_COST_BY} – ${plan.name}`} />;
  }

  return (
    <Card
      title={PLANNED_COST_BY}
      actions={
        choice &&
        query.data && (
          <GroupingPickers groupings={query.data.groupings} choice={choice} onChange={setChosen} />
        )
      }
    >
      {body()}
    </Card>
  );
}
