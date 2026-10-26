import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect, useState } from 'react';
import type { FormEvent } from 'react';
import { useForm } from 'react-hook-form';
import { useNavigate } from 'react-router';
import { ApiError, conflictMessage } from '@/api/errors';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { CheckboxField } from '@/ui/CheckboxField/CheckboxField';
import { FormErrorSummary } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import { FormField } from '@/ui/FormField/FormField';
import { FormFooter } from '@/ui/FormFooter/FormFooter';
import { FailureMessage } from '@/ui/FormPanel/FormPanel';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { FormSection } from '@/ui/FormSection/FormSection';
import { LockedNote } from '@/ui/LockedNote/LockedNote';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import { useUnsavedChanges } from '@/ui/useUnsavedChanges';
import type { DateFormat } from '@/format/intl';
import { useFormatters } from '@/format/useFormatters';
import { useCreateRole, useUpdateRole } from './api';
import type { PermissionCatalogue, PermissionKey, Role } from './api';
import { ADMINISTRATOR_LOCK, SCOPE_WORDS } from './access-words';
import { rolePath, rolesTabPath } from './access-paths';
import styles from './access.module.css';
import { inTableOrder, newRole, roleChanges, roleSchema } from './schemas';
import type { RoleValues } from './schemas';

const FIELDS = ['name', 'description', 'defaultScope', 'permissions'] as const;
const DEFAULT_SCOPES: Role['defaultScope'][] = ['myTeams', 'all'];

export interface RoleFormProps {
  /** The role being edited; without one, the form creates a role. */
  role?: Role;
  start: RoleValues;
  catalogue: PermissionCatalogue;
}

export function RoleForm({ role, start, catalogue }: RoleFormProps) {
  const navigate = useNavigate();
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const createRole = useCreateRole();
  const updateRole = useUpdateRole();
  const form = useForm<RoleValues>({
    resolver: zodResolver(roleSchema),
    defaultValues: { ...start, permissions: inTableOrder(start.permissions) },
    ...CHECK_ON_SAVE,
  });
  const { problems, onInvalid, showServerProblems } = useFormProblems(form, FIELDS);
  // After a conflict, Save again sends the version with the other person's change (D-203).
  const [followLatest, setFollowLatest] = useState(false);
  const [startVersion] = useState(role?.version ?? 0);
  const version = followLatest ? (role?.version ?? 0) : startVersion;
  const [saving, setSaving] = useState(false);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  // Where to go once saved: the guard lets the page go without asking.
  const [leaveTo, setLeaveTo] = useState<string | null>(null);
  const { dialog } = useUnsavedChanges(form.formState.isDirty && leaveTo === null);

  useEffect(() => {
    if (leaveTo !== null) {
      void navigate(leaveTo);
    }
  }, [leaveTo, navigate]);
  const backTo = role ? rolePath(role.id) : rolesTabPath();

  async function save(values: RoleValues) {
    setSaving(true);
    try {
      const result = role
        ? await updateRole.mutateAsync({
            roleId: role.id,
            body: roleChanges(start, values, version),
          })
        : await createRole.mutateAsync(newRole(values));
      show({ message: `Role '${result.name}' ${role ? 'saved' : 'created'}` });
      setLeaveTo(rolePath(result.id));
    } catch (error) {
      const result = roleFailure(error, dateFormat);
      showServerProblems(result.problems);
      setFailure(result.failure);
      setFollowLatest((follow) => follow || result.conflict);
    } finally {
      setSaving(false);
    }
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!saving) {
      void form.handleSubmit(save, onInvalid)(event);
    }
  }

  if (role?.builtIn === 'administrator') {
    return (
      <Card>
        <LockedNote>{ADMINISTRATOR_LOCK}</LockedNote>
      </Card>
    );
  }

  return (
    <>
      <form noValidate aria-busy={saving} className={styles.cards} onSubmit={submit}>
        {failure && <FailureMessage failure={failure} />}
        <FormErrorSummary errors={problems} />
        <Card>
          <FormSection title="Basics">
            <FormField
              control={form.control}
              name="name"
              render={(field) => (
                <TextField label="Name" marking="Required" maxLength={60} {...field} />
              )}
            />
            <FormField
              control={form.control}
              name="description"
              render={(field) => (
                <TextAreaField
                  label="Description"
                  hint="One sentence on who the role is for."
                  maxLength={300}
                  rows={2}
                  {...field}
                />
              )}
            />
            <FormField
              control={form.control}
              name="defaultScope"
              render={(field) => (
                <fieldset className={styles.group}>
                  <legend className={styles.legend}>Default scope</legend>
                  <p className={styles.note}>
                    Where the role applies when it is given to someone; it can be changed for each
                    person.
                  </p>
                  {DEFAULT_SCOPES.map((scope) => (
                    <label key={scope} className={styles.radio}>
                      <input
                        type="radio"
                        name="defaultScope"
                        value={scope}
                        checked={field.value === scope}
                        onChange={() => field.onChange(scope)}
                      />
                      {SCOPE_WORDS[scope]}
                    </label>
                  ))}
                </fieldset>
              )}
            />
          </FormSection>
        </Card>
        <Card>
          <FormSection
            title="Permissions"
            description="What people with this role may do. Choose at least one."
          >
            <FormField
              control={form.control}
              name="permissions"
              render={(field) => (
                <PermissionGroups
                  catalogue={catalogue}
                  value={field.value}
                  onChange={field.onChange}
                  error={field.error}
                />
              )}
            />
          </FormSection>
        </Card>
        <FormFooter>
          <Button type="submit" variant="primary">
            {role ? 'Save' : 'Create role'}
          </Button>
          <Button onClick={() => void navigate(backTo)}>Cancel</Button>
        </FormFooter>
      </form>
      {dialog}
    </>
  );
}

