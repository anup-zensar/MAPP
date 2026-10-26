import { Pencil } from 'lucide-react';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { Button } from '@/ui/Button/Button';
import { SidePanel } from '@/ui/SidePanel/SidePanel';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { formatPercent } from '@/format/decimal';
import { useFormatters } from '@/format/useFormatters';
import { MetaLine } from '../MetaLine';
import { SETTING_NOUNS } from '../settings-nouns';
import styles from '@/ui/settings-table.module.css';
import { UsageLine } from '../UsageLine';
import { moreActions } from './modelActions';
import type { ModelActions } from './modelActions';
import type { Model } from './modelsApi';
import { WaterfallExample } from './WaterfallExample';
import { CanChange } from '@/ui/CanChange/CanChange';

export interface ModelDetailsProps {
  /** The model; null when it no longer exists. */
  model: Model | null;
  actions: ModelActions;
  onClose: () => void;
}

/**
 * A model's details in a side panel (US1 scenario 3): every field, the worked example, the plans
 * using it, who created and last changed it; **Edit** and ⋯ (Duplicate, Set inactive or
 * Reactivate, Delete).
 */
export function ModelDetails({ model, actions, onClose }: ModelDetailsProps) {
  const { money } = useFormatters();

  if (!model) {
    return (
      <SidePanel
        open
        onOpenChange={(open) => !open && onClose()}
        title="Model"
        footer={<Button onClick={onClose}>Back to Models</Button>}
      >
        <p role="alert" className={styles.gone}>
          {SETTING_NOUNS.model.notFound}
        </p>
      </SidePanel>
    );
  }

  return (
    <SidePanel
      open
      onOpenChange={(open) => !open && onClose()}
      title={model.name}
      footer={
        <CanChange otherwise={<Button onClick={onClose}>Close</Button>}>
          <>
            <Button variant="primary" icon={<Pencil />} onClick={() => actions.edit(model)}>
              Edit
            </Button>
            <ActionsMenu label={model.name} items={moreActions(model, actions)} />
          </>
        </CanChange>
      }
    >
      <div className={styles.details}>
        <div className={styles.statusRow}>
          <StatusChip tone={model.active ? 'positive' : 'neutral'}>
            {model.active ? 'Active' : 'Inactive'}
          </StatusChip>
          <span>{model.id}</span>
        </div>
        <dl className={styles.facts}>
          <div>
            <dt>INQ→MQL rate</dt>
            <dd className={styles.numeric}>{formatPercent(model.inqToMqlRate)}</dd>
          </div>
          <div>
            <dt>MQL→deal rate</dt>
            <dd className={styles.numeric}>{formatPercent(model.mqlToDealRate)}</dd>
          </div>
          <div>
            <dt>Average deal size</dt>
            <dd className={styles.numeric}>{money(model.averageDealSize)}</dd>
          </div>
          <div>
            <dt>Allowed activity types</dt>
            <dd>
              <ul role="list" className={styles.chips}>
                {model.activityTypes.map((type) => (
                  <li key={type.id} className={styles.chip}>
                    {type.active ? type.name : `${type.name} (inactive)`}
                  </li>
                ))}
              </ul>
            </dd>
          </div>
          <div>
            <dt>Used by</dt>
            <dd>
              <UsageLine usage={model.usage} />
            </dd>
          </div>
        </dl>
        <WaterfallExample
          inqToMqlRate={model.inqToMqlRate}
          mqlToDealRate={model.mqlToDealRate}
          averageDealSize={model.averageDealSize.amount}
        />
        <MetaLine record={model} />
      </div>
    </SidePanel>
  );
}
