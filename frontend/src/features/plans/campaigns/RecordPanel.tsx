import type { UseQueryResult } from '@tanstack/react-query';
import { Navigate, useLocation, useNavigate } from 'react-router';
import { useAttributeForm } from '@/features/settings';
import { allows } from '@/session/gate-action';
import { usePageTitle } from '@/ui/usePageTitle';
import { planCurrencyOf } from '../plan-currency';
import { usePlan } from './api';
import type { Campaign, Initiative } from './api';
import { PanelFailure, PanelLoading, StatePanel } from './PanelFrame';
import { targetOf } from './record-target';
import { RecordDetails } from './RecordDetails';
import { RecordEdit } from './RecordEdit';
import { RefusedPanel } from './RefusedPanel';
import { EDIT_CAMPAIGN, EDIT_INITIATIVE, LEVEL_WORD } from './texts';
import type { RecordLevel } from './texts';
import { useRecordActions } from './useRecordActions';

export interface RecordPanelProps {
  level: RecordLevel;
  mode: 'details' | 'edit';
  planId: string;
  query: UseQueryResult<Campaign | Initiative>;
}

/**
 * A campaign's or initiative's details or edit panel over the plan page; a record that has moved
 * since the address was made opens under its new plan.
 */
export function RecordPanel({ level, mode, planId, query }: RecordPanelProps) {
  const location = useLocation();
  const record = query.data;
  usePageTitle(record?.name ?? null);
  const closeTo = `/plans/${planId}`;
  const title =
    mode === 'edit' ? (level === 'campaign' ? EDIT_CAMPAIGN : EDIT_INITIATIVE) : LEVEL_WORD[level];

  if (query.isError) {
    return (
      <StatePanel title={title} closeTo={closeTo}>
        <PanelFailure error={query.error} onRetry={() => void query.refetch()} />
      </StatePanel>
    );
  }
  if (!record) {
    return (
      <StatePanel title={title} closeTo={closeTo}>
        <PanelLoading />
      </StatePanel>
    );
  }
  if (record.planId !== planId) {
    const prefix = `/plans/${planId}/`;
    const to = `/plans/${record.planId}/${location.pathname.slice(prefix.length)}${location.search}`;
    return <Navigate to={to} replace />;
  }
  if (mode === 'details') {
    return <DetailsPanel level={level} record={record} planId={planId} />;
  }
  return <EditWhenReady level={level} record={record} planId={planId} title={title} />;
}

function DetailsPanel({
  level,
  record,
  planId,
}: {
  level: RecordLevel;
  record: Campaign | Initiative;
  planId: string;
}) {
  const navigate = useNavigate();
  const actions = useRecordActions({ onDeleted: () => void navigate(`/plans/${planId}`) });
  const target = targetOf(record);
  return (
    <RecordDetails
      level={level}
      record={record}
      planId={planId}
      recordItems={actions.copyAndMoveItems(target)}
      moreItems={actions.deleteItem(target)}
      dialogs={actions.dialogs}
    />
  );
}

function EditWhenReady({
  level,
  record,
  planId,
  title,
}: {
  level: RecordLevel;
  record: Campaign | Initiative;
  planId: string;
  title: string;
}) {
  const attributeForm = useAttributeForm(level);
  const plan = usePlan(planId);
  const closeTo = `/plans/${planId}`;
  const failed = [attributeForm, plan].find((part) => part.isError);
  if (!allows(record.access?.edit)) {
    return <RefusedPanel title={title} closeTo={closeTo} availability={record.access?.edit} />;
  }
  if (failed) {
    return (
      <StatePanel title={title} closeTo={closeTo}>
        <PanelFailure error={failed.error} onRetry={() => void failed.refetch()} />
      </StatePanel>
    );
  }
  if (!attributeForm.data || !plan.data) {
    return (
      <StatePanel title={title} closeTo={closeTo}>
        <PanelLoading />
      </StatePanel>
    );
  }
  return (
    <RecordEdit
      level={level}
      record={record}
      planId={planId}
      attributeForm={attributeForm.data}
      businessUnit={plan.data.businessUnit.map((option) => option.id)}
      currency={planCurrencyOf(plan.data)}
    />
  );
}
