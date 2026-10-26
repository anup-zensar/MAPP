import { Pencil } from 'lucide-react';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { Button } from '@/ui/Button/Button';
import { SidePanel } from '@/ui/SidePanel/SidePanel';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { MetaLine } from '../MetaLine';
import { SETTING_NOUNS } from '../settings-nouns';
import styles from '@/ui/settings-table.module.css';
import { UsageLine } from '../UsageLine';
import type { FinanceBudget, FinanceBudgetItem } from './financeBudgetsApi';
import { moreActions } from './financeItemActions';
import type { FinanceItemActions } from './financeItemActions';
import { CanChange } from '@/ui/CanChange/CanChange';

export interface FinanceBudgetItemDetailsProps {
  /** The item; null when it no longer exists. */
  item: FinanceBudgetItem | null;
  /** Its finance budget, with the plans using it as their default. */
  budget?: FinanceBudget;
  actions: FinanceItemActions;
  onClose: () => void;
}

/**
 * A finance budget item's details in a side panel (US5 scenario 4): its year, budget and code, the
 * line items linked to it, the plans using its budget as their default, and who created and last
 * changed it; **Edit** and ⋯ (Set inactive or Reactivate, Delete).
 */
export function FinanceBudgetItemDetails({
  item,
  budget,
  actions,
  onClose,
}: FinanceBudgetItemDetailsProps) {
  if (!item) {
    return (
      <SidePanel
        open
        onOpenChange={(open) => !open && onClose()}
        title="Finance budget item"
        footer={<Button onClick={onClose}>Back to Finance budget items</Button>}
      >
        <p role="alert" className={styles.gone}>
          {SETTING_NOUNS.financeBudgetItem.notFound}
        </p>
      </SidePanel>
    );
  }

  return (
    <SidePanel
      open
      onOpenChange={(open) => !open && onClose()}
      title={item.name}
      footer={
        <CanChange otherwise={<Button onClick={onClose}>Close</Button>}>
          <>
            <Button variant="primary" icon={<Pencil />} onClick={() => actions.edit(item)}>
              Edit
            </Button>
            <ActionsMenu label={item.name} items={moreActions(item, actions)} />
          </>
        </CanChange>
      }
    >
      <div className={styles.details}>
        <div className={styles.statusRow}>
          <StatusChip tone={item.active ? 'positive' : 'neutral'}>
            {item.active ? 'Active' : 'Inactive'}
          </StatusChip>
          <span>{item.id}</span>
        </div>
        <dl className={styles.facts}>
          <div>
            <dt>Budget year</dt>
            <dd>{item.year}</dd>
          </div>
          <div>
            <dt>Budget name</dt>
            <dd>
              {item.budgetName}
              {budget && <UsageLine usage={budget.usage} />}
            </dd>
          </div>
          <div>
            <dt>Code</dt>
            <dd>{item.code || '—'}</dd>
          </div>
          <div>
            <dt>Linked to</dt>
            <dd>
              <UsageLine usage={item.usage} />
            </dd>
          </div>
        </dl>
        <MetaLine record={item} />
      </div>
    </SidePanel>
  );
}
