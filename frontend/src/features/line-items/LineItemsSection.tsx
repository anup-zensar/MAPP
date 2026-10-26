import { CircleAlert, Plus, ReceiptText } from 'lucide-react';
import { useEffect } from 'react';
import { Link, useLocation, useNavigate } from 'react-router';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { LinkButton } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { CompletenessChip } from '@/ui/CompletenessChip/CompletenessChip';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import type { components } from '@/api/generated/schema';
import { allows, gateItem } from '@/session/gate-action';
import { cx } from '@/ui/cx';
import { fieldHref, recordHref } from '@/ui/record-links';
import { useFinancialRecord } from './gsap-financials';
import type { FinancialFiguresData } from './gsap-financials';
import { moneyText } from './line-item-math';
import { useLineItemSection } from './api';
import type { LineItem } from './api';
import styles from './LineItemsSection.module.css';
import {
  ADD_LINE_ITEM,
  DELETE,
  DUPLICATE,
  EDIT,
  EMPTY_SECTION,
  SECTION_SUBTITLE,
  SECTION_TITLE,
  planCurrencyColumn,
  SPREAD_LABELS,
  usdColumn,
  VIEW,
} from './line-item-text';
import { SpreadGrid } from './SpreadGrid';
import { SwbsChip } from './SwbsChip';
import { useLineItemActions } from './useLineItemActions';
import { UnassignedBalanceRow } from './UnassignedBalanceRow';

type LineItemAccess = components['schemas']['LineItemAccess'];

export interface LineItemsSectionProps {
  activityId: string;
}

/**
 * The Line items section of feature 006's activity page (US1, FR-001 – FR-005): one row per line
 * item with its amounts, spread, completeness and SWBS request, the unassigned planned cost, and
 * the spread grid. A row given in the address's state (highlightLineItemId, after a create) is
 * scrolled into view and highlighted.
 */
export function LineItemsSection({ activityId }: LineItemsSectionProps) {
  const section = useLineItemSection(activityId);
  // The GSAP figures of each line item, from the activity's record view: the same query as the
  // Financials card, so the page asks once (015 FR-016, plan "Query keys").
  const financials = useFinancialRecord(activityId);
  const actions = useLineItemActions();
  const location = useLocation();
  const navigate = useNavigate();
  const highlight = (location.state as { highlightLineItemId?: string } | null)
    ?.highlightLineItemId;

  useEffect(() => {
    if (highlight && section.data) {
      document
        .getElementById(rowId(highlight))
        ?.scrollIntoView({ block: 'center', behavior: 'smooth' });
    }
  }, [highlight, section.data]);

  const access = section.data?.access;
  // 014 FR-011: Add line item only for a person who may add line items here.
  const addButton = allows(access?.add) ? (
    <LinkButton to={`/activities/${encodeURIComponent(activityId)}/line-items/new`} icon={<Plus />}>
      {ADD_LINE_ITEM}
    </LinkButton>
  ) : undefined;

  if (section.isPending) {
    return (
      <Card title={SECTION_TITLE} subtitle={SECTION_SUBTITLE}>
        <Skeleton lines={3} />
      </Card>
    );
  }
  if (section.isError) {
    return (
      <Card title={SECTION_TITLE} subtitle={SECTION_SUBTITLE}>
        <ErrorState error={section.error} onRetry={() => void section.refetch()} />
      </Card>
    );
  }

  const { items, balance, grid, activity } = section.data;
  // The reader's display currency: the API answers the USD amounts in it (FR-005, D-213).
  const displayCode = balance.assigned.currency;
  // The plan currency (016 FR-015), when the API sends the plan amounts and it is not the reader's.
  const planCode = items.find((item) => item.plannedCostPlan)?.plannedCostPlan?.currency;
  const planColumn = planCode !== undefined && planCode !== displayCode ? planCode : null;
  const figuresOf = (id: string): GsapCell =>
    financials.data
      ? (financials.data.breakdown.find((row) => row.record.id === id)?.figures ?? 'none')
      : financials.isError
        ? 'none'
        : 'loading';

  return (
    <Card
      title={SECTION_TITLE}
      subtitle={SECTION_SUBTITLE}
      actions={items.length > 0 ? addButton : undefined}
    >
      <div className={styles.stack}>
        {items.length === 0 ? (
          <EmptyState icon={<ReceiptText />} title={EMPTY_SECTION} action={addButton} />
        ) : (
          // The rows scroll sideways inside the card on narrow screens, never the page.
          <div
            className={styles.scroll}
            role="region"
            aria-label="Line items table, scrolls sideways"
            tabIndex={0}
          >
            <table className={styles.table}>
              <caption className="visually-hidden">{SECTION_TITLE}</caption>
              <thead>
                <tr>
                  <th scope="col" className={styles.nameColumn}>
                    Name
                  </th>
                  <th scope="col" className={styles.number}>
                    Planned cost
                  </th>
                  {planColumn && (
                    <th scope="col" className={styles.number}>
                      {planCurrencyColumn(planColumn)}
                    </th>
                  )}
                  <th scope="col" className={styles.number}>
                    {usdColumn(displayCode)}
                  </th>
                  <th scope="col" className={styles.number}>
                    {`Committed (${displayCode})`}
                  </th>
                  <th scope="col" className={styles.number}>
                    {`Invoiced (${displayCode})`}
                  </th>
                  <th scope="col">Spread</th>
                  <th scope="col">Completeness</th>
                  <th scope="col">SWBS request</th>
                  <th scope="col">
                    <span className="visually-hidden">Actions</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {items.map((item) => (
                  <Row
                    key={item.id}
                    item={item}
                    planColumn={planColumn !== null}
                    gsap={figuresOf(item.id)}
                    highlighted={item.id === highlight}
                    onSelectField={(field) => void navigate(fieldHref('lineItem', item.id, field))}
                    onView={() => void navigate(recordHref('lineItem', item.id))}
                    onEdit={() => void navigate(`${recordHref('lineItem', item.id)}/edit`)}
                    onDuplicate={() => actions.duplicate(item, activity)}
                    onDelete={() => actions.remove(item, activity)}
                    access={access}
                  />
                ))}
              </tbody>
            </table>
          </div>
        )}
        <UnassignedBalanceRow balance={balance} />
        {items.length > 0 && (
          <SpreadGrid
            grid={grid}
            noDates={!activity.startDate || (!activity.endDate && !activity.ongoing)}
          />
        )}
      </div>
      {actions.dialogs}
    </Card>
  );
}

