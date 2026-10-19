import { Pencil } from 'lucide-react';
import { ActionsMenu, Button, SidePanel, StatusChip } from '@/components';
import { formatPercent } from '@/lib/decimal';
import { useFormatters } from '@/lib/useFormatters';
import { MetaLine } from '../shared/MetaLine';
import { SETTING_NOUNS } from '../shared/settingsNouns';
import styles from '../shared/SettingsTable.module.css';
import { UsageLine } from '../shared/UsageLine';
import { moreActions } from './modelActions';
import type { ModelActions } from './modelActions';
import type { Model } from './modelsApi';
import { WaterfallExample } from './WaterfallExample';

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
        <>
          <Button variant="primary" icon={<Pencil />} onClick={() => actions.edit(model)}>
            Edit
          </Button>
          <ActionsMenu label={model.name} items={moreActions(model, actions)} />
        </>
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
