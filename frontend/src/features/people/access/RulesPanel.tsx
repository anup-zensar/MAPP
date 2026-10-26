import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { ApiError, conflictMessage } from '@/api/errors';
import { CheckboxField } from '@/ui/CheckboxField/CheckboxField';
import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { FormField } from '@/ui/FormField/FormField';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import { useFormatters } from '@/format/useFormatters';
import { useUpdateAccessRules } from './api';
import type { AccessRules, AccessRulesUpdate, Role } from './api';
import { NO_SELF_APPROVAL } from './access-words';
import { rulesSchema } from './schemas';
import type { RulesValues } from './schemas';

const FIELDS = ['noSelfApproval', 'newPersonRoleId'] as const;
/** The "no role" choice of the role for new people. */
const NO_ROLE = 'none';

export interface RulesPanelProps {
  rules: AccessRules;
  roles: Role[];
  onClose: () => void;
}

export function RulesPanel({ rules, roles, onClose }: RulesPanelProps) {
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const update = useUpdateAccessRules();
  const [start] = useState<RulesValues>(() => ({
    noSelfApproval: rules.noSelfApproval,
    newPersonRoleId: rules.newPersonRole?.id ?? NO_ROLE,
  }));
  const form = useForm<RulesValues>({
    resolver: zodResolver(rulesSchema),
    defaultValues: start,
    ...CHECK_ON_SAVE,
  });
  const { problems, onInvalid, showServerProblems } = useFormProblems(form, FIELDS);
  const [saving, setSaving] = useState(false);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  // Administrators are never made automatically (text 17): the role isn't offered.
  const options = [
    { id: NO_ROLE, label: 'No role', active: true },
    ...roles
      .filter((role) => role.builtIn !== 'administrator')
      .map((role) => ({ id: role.id, label: role.name, active: true })),
  ];

  async function save({ noSelfApproval, newPersonRoleId }: RulesValues) {
    const roleChanged = newPersonRoleId !== start.newPersonRoleId;
    if (noSelfApproval === start.noSelfApproval && !roleChanged) {
      onClose();
      return;
    }
    const body: AccessRulesUpdate = {
      ...(noSelfApproval !== start.noSelfApproval ? { noSelfApproval } : {}),
      ...(roleChanged
        ? { newPersonRoleId: newPersonRoleId === NO_ROLE ? '' : newPersonRoleId }
        : {}),
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
        showServerProblems(apiError.fields);
        setFailure(null);
      } else {
        showServerProblems([]);
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
      isDirty={form.formState.isDirty}
      saving={saving}
      problems={problems}
      failure={failure}
      onSave={() => void form.handleSubmit(save, onInvalid)()}
      onClose={onClose}
    >
      <FormField
        control={form.control}
        name="noSelfApproval"
        render={({ value, ...field }) => (
          <CheckboxField
            label={NO_SELF_APPROVAL}
            hint="Segregation of duties: an administrator decides on their behalf when they are a level's only approver."
            checked={value}
            {...field}
          />
        )}
      />
      <FormField
        control={form.control}
        name="newPersonRoleId"
        render={(field) => (
          <ChoiceField
            label="Role for new people"
            hint="Given with scope My teams to each person added in Settings › People."
            options={options}
            {...field}
            onChange={(value) => field.onChange(value || NO_ROLE)}
          />
        )}
      />
    </FormPanel>
  );
}
