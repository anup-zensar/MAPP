import { Pencil } from 'lucide-react';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { useCanChange } from '@/ui/useCanChange';
import { plural } from '@/format/text';
import styles from '@/ui/settings-table.module.css';
import type { FinanceBudget, FinanceBudgetItem } from './api';
import { moreActions } from './finance-item-actions';
import type { FinanceItemActions } from './finance-item-actions';
import page from './FinanceBudgetItemsPage.module.css';

export interface BudgetCardProps {
  budget: FinanceBudget;
  /** The budget's items that match the filters. */
  items: FinanceBudgetItem[];
  actions: FinanceItemActions;
  onRename: () => void;
}

/** One finance budget: its name, item count and the plans using it, with Rename budget. */
export function BudgetCard({ budget, items, actions, onRename }: BudgetCardProps) {
  const canChange = useCanChange();
  return (
    <Card
      title={budget.name}
      subtitle={`${plural(budget.items.length, 'item')} · ${budget.usage.summary}`}
      actions={
        canChange ? (
          <Button
            size="sm"
            icon={<Pencil />}
            aria-label={`Rename budget '${budget.name}'`}
            onClick={onRename}
          >
            Rename budget
          </Button>
        ) : undefined
      }
    >
      <div className={styles.tableScroll}>
        {/* Fixed column widths, so the columns of every budget's table line up (UI audit ST-04). */}
        <table className={`${styles.table} ${styles.fixed}`}>
          <colgroup>
            <col className={page.itemColumn} />
            <col className={page.codeColumn} />
            <col />
            <col className={page.statusColumn} />
            <col className={page.actionsColumn} />
          </colgroup>
          <thead>
            <tr>
              <th scope="col">Item</th>
              <th scope="col">Code</th>
              <th scope="col">Linked to</th>
              <th scope="col">Status</th>
              <th scope="col">
                <span className="visually-hidden">Actions</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id}>
                <th scope="row">
                  <button
                    type="button"
                    className={styles.nameButton}
                    onClick={() => actions.view(item)}
                  >
                    {item.name}
                  </button>
                </th>
                <td className={styles.nowrap}>{item.code || '—'}</td>
                <td>{item.usage.summary}</td>
                <td className={styles.nowrap}>
                  <StatusChip tone={item.active ? 'positive' : 'neutral'}>
                    {item.active ? 'Active' : 'Inactive'}
                  </StatusChip>
                </td>
                <td className={styles.actionsCell}>
                  <ActionsMenu
                    label={item.name}
                    items={[
                      { label: 'View', onSelect: () => actions.view(item) },
                      ...(canChange
                        ? [
                            { label: 'Edit', onSelect: () => actions.edit(item) },
                            ...moreActions(item, actions),
                          ]
                        : []),
                    ]}
                  />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Card>
  );
}
