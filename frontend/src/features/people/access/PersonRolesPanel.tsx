import { useState } from 'react';
import { ApiError, conflictMessage } from '@/api/errors';
import { Button } from '@/ui/Button/Button';
import { CheckboxField } from '@/ui/CheckboxField/CheckboxField';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import { problemOf } from '@/ui/FormPanel/problem-of';
import { LockedNote } from '@/ui/LockedNote/LockedNote';
import { SidePanel } from '@/ui/SidePanel/SidePanel';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { TeamPicker } from '@/ui/TeamPicker/TeamPicker';
import { useToast } from '@/ui/Toast/useToast';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import type { TeamRef } from '@/ui/TeamPill/TeamPill';
import { useTeams } from '../api';
import { firstName } from '../texts';
import { useCloseAfterSave } from '@/ui/useCloseAfterSave';
import { useFormatters } from '@/format/useFormatters';
import { useCheckPersonRolesImpact, usePersonRoles, useRoles, useSetPersonRoles } from './api';
import type { AccessScopeType, ImpactWarning, PersonRoles, Role } from './api';
import { ADMINISTRATOR_SCOPE, SCOPE_WORDS, summariseAccess } from './access-words';
import {
  draftsDiffer,
  draftsOf,
  needsImpactCheck,
  previewPermissions,
  rolesBody,
} from './role-drafts';
import type { RoleDraft } from './role-drafts';
import styles from './access.module.css';

export interface PersonRolesPanelProps {
  personId: string;
  /** The person is the one signed in (taking away your own Administrator role is checked). */
  isYou: boolean;
  onClose: () => void;
}

/**
 * A person's roles (US3 scenarios 2, 4, 5; FR-017, FR-021, FR-022): one check box per role with its
 * description, and for each ticked role where it applies — All markets, My teams or Chosen teams.
 * The summary of what the roles allow follows every change. Taking away approving, or your own
 * Administrator role, is checked first and its warnings confirmed (Keep editing · Save changes).
 */
export function PersonRolesPanel({ personId, isYou, onClose }: PersonRolesPanelProps) {
  const person = usePersonRoles(personId);
  const roles = useRoles();
  const teams = useTeams();

  if (person.data && roles.data) {
    return (
      <RolesForm
        person={person.data}
        roles={roles.data.items}
        allTeams={teams.data?.items ?? person.data.teams}
        isYou={isYou}
        onClose={onClose}
      />
    );
  }

  const error = person.error ?? roles.error;
  return (
    <SidePanel
      open
      onOpenChange={(open) => !open && onClose()}
      title="Roles"
      footer={<Button onClick={onClose}>Back to Roles and access</Button>}
    >
      {error ? (
        <ErrorState
          error={error}
          onRetry={() => {
            void person.refetch();
            void roles.refetch();
          }}
        />
      ) : (
        <Skeleton lines={8} height="20px" />
      )}
    </SidePanel>
  );
}

interface RolesFormProps {
  person: PersonRoles;
  roles: Role[];
  allTeams: TeamRef[];
  isYou: boolean;
  onClose: () => void;
}

