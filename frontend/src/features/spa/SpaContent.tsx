import { useId } from 'react';
import { useFormatters } from '@/format/useFormatters';
import { LinkButton } from '@/ui/Button/Button';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import type { SpaView } from './api';
import type { TraceRequest } from './FigureButton';
import type { SpaUrlState } from './schemas';
import { SpaActivities } from './SpaActivities';
import { SpaDimension } from './SpaDimension';
import { SpaHeadline } from './SpaHeadline';
import { SpaBudget, SpaProcurement } from './SpaPanels';
import { SpaPeriods } from './SpaPeriods';
import { FINANCE_NONE, WHOLE_PLAN, financeAsOf } from './spa-text';
import { TracePanel } from './TracePanel';
import styles from './Spa.module.css';

interface SpaContentProps {
  view: SpaView;
  state: SpaUrlState;
  onChange: (next: Partial<SpaUrlState>) => void;
  trace: TraceRequest | null;
  onTrace: (request: TraceRequest | null) => void;
}

/** The view's figures, cards and table below the header. */
export function SpaContent({ view, state, onChange, trace, onTrace }: SpaContentProps) {
  return (
    <>
      <ScopeBar view={view} onCampaign={(campaign) => onChange({ campaign })} />
      {view.notes.map((note) => (
        <p key={note} className={styles.note}>
          {note}
        </p>
      ))}
      {view.emptyMessage ? (
        <EmptyState
          title={view.emptyMessage}
          action={
            <LinkButton to={`/plans/${encodeURIComponent(view.plan.id)}`}>
              Go to the plan tree
            </LinkButton>
          }
        />
      ) : (
        <>
          <SpaHeadline view={view} onTrace={onTrace} />
          <div className={styles.columns}>
            <SpaPeriods
              view={view}
              period={state.period}
              onPeriod={(period) => onChange({ period })}
              onTrace={onTrace}
            />
            <SpaDimension
              view={view}
              onDimension={(dimension) => onChange({ dimension })}
              onTrace={onTrace}
            />
          </div>
          <SpaActivities view={view} sort={state.sort} onSort={(sort) => onChange({ sort })} />
          <div className={styles.columns}>
            <SpaBudget view={view} />
            <SpaProcurement view={view} />
          </div>
        </>
      )}
      <TracePanel planId={view.plan.id} request={trace} onClose={() => onTrace(null)} />
    </>
  );
}

/** The scope picker (013 FR-005) and how recent the finance data is (FR-009). */
function ScopeBar({
  view,
  onCampaign,
}: {
  view: SpaView;
  onCampaign: (id: string | undefined) => void;
}) {
  const id = useId();
  const { dateTime } = useFormatters();
  const scope = view.scope.type === 'campaign' ? view.scope.id : '';
  return (
    <div className={styles.scopeBar}>
      <label className={styles.picker} htmlFor={id}>
        Scope
        <select
          id={id}
          className={styles.select}
          value={scope}
          onChange={(event) => onCampaign(event.target.value || undefined)}
        >
          <option value="">{WHOLE_PLAN}</option>
          {view.campaigns.map((campaign) => (
            <option key={campaign.id} value={campaign.id}>
              {campaign.name}
            </option>
          ))}
        </select>
      </label>
      <span className={styles.muted}>
        {view.financeNote ??
          (view.financeDataAsOf ? financeAsOf(dateTime(view.financeDataAsOf)) : FINANCE_NONE)}
      </span>
    </div>
  );
}
