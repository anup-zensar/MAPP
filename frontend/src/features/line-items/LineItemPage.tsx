import { Pencil } from 'lucide-react';
import { useNavigate, useParams } from 'react-router';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { LinkButton } from '@/ui/Button/Button';
import { CompletenessChip } from '@/ui/CompletenessChip/CompletenessChip';
import { LevelTag } from '@/ui/LevelTag/LevelTag';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { HistorySection, showHistory } from '@/features/activity-log';
import { allows, gateItem } from '@/session/gate-action';
import { fieldHref, recordHref } from '@/ui/record-links';
import { usePageTitle } from '@/ui/usePageTitle';
import { LineItemFinancials } from './FinancialsSection';
import styles from './LineItemPage.module.css';
import { activityBreadcrumb, activityLineItemsHref } from './line-item-links';
import { LineItemDetailsCard } from './LineItemDetailsCard';
import { LineItemFigures } from './LineItemFigures';
import { LineItemFinanceCard } from './LineItemFinanceCard';
import { LineItemLoadFailed, LineItemLoading } from './LineItemPageStates';
import { LineItemSpreadCard } from './LineItemSpreadCard';
import { useLineItem } from './api';
import { DELETE, DUPLICATE, EDIT, VIEW_HISTORY } from './line-item-text';
import { SwbsChip } from './SwbsChip';
import { useLineItemActions } from './useLineItemActions';

/**
 * A line item's page (US4, FR-026), at /line-items/:lineItemId, on the record template of design
 * system §2.4 B and as wide as its activity's page (UI audit LI-01): the header with the level tag,
 * completeness and SWBS chips, ID and owner in the meta row, ⋯ (Duplicate, View history, Delete)
 * and Edit; the summary cards (planned cost, spread, the activity's unassigned planned cost); the
 * Details and Finance coding cards, each value in the one key–value layout ("—" when empty); GSAP's
 * Financials (feature 015); the spread; then its History (feature 011).
 */
export function LineItemPage() {
  const { lineItemId = '' } = useParams();
  const navigate = useNavigate();
  const details = useLineItem(lineItemId);
  const actions = useLineItemActions((activity) => navigate(activityLineItemsHref(activity)));
  usePageTitle(details.data?.lineItem.name ?? 'Line item');

  if (details.isError) {
    return (
      <LineItemLoadFailed
        title="Line item"
        error={details.error}
        onRetry={() => void details.refetch()}
      />
    );
  }
  if (!details.data) {
    return <LineItemLoading title="Line item" />;
  }

  const { lineItem, activity, balance, access } = details.data;
  const href = recordHref('lineItem', lineItem.id);

  return (
    <>
      <PageHeader
        breadcrumb={[...activityBreadcrumb(activity), { label: lineItem.name }]}
        title={lineItem.name}
        meta={
          <>
            <LevelTag level="lineItem" />
            <CompletenessChip
              missing={lineItem.missing.map((field) => ({ id: field.field, label: field.label }))}
              onSelectField={(field) => void navigate(fieldHref('lineItem', lineItem.id, field))}
            />
            <SwbsChip status={lineItem.swbsStatus} />
            <span className={styles.fact}>{lineItem.id}</span>
            {lineItem.owner ? (
              <PersonLabel person={lineItem.owner} />
            ) : (
              <span className={styles.fact}>No owner yet</span>
            )}
          </>
        }
        actions={
          <>
            {/* ⋯ first, the primary last (design system §2.3, UI audit LI-02). */}
            <ActionsMenu
              label={lineItem.name}
              items={[
                ...gateItem(
                  { label: DUPLICATE, onSelect: () => actions.duplicate(lineItem, activity) },
                  access?.duplicate,
                ),
                // Brings the History section below into view and focuses it (feature 011).
                { label: VIEW_HISTORY, onSelect: () => showHistory() },
                ...gateItem(
                  {
                    label: DELETE,
                    tone: 'danger',
                    separatorBefore: true,
                    onSelect: () => actions.remove(lineItem, activity),
                  },
                  access?.delete,
                ),
              ]}
            />
            {/* 014 FR-011: Edit only for a person who may edit the line item. */}
            {allows(access?.edit) && (
              <LinkButton to={`${href}/edit`} variant="primary" icon={<Pencil />}>
                {EDIT}
              </LinkButton>
            )}
          </>
        }
      />
      <div className={styles.page}>
        <LineItemFigures lineItem={lineItem} balance={balance} />
        <div className={styles.columns}>
          <LineItemDetailsCard lineItem={lineItem} />
          <LineItemFinanceCard lineItem={lineItem} activity={activity} />
        </div>
        {/* GSAP's money for this line item, open (015 FR-017). */}
        <LineItemFinancials lineItemId={lineItem.id} />
        <LineItemSpreadCard lineItem={lineItem} />
        <HistorySection
          recordType="lineItem"
          recordId={lineItem.id}
          recordName={lineItem.name}
          level="lineItem"
        />
      </div>
      {actions.dialogs}
    </>
  );
}
