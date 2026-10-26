import { Pencil } from 'lucide-react';
import { useState } from 'react';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import list from '@/ui/settings-table.module.css';
import { useFormatters } from '@/format/useFormatters';
import { useAccessRules, useRoles } from './api';
import { NO_SELF_APPROVAL, SCOPE_WORDS } from './access-words';
import { RulesPanel } from './RulesPanel';
import styles from './access.module.css';

/**
 * The Rules tab (FR-019, FR-026): whether people may approve what they own or submitted, and the
 * role a person added in Settings › People receives (with scope My teams). **Edit** opens the
 * rules in a side panel, saved with their version.
 */
export function RulesTab() {
  const rules = useAccessRules();
  const roles = useRoles();
  const { dateTime } = useFormatters();
  const [editing, setEditing] = useState(false);

  if (!rules.data) {
    return (
      <Card title="Access rules">
        {rules.isError ? (
          <ErrorState error={rules.error} onRetry={() => void rules.refetch()} />
        ) : (
          <Skeleton lines={3} height="20px" />
        )}
      </Card>
    );
  }

  const data = rules.data;
  return (
    <Card
      title="Access rules"
      actions={
        <Button size="sm" icon={<Pencil />} onClick={() => setEditing(true)}>
          Edit rules
        </Button>
      }
    >
      <div className={styles.rules}>
        <dl className={list.facts}>
          <div>
            <dt>{NO_SELF_APPROVAL}</dt>
            <dd>
              {data.noSelfApproval
                ? 'On — someone else must approve.'
                : 'Off — owners and submitters may approve their own activities.'}
            </dd>
          </div>
          <div>
            <dt>Role for new people</dt>
            <dd>
              {data.newPersonRole
                ? `${data.newPersonRole.name} · ${SCOPE_WORDS.myTeams}`
                : 'No role — an administrator gives one.'}
            </dd>
          </div>
        </dl>
        <p className={list.meta}>
          {data.updatedAt && data.updatedBy
            ? `Last changed by ${data.updatedBy.name} on ${dateTime(data.updatedAt)}`
            : 'MAPP’s defaults, never changed.'}
        </p>
      </div>
      {editing && (
        <RulesPanel
          rules={data}
          roles={roles.data?.items ?? []}
          onClose={() => setEditing(false)}
        />
      )}
    </Card>
  );
}
