import { useQueryClient } from '@tanstack/react-query';
import { Pencil } from 'lucide-react';
import { useState } from 'react';
import { Link, useNavigate } from 'react-router';
import { ApiError } from '@/api/errors';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { Button } from '@/ui/Button/Button';
import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import { problemOf } from '@/ui/FormPanel/problem-of';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { SearchField } from '@/ui/SearchField/SearchField';
import { SidePanel } from '@/ui/SidePanel/SidePanel';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { useFormatters } from '@/format/useFormatters';
import { useUndoableDelete } from '@/ui/useUndoableDelete';
import list from '@/ui/settings-table.module.css';
import { CanChange } from '@/ui/CanChange/CanChange';
import { formFailure, useShowFailure } from './feedback';
import { OwnerImpactDialog } from './OwnerImpactDialog';
import { PersonField } from './PersonField';
import {
  fetchLatestTeam,
  useCheckTeamImpact,
  useCreateTeam,
  useDeleteTeam,
  useTeam,
  useUpdateTeam,
} from './api';
import type { ImpactWarning, Team, TeamColour, TeamUpdate } from './api';
import { RecordLine } from './RecordLine';
import { teamMoreItems } from './team-actions';
import type { TeamActions } from './team-actions';
import { removedIds } from './removed-ids';
import { deleteTeamText, teamInUseText, usedByText } from './texts';
import styles from './panels.module.css';

/** The team palette (D-025), in the order of the Colour field. */
const COLOURS: { id: TeamColour; label: string }[] = [
  { id: 'blue', label: 'Blue' },
  { id: 'violet', label: 'Violet' },
  { id: 'teal', label: 'Teal' },
  { id: 'magenta', label: 'Magenta' },
  { id: 'orange', label: 'Orange' },
  { id: 'green', label: 'Green' },
  { id: 'slate', label: 'Slate' },
  { id: 'gold', label: 'Gold' },
];

/** Above this many members, the details offer "Search members". */
const SEARCH_ABOVE = 10;

/** The lists Undo of a team delete refreshes: the team and its memberships are back. */
const UNDO_OPTIONS = { invalidate: [['teams'], ['people'], ['session']] };

export interface TeamDetailsPanelProps {
  teamId: string;
  actions: TeamActions;
  onClose: () => void;
}

/**
 * A team's details (US2-3): colour, status, description, members, the plans using it and who
 * created and last changed it. The one panel footer of Settings: **Edit** (primary) and ⋯ with Set
 * team inactive or Reactivate, and Delete.
 */
export function TeamDetailsPanel({ teamId, actions, onClose }: TeamDetailsPanelProps) {
  const team = useTeam(teamId);

  if (team.data) {
    const data = team.data;
    return (
      <SidePanel
        open
        onOpenChange={(open) => !open && onClose()}
        title={data.name}
        footer={
          <CanChange
            permission="people.manage"
            otherwise={<Button onClick={onClose}>Close</Button>}
          >
            <>
              <Button variant="primary" icon={<Pencil />} onClick={() => actions.edit(data)}>
                Edit
              </Button>
              <ActionsMenu label={data.name} items={teamMoreItems(data, actions)} />
            </>
          </CanChange>
        }
      >
        <TeamDetails team={data} />
      </SidePanel>
    );
  }

  const notFound = team.error instanceof ApiError && team.error.kind === 'notFound';
  return (
    <SidePanel
      open
      onOpenChange={(open) => !open && onClose()}
      title="Team"
      footer={<Button onClick={onClose}>Back to Teams</Button>}
    >
      {team.isPending && <Skeleton lines={6} height="18px" />}
      {notFound && <p className={list.gone}>{team.error?.message}</p>}
      {team.isError && !notFound && (
        <ErrorState error={team.error} onRetry={() => void team.refetch()} />
      )}
    </SidePanel>
  );
}

