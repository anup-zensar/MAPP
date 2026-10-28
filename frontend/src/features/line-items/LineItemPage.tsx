import { Pencil } from 'lucide-react';
import { useNavigate, useParams } from 'react-router';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { LinkButton } from '@/ui/Button/Button';
import { CompletenessChip } from '@/ui/CompletenessChip/CompletenessChip';
import { LevelTag } from '@/ui/LevelTag/LevelTag';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { HistorySection, showHistory } from '@/features/activity-log';
import { LineItemPurchaseRequests } from '@/features/purchase-requests';
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
import { SwbsCard } from './SwbsCard';
import { SwbsChip } from './SwbsChip';
import { useLineItemActions } from './useLineItemActions';

/**
 * A line item's page (US4, FR-026) on the record template of design system §2.4 B: header, summary
 * cards, Details and Finance coding, the SWBS request (030), GSAP's Financials (015), its purchase
 * requests (032), the spread and its History (011).
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
            <ActionsMenu
              label={lineItem.name}
              items={[
                ...gateItem(
                  { label: DUPLICATE, onSelect: () => actions.duplicate(lineItem, activity) },
                  access?.duplicate,
                ),
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
        <SwbsCard lineItem={lineItem} />
        <LineItemFinancials lineItemId={lineItem.id} />
        <LineItemPurchaseRequests lineItemId={lineItem.id} />
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