interface PermissionGroupsProps {
  catalogue: PermissionCatalogue;
  value: PermissionKey[];
  onChange: (permissions: PermissionKey[]) => void;
  error?: string;
}

/** One check box per permission, by group; the ticked ones stay in Table A's order. */
function PermissionGroups({ catalogue, value, onChange, error }: PermissionGroupsProps) {
  function toggle(key: PermissionKey, ticked: boolean) {
    onChange(
      ticked ? inTableOrder([...value, key]) : value.filter((permission) => permission !== key),
    );
  }

  return (
    <div className={styles.groups} id="permissions">
      {catalogue.groups.map((group) => (
        <fieldset key={group.name} className={styles.group}>
          <legend className={styles.groupTitle}>{group.name}</legend>
          {group.permissions.map((permission) => (
            <CheckboxField
              key={permission.key}
              label={permission.label}
              name={`permission-${permission.key}`}
              hint={permission.scoped ? 'Limited to the role’s scope' : undefined}
              checked={value.includes(permission.key)}
              onChange={(ticked) => toggle(permission.key, ticked)}
            />
          ))}
        </fieldset>
      ))}
      {error && <p className={styles.problem}>{error}</p>}
    </div>
  );
}

interface RoleFailure {
  problems: FieldProblem[];
  failure: SaveFailure | null;
  conflict: boolean;
}

/** A failed save: field problems (or the API's message when it names none), or a conflict. */
function roleFailure(error: unknown, dateFormat: DateFormat): RoleFailure {
  const apiError = error instanceof ApiError ? error : new ApiError('other');
  if (apiError.kind === 'validation' || apiError.kind === 'nameTaken') {
    const failure = apiError.fields.length > 0 ? null : { message: apiError.message };
    return { problems: apiError.fields, failure, conflict: false };
  }
  if (apiError.kind === 'conflict') {
    const failure = {
      message: conflictMessage(apiError, 'role', dateFormat),
      nextStep:
        'Your changes are still here. Save again to replace their changes, or Cancel to see theirs.',
    };
    return { problems: [], failure, conflict: true };
  }
  return { problems: [], failure: { message: apiError.message }, conflict: false };
}
