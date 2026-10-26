import { useEffect, useState } from 'react';
import type { FormEvent } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router';
import { ApiError, conflictMessage } from '@/api/errors';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { CheckboxField } from '@/ui/CheckboxField/CheckboxField';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { FormErrorSummary } from '@/ui/FormErrorSummary/FormErrorSummary';
import { FormFooter } from '@/ui/FormFooter/FormFooter';
import { FailureMessage } from '@/ui/FormPanel/FormPanel';
import { problemOf } from '@/ui/FormPanel/problemOf';
import { FormSection } from '@/ui/FormSection/FormSection';
import { LockedNote } from '@/ui/LockedNote/LockedNote';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { SaveFailure } from '@/ui/FormPanel/problemOf';
import { useFormatters } from '@/lib/useFormatters';
import { usePageTitle } from '@/lib/usePageTitle';
import { useUnsavedChanges } from '@/lib/useUnsavedChanges';
import { useCreateRole, usePermissionCatalogue, useRole, useUpdateRole } from './accessApi';
import type { PermissionCatalogue, PermissionKey, Role, RoleUpdate } from './accessApi';
import { ADMINISTRATOR_LOCK, permissionOrder, SCOPE_WORDS } from './accessWords';
import { rolePath, rolesTabPath } from './accessPaths';
import styles from './access.module.css';

type DefaultScope = Role['defaultScope'];

/** What the form holds. */
interface RoleValues {
  name: string;
  description: string;
  defaultScope: DefaultScope;
  permissions: PermissionKey[];
}

const EMPTY: RoleValues = { name: '', description: '', defaultScope: 'myTeams', permissions: [] };

/**
 * New role (empty, or `?copyOf=RL-…` filled in from another role as "Copy of …") at
 * /settings/access/roles/new, and Edit role at /settings/access/roles/:roleId/edit (US4, FR-018):
 * name, description, default scope and the permissions by group. Unsaved changes are protected.
 */
export function RoleFormPage() {
  const { roleId } = useParams();
  const [params] = useSearchParams();
  const copyOf = params.get('copyOf') ?? '';
  const sourceId = roleId ?? copyOf;
  const source = useRole(sourceId);
  const catalogue = usePermissionCatalogue();
  const editing = Boolean(roleId);
  const title = editing ? `Edit role${source.data ? ` '${source.data.name}'` : ''}` : 'New role';
  usePageTitle(editing ? 'Edit role' : 'New role');

  const breadcrumb = [
    { label: 'Settings', to: '/settings' },
    { label: 'Roles and access', to: rolesTabPath() },
    ...(editing && source.data ? [{ label: source.data.name, to: rolePath(source.data.id) }] : []),
    { label: editing ? 'Edit' : 'New role' },
  ];

  const waitingForSource = Boolean(sourceId) && !source.data;
  if (waitingForSource || !catalogue.data) {
    const error = (sourceId ? source.error : null) ?? catalogue.error;
    return (
      <>
        <PageHeader breadcrumb={breadcrumb} title={title} />
        <Card>
          {error ? (
            <ErrorState
              error={error}
              onRetry={() => {
                void source.refetch();
                void catalogue.refetch();
              }}
            />
          ) : (
            <Skeleton lines={8} height="20px" />
          )}
        </Card>
      </>
    );
  }

  const start: RoleValues = source.data
    ? {
        name: editing ? source.data.name : `Copy of ${source.data.name}`,
        description: source.data.description,
        defaultScope: source.data.defaultScope,
        permissions: source.data.permissions,
      }
    : EMPTY;

  return (
    <>
      <PageHeader breadcrumb={breadcrumb} title={title} />
      <RoleForm
        key={sourceId}
        role={editing ? source.data : undefined}
        start={start}
        catalogue={catalogue.data}
      />
    </>
  );
}

interface RoleFormProps {
  /** The role being edited; without one, the form creates a role. */
  role?: Role;
  start: RoleValues;
  catalogue: PermissionCatalogue;
}

