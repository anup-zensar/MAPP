import { Pencil } from 'lucide-react';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { Button } from '@/ui/Button/Button';
import { SidePanel } from '@/ui/SidePanel/SidePanel';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { useFormatters } from '@/format/useFormatters';
import { MetaLine } from '../MetaLine';
import { SETTING_NOUNS } from '../settings-nouns';
import styles from '@/ui/settings-table.module.css';
import { UsageLine } from '../UsageLine';
import { moreActions } from './activity-type-actions';
import type { ActivityTypeActions } from './activity-type-actions';
import type { ActivityType } from './api';
import { CanChange } from '@/ui/CanChange/CanChange';

export interface ActivityTypeDetailsProps {
  /** The activity type; null when it no longer exists. */
  type: ActivityType | null;
  actions: ActivityTypeActions;
  onClose: () => void;
}

/**
 * An activity type's details in a side panel (US5 scenario 2): its default planned cost, the
 * models allowing it, where it is used and who created and last changed it; **Edit** and ⋯ (Set
 * inactive or Reactivate, Delete).
 */
export function ActivityTypeDetails({ type, actions, onClose }: ActivityTypeDetailsProps) {
  const { money } = useFormatters();

  if (!type) {
    return (
      <SidePanel
        open
        onOpenChange={(open) => !open && onClose()}
        title="Activity type"
        footer={<Button onClick={onClose}>Back to Activity types</Button>}
      >
        <p role="alert" className={styles.gone}>
          {SETTING_NOUNS.activityType.notFound}
        </p>
      </SidePanel>
    );
  }

  return (
    <SidePanel
      open
      onOpenChange={(open) => !open && onClose()}
      title={type.name}
      footer={
        <CanChange otherwise={<Button onClick={onClose}>Close</Button>}>
          <>
            <Button variant="primary" icon={<Pencil />} onClick={() => actions.edit(type)}>
              Edit
            </Button>
            <ActionsMenu label={type.name} items={moreActions(type, actions)} />
          </>
        </CanChange>
      }
    >
      <div className={styles.details}>
        <div className={styles.statusRow}>
          <StatusChip tone={type.active ? 'positive' : 'neutral'}>
            {type.active ? 'Active' : 'Inactive'}
          </StatusChip>
          <span>{type.id}</span>
        </div>
        <dl className={styles.facts}>
          <div>
            <dt>Default planned cost</dt>
            <dd className={styles.numeric}>
              {type.defaultPlannedCost ? money(type.defaultPlannedCost) : '—'}
            </dd>
          </div>
          <div>
            <dt>Allowed in models</dt>
            <dd>
              {type.models.length === 0 ? (
                '—'
              ) : (
                <ul role="list" className={styles.chips}>
                  {type.models.map((model) => (
                    <li key={model.id} className={styles.chip}>
                      {model.active ? model.name : `${model.name} (inactive)`}
                    </li>
                  ))}
                </ul>
              )}
            </dd>
          </div>
          <div>
            <dt>Used by</dt>
            <dd>
              <UsageLine usage={type.usage} />
            </dd>
          </div>
        </dl>
        <MetaLine record={type} />
      </div>
    </SidePanel>
  );
}
