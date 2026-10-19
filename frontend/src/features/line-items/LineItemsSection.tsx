import { Plus, ReceiptText } from 'lucide-react';
import { useEffect } from 'react';
import { Link, useLocation, useNavigate } from 'react-router';
import {
  ActionsMenu,
  Card,
  CompletenessChip,
  EmptyState,
  ErrorState,
  LinkButton,
  Skeleton,
} from '@/components';
import { cx } from '@/lib/cx';
import { fieldHref, recordHref } from '@/lib/recordLinks';
import { moneyText } from './lineItemMath';
import { useLineItemSection } from './lineItemsApi';
import type { LineItem } from './lineItemsApi';
import styles from './LineItemsSection.module.css';
import {
  ADD_LINE_ITEM,
  DELETE,
  DUPLICATE,
  EDIT,
  EMPTY_SECTION,
  LABEL,
  SECTION_SUBTITLE,
  SECTION_TITLE,
  SPREAD_LABELS,
  usdColumn,
  VIEW,
} from './lineItemText';
import { SpreadGrid } from './SpreadGrid';
import { SwbsChip } from './SwbsChip';
import { useLineItemActions } from './useLineItemActions';
import { UnassignedBalanceRow } from './UnassignedBalanceRow';

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

  const addButton = (
    <LinkButton to={`/activities/${encodeURIComponent(activityId)}/line-items/new`} icon={<Plus />}>
      {ADD_LINE_ITEM}
    </LinkButton>
  );

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
  const usdCode = balance.assigned.currency;

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
          <div className={styles.scroll}>
            <table className={styles.table}>
              <caption className="visually-hidden">{SECTION_TITLE}</caption>
              <thead>
                <tr>
                  <th scope="col">Name</th>
                  <th scope="col">{LABEL.currency}</th>
                  <th scope="col" className={styles.number}>
                    {LABEL.plannedCost}
                  </th>
                  <th scope="col" className={styles.number}>
                    {usdColumn(usdCode)}
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
                    highlighted={item.id === highlight}
                    onSelectField={(field) => void navigate(fieldHref('lineItem', item.id, field))}
                    onView={() => void navigate(recordHref('lineItem', item.id))}
                    onEdit={() => void navigate(`${recordHref('lineItem', item.id)}/edit`)}
                    onDuplicate={() => actions.duplicate(item, activity)}
                    onDelete={() => actions.remove(item, activity)}
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

interface RowProps {
  item: LineItem;
  highlighted: boolean;
  onSelectField: (field: string) => void;
  onView: () => void;
  onEdit: () => void;
  onDuplicate: () => void;
  onDelete: () => void;
}

function Row({
  item,
  highlighted,
  onSelectField,
  onView,
  onEdit,
  onDuplicate,
  onDelete,
}: RowProps) {
  return (
    <tr id={rowId(item.id)} className={cx(highlighted && styles.highlight)}>
      <td>
        <Link to={recordHref('lineItem', item.id)} className={styles.name}>
          {item.name}
        </Link>
        <span className={styles.id}>{item.id}</span>
      </td>
      <td>{item.currency.code}</td>
      <td className={styles.number}>{moneyText(item.plannedCost)}</td>
      <td className={styles.number}>{moneyText(item.plannedCostUsd)}</td>
      <td>{SPREAD_LABELS[item.spread.method]}</td>
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
            { label: EDIT, onSelect: onEdit },
            { label: DUPLICATE, onSelect: onDuplicate },
            { label: DELETE, onSelect: onDelete, tone: 'danger', separatorBefore: true },
          ]}
        />
      </td>
    </tr>
  );
}

function rowId(lineItemId: string): string {
  return `line-item-${lineItemId}`;
}