function RoleForm({ role, start, catalogue }: RoleFormProps) {
  const navigate = useNavigate();
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const createRole = useCreateRole();
  const updateRole = useUpdateRole();
  const [values, setValues] = useState(start);
  // After a conflict, Save again sends the version with the other person's change (D-203).
  const [followLatest, setFollowLatest] = useState(false);
  const [startVersion] = useState(role?.version ?? 0);
  const version = followLatest ? (role?.version ?? 0) : startVersion;
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const permissions = [...values.permissions].sort(
    (a, b) => permissionOrder(a) - permissionOrder(b),
  );
  const changed = {
    name: values.name !== start.name,
    description: values.description !== start.description,
    defaultScope: values.defaultScope !== start.defaultScope,
    permissions:
      permissions.join() !==
      [...start.permissions].sort((a, b) => permissionOrder(a) - permissionOrder(b)).join(),
  };
  const isDirty = Object.values(changed).some(Boolean);
  // Where to go once saved: the guard lets the page go without asking.
  const [leaveTo, setLeaveTo] = useState<string | null>(null);
  const { dialog } = useUnsavedChanges(isDirty && leaveTo === null);

  useEffect(() => {
    if (leaveTo !== null) {
      void navigate(leaveTo);
    }
  }, [leaveTo, navigate]);
  const isAdministrator = role?.builtIn === 'administrator';
  const backTo = role ? rolePath(role.id) : rolesTabPath();

  function set<K extends keyof RoleValues>(key: K, value: RoleValues[K]) {
    setValues((current) => ({ ...current, [key]: value }));
  }

  function toggle(key: PermissionKey, ticked: boolean) {
    setValues((current) => ({
      ...current,
      permissions: ticked
        ? [...current.permissions, key]
        : current.permissions.filter((permission) => permission !== key),
    }));
  }

  async function save() {
    setSaving(true);
    try {
      let result: Role;
      if (role) {
        // Only what changed, and the version (D-203, D-210).
        const body: RoleUpdate = {
          ...(changed.name ? { name: values.name } : {}),
          ...(changed.description ? { description: values.description } : {}),
          ...(changed.defaultScope ? { defaultScope: values.defaultScope } : {}),
          ...(changed.permissions ? { permissions } : {}),
          version,
        };
        result = await updateRole.mutateAsync({ roleId: role.id, body });
      } else {
        result = await createRole.mutateAsync({
          name: values.name,
          description: values.description,
          defaultScope: values.defaultScope,
          permissions,
        });
      }
      show({ message: `Role '${result.name}' ${role ? 'saved' : 'created'}` });
      setLeaveTo(rolePath(result.id));
    } catch (error) {
      const apiError = error instanceof ApiError ? error : new ApiError('other');
      if (apiError.kind === 'validation' || apiError.kind === 'nameTaken') {
        setProblems(apiError.fields);
        setFailure(apiError.fields.length > 0 ? null : { message: apiError.message });
      } else if (apiError.kind === 'conflict') {
        setProblems([]);
        setFailure({
          message: conflictMessage(apiError, 'role', dateFormat),
          nextStep:
            'Your changes are still here. Save again to replace their changes, or Cancel to see theirs.',
        });
        setFollowLatest(true);
      } else {
        setProblems([]);
        setFailure({ message: apiError.message });
      }
    } finally {
      setSaving(false);
    }
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!saving) {
      void save();
    }
  }

  if (isAdministrator) {
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
            <TextField
              label="Name"
              name="name"
              marking="Required"
              maxLength={60}
              value={values.name}
              onChange={(name) => set('name', name)}
              error={problemOf(problems, 'name')}
            />
            <TextAreaField
              label="Description"
              name="description"
              hint="One sentence on who the role is for."
              maxLength={300}
              rows={2}
              value={values.description}
              onChange={(description) => set('description', description)}
              error={problemOf(problems, 'description')}
            />
            <fieldset className={styles.group}>
              <legend className={styles.legend}>Default scope</legend>
              <p className={styles.note}>
                Where the role applies when it is given to someone; it can be changed for each
                person.
              </p>
              {(['myTeams', 'all'] as DefaultScope[]).map((scope) => (
                <label key={scope} className={styles.radio}>
                  <input
                    type="radio"
                    name="defaultScope"
                    value={scope}
                    checked={values.defaultScope === scope}
                    onChange={() => set('defaultScope', scope)}
                  />
                  {SCOPE_WORDS[scope]}
                </label>
              ))}
            </fieldset>
          </FormSection>
        </Card>
        <Card>
          <FormSection
            title="Permissions"
            description="What people with this role may do. Choose at least one."
          >
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
                      checked={values.permissions.includes(permission.key)}
                      onChange={(ticked) => toggle(permission.key, ticked)}
                    />
                  ))}
                </fieldset>
              ))}
              {problemOf(problems, 'permissions') && (
                <p className={styles.problem}>{problemOf(problems, 'permissions')}</p>
              )}
            </div>
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
