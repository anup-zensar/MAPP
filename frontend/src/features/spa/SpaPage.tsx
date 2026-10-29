import { useState } from 'react';
import { useParams, useSearchParams } from 'react-router';
import { ApiError } from '@/api/errors';
import { BcpirTag, PlanStatusChip } from '@/features/plans';
import { useFormatters } from '@/format/useFormatters';
import { LinkButton } from '@/ui/Button/Button';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { NoAccess } from '@/ui/NoAccess/NoAccess';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { usePageTitle } from '@/ui/usePageTitle';
import { useSpaView } from './api';
import type { SpaView } from './api';
import type { TraceRequest } from './FigureButton';
import { readSpaState, toSpaParams } from './schemas';
import type { SpaUrlState } from './schemas';
import { SpaContent } from './SpaContent';
import { NOT_FOUND, TITLE } from './spa-text';
import styles from './Spa.module.css';

/** The SP&A view of a plan or of one of its campaigns (013 US1, US3), at /plans/:planId/spa. */
export function SpaPage() {
  const { planId = '' } = useParams();
  const [params, setParams] = useSearchParams();
  const state = readSpaState(params);
  const { displayCurrency } = useFormatters();
  const query = useSpaView(
    { planId, campaign: state.campaign, dimension: state.dimension },
    displayCurrency,
  );
  const [trace, setTrace] = useState<TraceRequest | null>(null);
  usePageTitle(query.data ? `${TITLE} · ${query.data.plan.name}` : TITLE);

  function change(next: Partial<SpaUrlState>) {
    setParams(toSpaParams({ ...state, ...next }));
  }

  if (query.error instanceof ApiError && query.error.kind === 'forbidden') {
    return <NoAccess message={query.error.message} />;
  }
  if (query.error instanceof ApiError && query.error.kind === 'notFound') {
    return <SpaNotFound />;
  }
  return (
    <div className={styles.page}>
      <SpaHeader view={query.data} />
      {query.isError && <ErrorState error={query.error} onRetry={() => void query.refetch()} />}
      {!query.data && !query.isError && <Skeleton lines={8} />}
      {query.data && (
        <SpaContent
          view={query.data}
          state={state}
          onChange={change}
          trace={trace}
          onTrace={setTrace}
        />
      )}
    </div>
  );
}

function SpaHeader({ view }: { view: SpaView | undefined }) {
  const plan = view?.plan;
  return (
    <PageHeader
      breadcrumb={[
        { label: 'Plans', to: '/plans' },
        ...(plan ? [{ label: plan.name, to: `/plans/${encodeURIComponent(plan.id)}` }] : []),
        { label: TITLE },
      ]}
      title={TITLE}
      meta={
        plan && (
          <div className={styles.meta} role="group" aria-label="Plan facts">
            <PlanStatusChip status={plan.status} />
            <BcpirTag plan={plan} />
            <span>{plan.year}</span>
          </div>
        )
      }
      actions={
        plan && (
          <LinkButton
            to={`/all-work?${new URLSearchParams({ plan: plan.id, year: String(plan.year) })}`}
          >
            Open in All work
          </LinkButton>
        )
      }
    />
  );
}

function SpaNotFound() {
  return (
    <PageHeader
      breadcrumb={[{ label: 'Plans', to: '/plans' }, { label: TITLE }]}
      title={TITLE}
      description={NOT_FOUND}
      actions={
        <LinkButton to="/plans" variant="primary">
          Go to Plans
        </LinkButton>
      }
    />
  );
}
