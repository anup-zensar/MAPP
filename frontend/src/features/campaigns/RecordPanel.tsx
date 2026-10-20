import type { UseQueryResult } from '@tanstack/react-query';
import { Navigate, useLocation, useNavigate } from 'react-router';
import { useAttributeForm } from '@/api/settingsQueries';
import { usePageTitle } from '@/lib/usePageTitle';
import { planCurrencyOf } from '@/features/plans/planCurrency';
import { usePlan } from './campaignApi';
import type { Campaign, Initiative } from './campaignApi';
import { PanelFailure, PanelLoading, StatePanel } from './PanelFrame';
import { targetOf } from './campaignRows';
import { RecordDetails } from './RecordDetails';
import { RecordEdit } from './RecordEdit';
import { EDIT_CAMPAIGN, EDIT_INITIATIVE, LEVEL_WORD } from './texts';
import type { RecordLevel } from './texts';
import { useRecordActions } from './useRecordActions';

export interface RecordPanelProps {
  level: RecordLevel;
  mode: 'details' | 'edit';
  /** The plan in the address. */
  planId: string;
  query: UseQueryResult<Campaign | Initiative>;
}

/**
 * A campaign's or initiative's own address (FR-017): its details or edit panel over the plan
 * page, a loading or failure state, "no longer exists" with **Back to the plan**, and the same
 * panel under the right plan when the record has moved since the address was made.
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

/**
 * The details with **Copy**, **Move to…** and **Delete** under More; after a delete the panel
 * closes (FR-029), after a move or copy the plan page opens with the record highlighted.
 */
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

/** The edit form, once the Shell attributes and the plan's business unit are known. */
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