function TeamDetails({ team }: { team: Team }) {
  const [search, setSearch] = useState('');
  const colour = COLOURS.find((option) => option.id === team.colour);
  const query = search.trim().toLowerCase();
  const members = team.members.filter((member) =>
    `${member.name} ${member.jobTitle}`.toLowerCase().includes(query),
  );

  return (
    <div className={list.details}>
      <div className={list.statusRow}>
        <StatusChip tone={team.active ? 'positive' : 'neutral'}>
          {team.active ? 'Active' : 'Inactive'}
        </StatusChip>
      </div>
      <dl className={list.facts}>
        <div>
          <dt>Description</dt>
          <dd>{team.description || '—'}</dd>
        </div>
        <div>
          <dt>Colour</dt>
          <dd className={styles.colour}>
            <span
              className={styles.dot}
              style={{ background: `var(--team-${team.colour})` }}
              aria-hidden="true"
            />
            {colour?.label ?? team.colour}
          </dd>
        </div>
      </dl>

      <section className={styles.section} aria-labelledby="team-members">
        <h3 id="team-members" className={list.sectionTitle}>
          {`Members (${team.members.length})`}
        </h3>
        {team.members.length > SEARCH_ABOVE && (
          <SearchField
            label="Search members"
            placeholder="Search members"
            value={search}
            onChange={setSearch}
          />
        )}
        {team.members.length === 0 ? (
          <p className={list.muted}>
            No members yet. Add people so they can own this team's plans.
          </p>
        ) : (
          <ul role="list" className={styles.list}>
            {members.map((member) => (
              <li key={member.id}>
                <PersonLabel person={member} detail={member.jobTitle || undefined} />
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className={styles.section} aria-labelledby="team-plans">
        <h3 id="team-plans" className={list.sectionTitle}>
          Used by
        </h3>
        {team.plans.length === 0 ? (
          <p className={list.muted}>Not used by any plan yet.</p>
        ) : (
          <>
            <p className={list.muted}>{usedByText(team.plans.length)}</p>
            <ul role="list" className={styles.list}>
              {team.plans.map((plan) => (
                <li key={plan.id}>
                  {plan.inRecycleBin ? (
                    `${plan.name} (in the recycle bin)`
                  ) : (
                    <Link className={list.inlineLink} to={`/plans/${encodeURIComponent(plan.id)}`}>
                      {plan.name}
                    </Link>
                  )}
                </li>
              ))}
            </ul>
          </>
        )}
      </section>

      <RecordLine record={team} />
    </div>
  );
}

export interface TeamFormPanelProps {
  /** The team to edit; without one, the panel creates a new team. */
  team?: Team;
  onClose: () => void;
}

/**
 * "New team" or "Edit team" (US2-1, US2-4, US2-5): name, description, colour and members. When
 * members were removed, the API's impact check comes first and its warnings are confirmed in
 * OwnerImpactDialog (plan "Save flow").
 */
export function TeamFormPanel({ team, onClose }: TeamFormPanelProps) {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const createTeam = useCreateTeam();
  const updateTeam = useUpdateTeam();
  const checkImpact = useCheckTeamImpact();
  // The edit-form rule of feature 001: the team is copied into the form once, when it opens.
  const [start] = useState(() => ({
    name: team?.name ?? '',
    description: team?.description ?? '',
    colour: team?.colour ?? '',
    memberIds: team?.members.map((member) => member.id) ?? [],
  }));
  const [name, setName] = useState(start.name);
  const [description, setDescription] = useState(start.description);
  const [colour, setColour] = useState<string>(start.colour);
  const [memberIds, setMemberIds] = useState(start.memberIds);
  const [version, setVersion] = useState(team?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const [warnings, setWarnings] = useState<ImpactWarning[]>([]);

  const isDirty =
    name !== start.name ||
    description !== start.description ||
    colour !== start.colour ||
    memberIds.join() !== start.memberIds.join();
  // Only the fields that changed, and the version (D-203, D-210): after a conflict, Save again
  // replaces only what this person changed.
  const body: TeamUpdate = {
    ...(name !== start.name ? { name } : {}),
    ...(description !== start.description ? { description } : {}),
    ...(colour !== start.colour ? { colour } : {}),
    ...(memberIds.join() !== start.memberIds.join() ? { memberIds } : {}),
    version,
  };

  function done(saved: Team, verb: 'created' | 'saved') {
    show({
      message: `Team '${saved.name}' ${verb}`,
      action: {
        label: 'View',
        altText: 'Open it from the Teams list',
        onClick: () => void navigate(`/settings/teams/${encodeURIComponent(saved.id)}`),
      },
    });
    onClose();
  }

  async function fail(error: unknown) {
    const result = formFailure(error, 'team', dateFormat);
    setProblems(result.problems);
    setFailure(result.failure);
    if (result.conflict && team) {
      // Typed values stay; the form takes the version with the other person's change, so Save
      // again replaces it on purpose.
      const latest = await fetchLatestTeam(queryClient, team.id);
      if (latest) {
        setVersion(latest.version);
      }
    }
  }

  async function save() {
    setSaving(true);
    try {
      if (!team) {
        const created = await createTeam.mutateAsync({
          name,
          description,
          memberIds,
          ...(colour ? { colour } : {}),
        });
        done(created, 'created');
        return;
      }
      if (removedIds(start.memberIds, memberIds).length > 0) {
        const impact = await checkImpact.mutateAsync({ id: team.id, body });
        if (impact.warnings.length > 0) {
          setProblems([]);
          setFailure(null);
          setWarnings(impact.warnings);
          return;
        }
      }
      done(await updateTeam.mutateAsync({ id: team.id, body }), 'saved');
    } catch (error) {
      await fail(error);
    } finally {
      setSaving(false);
    }
  }

  async function saveAfterWarnings() {
    if (!team) {
      return;
    }
    setSaving(true);
    try {
      const saved = await updateTeam.mutateAsync({ id: team.id, body });
      setWarnings([]);
      done(saved, 'saved');
    } catch (error) {
      setWarnings([]);
      await fail(error);
    } finally {
      setSaving(false);
    }
  }

  const chosenColour = COLOURS.find((option) => option.id === colour);
  return (
    <>
      <FormPanel
        title={team ? 'Edit team' : 'New team'}
        isDirty={isDirty}
        saving={saving}
        problems={problems}
        failure={failure}
        onSave={save}
        onClose={onClose}
      >
        <TextField
          label="Name"
          name="name"
          marking="Required"
          value={name}
          onChange={setName}
          error={problemOf(problems, 'name')}
        />
        <TextAreaField
          label="Description"
          name="description"
          hint="Up to 300 characters."
          rows={3}
          value={description}
          onChange={setDescription}
          error={problemOf(problems, 'description')}
        />
        <ChoiceField
          label="Colour"
          name="colour"
          hint="Shown as a dot next to the team's name."
          options={COLOURS.map((option) => ({ ...option, active: true }))}
          value={colour}
          onChange={setColour}
          error={problemOf(problems, 'colour')}
          note={
            chosenColour && (
              <span className={styles.colour} aria-hidden="true">
                <span
                  className={styles.dot}
                  style={{ background: `var(--team-${chosenColour.id})` }}
                />
                {chosenColour.label}
              </span>
            )
          }
        />
        <PersonField
          label="Members"
          name="memberIds"
          hint="Only active people can be added."
          multiple
          value={memberIds}
          onChange={setMemberIds}
          error={problemOf(problems, 'memberIds')}
        />
      </FormPanel>
      <OwnerImpactDialog
        warnings={warnings}
        busy={saving}
        onKeepEditing={() => setWarnings([])}
        onSave={saveAfterWarnings}
      />
    </>
  );
}

export interface DeleteTeamDialogProps {
  team: Team;
  /** Sets the team inactive instead (a used team); resolves when done. */
  setInactive: (team: Team) => Promise<void>;
  /** The team is in the recycle bin. */
  onDeleted: (team: Team) => void;
  onClose: () => void;
}

/**
 * Delete of a team (US2-6, US2-7): a team that plans use can't be deleted, so the dialog lists
 * them and offers Set team inactive; an unused team goes to the recycle bin with its memberships,
 * and the message offers Undo. When a plan started using the team meanwhile, the API refuses
 * (in_use) and the dialog shows the refreshed team's plans.
 */
export function DeleteTeamDialog({ team, setInactive, onDeleted, onClose }: DeleteTeamDialogProps) {
  const queryClient = useQueryClient();
  const deleteTeam = useDeleteTeam();
  const afterDelete = useUndoableDelete(UNDO_OPTIONS);
  const showFailure = useShowFailure('team');
  const [shown, setShown] = useState(team);
  const [busy, setBusy] = useState(false);
  const used = shown.plans.length > 0;

  async function remove() {
    setBusy(true);
    try {
      afterDelete(await deleteTeam.mutateAsync(shown.id));
      onDeleted(shown);
      onClose();
    } catch (error) {
      if (error instanceof ApiError && error.kind === 'inUse') {
        const latest = await fetchLatestTeam(queryClient, shown.id);
        if (latest && latest.plans.length > 0) {
          setShown(latest);
          setBusy(false);
          return;
        }
      }
      showFailure(error);
      onClose();
    }
  }

  async function confirmSetInactive() {
    setBusy(true);
    await setInactive(shown);
    onClose();
  }

  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={used ? `'${shown.name}' can't be deleted` : `Delete team '${shown.name}'?`}
      confirmLabel={used ? 'Set team inactive' : 'Delete team'}
      tone={used ? undefined : 'danger'}
      busy={busy}
      onConfirm={used ? confirmSetInactive : remove}
    >
      {used ? teamInUseText(shown) : deleteTeamText(shown.members.length)}
    </ConfirmDialog>
  );
}