function RolesForm({ person, roles, allTeams, isYou, onClose }: RolesFormProps) {
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const save = useSetPersonRoles();
  const check = useCheckPersonRolesImpact();
  // The edit-form rule of feature 001: the roles are copied into the form once, when it opens.
  const [start] = useState(() => draftsOf(person));
  const [drafts, setDrafts] = useState(start);
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const [warnings, setWarnings] = useState<ImpactWarning[]>([]);
  // The panel is open while the address names the person: it closes once the save is known.
  const { saved, closeAfterSave } = useCloseAfterSave(onClose);
  const name = person.person.name;
  // Drafts follow the order of the roles list, so problems "roles.<i>…" find their role.
  const ordered = roles.flatMap((role) => drafts.filter((draft) => draft.roleId === role.id));
  const body = rolesBody(ordered, person.version);
  const isDirty = draftsDiffer(start, ordered);
  const summary = summariseAccess(previewPermissions(ordered, roles, person.teams, allTeams));

  function tick(role: Role, ticked: boolean) {
    setDrafts((current) =>
      ticked
        ? [
            ...current,
            {
              roleId: role.id,
              scope: role.builtIn === 'administrator' ? 'all' : role.defaultScope,
              teamIds: [],
            },
          ]
        : current.filter((draft) => draft.roleId !== role.id),
    );
  }

  function change(roleId: string, changes: Partial<RoleDraft>) {
    setDrafts((current) =>
      current.map((draft) => (draft.roleId === roleId ? { ...draft, ...changes } : draft)),
    );
  }

  function fail(error: unknown) {
    const apiError = error instanceof ApiError ? error : new ApiError('other');
    if (apiError.kind === 'validation') {
      setProblems(apiError.fields);
      setFailure(null);
      return;
    }
    setProblems([]);
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

  async function send() {
    setSaving(true);
    try {
      await save.mutateAsync({ personId: person.person.id, body });
      show({ message: `Roles of ${name} saved` });
      closeAfterSave();
    } catch (error) {
      setWarnings([]);
      fail(error);
    } finally {
      setSaving(false);
    }
  }

  async function onSave() {
    if (!isDirty) {
      onClose();
      return;
    }
    if (!needsImpactCheck(start, ordered, roles, isYou)) {
      await send();
      return;
    }
    setSaving(true);
    try {
      const impact = await check.mutateAsync({ personId: person.person.id, body });
      setSaving(false);
      if (impact.warnings.length > 0) {
        setProblems([]);
        setFailure(null);
        setWarnings(impact.warnings);
        return;
      }
      await send();
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
        onSave={() => void onSave()}
        onClose={onClose}
      >
        {roles.map((role) => {
          const index = ordered.findIndex((draft) => draft.roleId === role.id);
          const draft = index >= 0 ? ordered[index] : undefined;
          return (
            <div key={role.id} className={styles.roleBlock}>
              <CheckboxField
                label={role.name}
                name={`role-${role.id}`}
                hint={role.description || undefined}
                checked={Boolean(draft)}
                onChange={(ticked) => tick(role, ticked)}
                error={index >= 0 ? problemOf(problems, `roles.${index}.roleId`) : undefined}
              />
              {draft && (
                <ScopeChoice
                  role={role}
                  draft={draft}
                  index={index}
                  allTeams={allTeams}
                  problems={problems}
                  onChange={(changes) => change(role.id, changes)}
                />
              )}
            </div>
          );
        })}
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
        onConfirm={() => void send()}
      >
        {warnings.map((warning) => (
          <p key={warning.message}>{warning.message}</p>
        ))}
      </ConfirmDialog>
    </>
  );
}

interface ScopeChoiceProps {
  role: Role;
  draft: RoleDraft;
  index: number;
  allTeams: TeamRef[];
  problems: FieldProblem[];
  onChange: (changes: Partial<RoleDraft>) => void;
}

/** Where a ticked role applies: All markets, My teams, or Chosen teams with the team picker. */
function ScopeChoice({ role, draft, index, allTeams, problems, onChange }: ScopeChoiceProps) {
  if (role.builtIn === 'administrator') {
    return (
      <div className={styles.scope}>
        <LockedNote>{ADMINISTRATOR_SCOPE}</LockedNote>
      </div>
    );
  }
  const scopeError = problemOf(problems, `roles.${index}.scope`);
  const choices: AccessScopeType[] = ['all', 'myTeams', 'teams'];
  return (
    <fieldset className={styles.scope}>
      <legend className={styles.legend}>{`Where ${role.name} applies`}</legend>
      {choices.map((type) => (
        <label key={type} className={styles.radio}>
          <input
            type="radio"
            name={`scope-${role.id}`}
            value={type}
            checked={draft.scope === type}
            onChange={() => onChange({ scope: type })}
          />
          {SCOPE_WORDS[type]}
        </label>
      ))}
      {draft.scope === 'teams' && (
        <TeamPicker
          label="Teams"
          name={`roles.${index}.teamIds`}
          teams={allTeams}
          value={draft.teamIds}
          onChange={(teamIds) => onChange({ teamIds })}
          error={problemOf(problems, `roles.${index}.teamIds`)}
        />
      )}
      {scopeError && <p className={styles.problem}>{scopeError}</p>}
    </fieldset>
  );
}
