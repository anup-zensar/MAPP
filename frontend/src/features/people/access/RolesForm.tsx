import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm, useWatch } from 'react-hook-form';
import { ApiError, conflictMessage } from '@/api/errors';
import { CheckboxField } from '@/ui/CheckboxField/CheckboxField';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { FormField } from '@/ui/FormField/FormField';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import { problemOf } from '@/ui/FormPanel/problem-of';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import type { TeamRef } from '@/ui/TeamPill/TeamPill';
import { useToast } from '@/ui/Toast/useToast';
import { useCloseAfterSave } from '@/ui/useCloseAfterSave';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import { useFormatters } from '@/format/useFormatters';
import { firstName } from '../texts';
import { useCheckPersonRolesImpact, useSetPersonRoles } from './api';
import type { ImpactWarning, PersonRoles, PersonRolesUpdate, Role } from './api';
import { summariseAccess } from './access-words';
import styles from './access.module.css';
import {
  draftsDiffer,
  draftsOf,
  inRolesOrder,
  needsImpactCheck,
  previewPermissions,
  rolesBody,
  withDraftChange,
  withRoleTicked,
} from './role-drafts';
import { ScopeChoice } from './ScopeChoice';
import { personRolesSchema } from './schemas';
import type { PersonRolesValues } from './schemas';

const FIELDS = ['roles'] as const;

export interface RolesFormProps {
  person: PersonRoles;
  roles: Role[];
  allTeams: TeamRef[];
  isYou: boolean;
  onClose: () => void;
}

export function RolesForm({ person, roles, allTeams, isYou, onClose }: RolesFormProps) {
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const save = useSetPersonRoles();
  const check = useCheckPersonRolesImpact();
  // The edit-form rule of feature 001: the roles are copied into the form once, when it opens.
  const [start] = useState(() => draftsOf(person));
  const form = useForm<PersonRolesValues>({
    resolver: zodResolver(personRolesSchema),
    defaultValues: { roles: start },
    ...CHECK_ON_SAVE,
  });
  const { problems, onInvalid, showServerProblems } = useFormProblems(form, FIELDS);
  const [saving, setSaving] = useState(false);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const [warnings, setWarnings] = useState<ImpactWarning[]>([]);
  // The panel is open while the address names the person: it closes once the save is known.
  const { saved, closeAfterSave } = useCloseAfterSave(onClose);
  const name = person.person.name;
  const drafts = useWatch({ control: form.control, name: 'roles' });
  const ordered = inRolesOrder(drafts, roles);
  const body = rolesBody(ordered, person.version);
  // Order, and teams kept while another scope is chosen, are no change (unlike formState.isDirty).
  const isDirty = draftsDiffer(start, ordered);
  const summary = summariseAccess(previewPermissions(ordered, roles, person.teams, allTeams));

  function fail(error: unknown) {
    const apiError = error instanceof ApiError ? error : new ApiError('other');
    if (apiError.kind === 'validation') {
      showServerProblems(apiError.fields);
      setFailure(null);
      return;
    }
    showServerProblems([]);
    setFailure({
      message:
        apiError.kind === 'conflict'
          ? conflictMessage(apiError, 'person', dateFormat)
          : apiError.message,
      nextStep:
        apiError.kind === 'conflict'
          ? 'Close the panel and open it again to see their change.'
          : undefined,
    });
  }

  async function send(update: PersonRolesUpdate) {
    setSaving(true);
    try {
      await save.mutateAsync({ personId: person.person.id, body: update });
      show({ message: `Roles of ${name} saved` });
      closeAfterSave();
    } catch (error) {
      setWarnings([]);
      fail(error);
    } finally {
      setSaving(false);
    }
  }

  async function onSave(values: PersonRolesValues) {
    const after = inRolesOrder(values.roles, roles);
    if (!draftsDiffer(start, after)) {
      onClose();
      return;
    }
    const update = rolesBody(after, person.version);
    if (!needsImpactCheck(start, after, roles, isYou)) {
      await send(update);
      return;
    }
    setSaving(true);
    try {
      const impact = await check.mutateAsync({ personId: person.person.id, body: update });
      setSaving(false);
      if (impact.warnings.length > 0) {
        showServerProblems([]);
        setFailure(null);
        setWarnings(impact.warnings);
        return;
      }
      await send(update);
    } catch (error) {
      setSaving(false);
      fail(error);
    }
  }

  return (
    <>
      <FormPanel
        title={`Roles of ${name}`}
        description={
          person.teams.length > 0
            ? `Teams: ${person.teams.map((team) => team.name).join(', ')}`
            : 'Not in any team yet: My teams covers no plan.'
        }
        isDirty={isDirty && !saved}
        saving={saving}
        problems={problems}
        failure={failure}
        onSave={() => void form.handleSubmit(onSave, onInvalid)()}
        onClose={onClose}
      >
        <FormField
          control={form.control}
          name="roles"
          render={(field) =>
            roles.map((role) => {
              const index = ordered.findIndex((draft) => draft.roleId === role.id);
              const draft = ordered[index];
              return (
                <div key={role.id} className={styles.roleBlock}>
                  <CheckboxField
                    label={role.name}
                    name={`role-${role.id}`}
                    hint={role.description || undefined}
                    checked={Boolean(draft)}
                    onChange={(ticked) => field.onChange(withRoleTicked(field.value, role, ticked))}
                    error={draft ? problemOf(problems, `roles.${index}.roleId`) : undefined}
                  />
                  {draft && (
                    <ScopeChoice
                      role={role}
                      draft={draft}
                      index={index}
                      allTeams={allTeams}
                      problems={problems}
                      onChange={(changes) =>
                        field.onChange(withDraftChange(field.value, role.id, changes))
                      }
                    />
                  )}
                </div>
              );
            })
          }
        />
        <section className={styles.preview} aria-labelledby="roles-preview">
          <h3 id="roles-preview" className={styles.previewTitle}>
            {`What ${firstName(name)} can do`}
          </h3>
          <ul className={styles.summary} aria-live="polite">
            {summary.map((line) => (
              <li key={line}>{line}</li>
            ))}
          </ul>
        </section>
      </FormPanel>
      <ConfirmDialog
        open={warnings.length > 0}
        onOpenChange={(open) => !open && setWarnings([])}
        title={`Save the roles of ${name}?`}
        cancelLabel="Keep editing"
        confirmLabel="Save changes"
        busy={saving}
        onConfirm={() => void send(body)}
      >
        {warnings.map((warning) => (
          <p key={warning.message}>{warning.message}</p>
        ))}
      </ConfirmDialog>
    </>
  );
}