/** A line item's GSAP figures, while they load, or none (no access, no data, a failure). */
type GsapCell = FinancialFiguresData | 'loading' | 'none';

interface RowProps {
  item: LineItem;
  /** Whether the plan-currency column is shown (016 FR-015). */
  planColumn: boolean;
  gsap: GsapCell;
  highlighted: boolean;
  onSelectField: (field: string) => void;
  onView: () => void;
  onEdit: () => void;
  onDuplicate: () => void;
  onDelete: () => void;
  /** What the person may do with the activity's line items (feature 014). */
  access?: LineItemAccess;
}

function Row({
  item,
  planColumn,
  gsap,
  highlighted,
  onSelectField,
  onView,
  onEdit,
  onDuplicate,
  onDelete,
  access,
}: RowProps) {
  return (
    <tr id={rowId(item.id)} className={cx(highlighted && styles.highlight)}>
      <td className={styles.nameColumn}>
        <Link to={recordHref('lineItem', item.id)} className={styles.name}>
          {item.name}
        </Link>
        <span className={styles.id}>{item.id}</span>
      </td>
      <td className={styles.number}>{moneyText(item.plannedCost)}</td>
      {planColumn && (
        <td className={styles.number}>
          {item.plannedCostPlan ? moneyText(item.plannedCostPlan) : '—'}
        </td>
      )}
      <td className={styles.number}>{moneyText(item.plannedCostDisplay)}</td>
      <GsapCells gsap={gsap} />
      <td className={styles.text}>{SPREAD_LABELS[item.spread.method]}</td>
      <td>
        <CompletenessChip
          missing={item.missing.map((field) => ({ id: field.field, label: field.label }))}
          onSelectField={onSelectField}
        />
      </td>
      <td>
        <SwbsChip status={item.swbsStatus} />
      </td>
      <td className={styles.actions}>
        <ActionsMenu
          label={item.name}
          items={[
            { label: VIEW, onSelect: onView },
            ...gateItem({ label: EDIT, onSelect: onEdit }, access?.edit),
            ...gateItem({ label: DUPLICATE, onSelect: onDuplicate }, access?.duplicate),
            ...gateItem(
              { label: DELETE, onSelect: onDelete, tone: 'danger', separatorBefore: true },
              access?.delete,
            ),
          ]}
        />
      </td>
    </tr>
  );
}

/**
 * Committed and Invoiced in the display currency (015 FR-016); Committed in red with an icon and
 * its name when it is above the line item's planned cost (FR-013, never colour alone).
 */
function GsapCells({ gsap }: { gsap: GsapCell }) {
  if (gsap === 'loading' || gsap === 'none') {
    const text = gsap === 'loading' ? '…' : '—';
    return (
      <>
        <td className={styles.number}>{text}</td>
        <td className={styles.number}>{text}</td>
      </>
    );
  }
  const over = gsap.committedAbovePlanned;
  return (
    <>
      <td className={cx(styles.number, over && styles.over)}>
        {over && (
          <CircleAlert
            role="img"
            aria-label="Committed above planned cost"
            className={styles.overIcon}
          />
        )}
        {moneyText(gsap.committed)}
      </td>
      <td className={styles.number}>{moneyText(gsap.invoiced)}</td>
    </>
  );
}

function rowId(lineItemId: string): string {
  return `line-item-${lineItemId}`;
}
