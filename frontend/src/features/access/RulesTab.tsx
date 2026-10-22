import { Pencil } from 'lucide-react';
import { useState } from 'react';
import { ApiError, conflictMessage } from '@/api/errors';
import {
  Button,
  Card,
  CheckboxField,
  ChoiceField,
  ErrorState,
  FormPanel,
  problemOf,
  Skeleton,
  useToast,
} from '@/components';
import type { FieldProblem, SaveFailure } from '@/components';
import list from '@/features/settings/shared/SettingsTable.module.css';
import { useFormatters } from '@/lib/useFormatters';
import { useAccessRules, useRoles, useUpdateAccessRules } from './accessApi';
import type { AccessRules, AccessRulesUpdate, Role } from './accessApi';
import { SCOPE_WORDS } from './accessWords';
import styles from './access.module.css';

/** The rule of FR-026, as the card and the form say it. */
const NO_SELF_APPROVAL = 'Nobody approves an activity they own or submitted';
/** The "no role" choice of the role for new people. */
const NO_ROLE = 'none';

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

interface RulesPanelProps {
  rules: AccessRules;
  roles: Role[];
  onClose: () => void;
}

function RulesPanel({ rules, roles, onClose }: RulesPanelProps) {
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const update = useUpdateAccessRules();
  const [start] = useState(() => ({
    noSelfApproval: rules.noSelfApproval,
    roleId: rules.newPersonRole?.id ?? NO_ROLE,
  }));
  const [noSelfApproval, setNoSelfApproval] = useState(start.noSelfApproval);
  const [roleId, setRoleId] = useState(start.roleId);
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const isDirty = noSelfApproval !== start.noSelfApproval || roleId !== start.roleId;
  // Administrators are never made automatically (text 17): the role isn't offered.
  const options = [
    { id: NO_ROLE, label: 'No role', active: true },
    ...roles
      .filter((role) => role.builtIn !== 'administrator')
      .map((role) => ({ id: role.id, label: role.name, active: true })),
  ];

  async function save() {
    if (!isDirty) {
      onClose();
      return;
    }
    const body: AccessRulesUpdate = {
      ...(noSelfApproval !== start.noSelfApproval ? { noSelfApproval } : {}),
      ...(roleId !== start.roleId ? { newPersonRoleId: roleId === NO_ROLE ? '' : roleId } : {}),
      version: rules.version,
    };
    setSaving(true);
    try {
      await update.mutateAsync(body);
      show({ message: 'Access rules saved' });
      onClose();
    } catch (error) {
      const apiError = error instanceof ApiError ? error : new ApiError('other');
      if (apiError.kind === 'validation') {
        setProblems(apiError.fields);
        setFailure(null);
      } else {
        setProblems([]);
        setFailure({
          message:
            apiError.kind === 'conflict'
              ? conflictMessage(apiError, 'set of access rules', dateFormat)
              : apiError.message,
        });
      }
    } finally {
      setSaving(false);
    }
  }

  return (
    <FormPanel
      title="Edit access rules"
      isDirty={isDirty}
      saving={saving}
      problems={problems}
      failure={failure}
      onSave={() => void save()}
      onClose={onClose}
    >
      <CheckboxField
        label={NO_SELF_APPROVAL}
        name="noSelfApproval"
        hint="Segregation of duties: an administrator decides on their behalf when they are a level's only approver."
        checked={noSelfApproval}
        onChange={setNoSelfApproval}
        error={problemOf(problems, 'noSelfApproval')}
      />
      <ChoiceField
        label="Role for new people"
        name="newPersonRoleId"
        hint="Given with scope My teams to each person added in Settings › People."
        options={options}
        value={roleId}
        onChange={(value) => setRoleId(value || NO_ROLE)}
        error={problemOf(problems, 'newPersonRoleId')}
      />
    </FormPanel>
  );
}
