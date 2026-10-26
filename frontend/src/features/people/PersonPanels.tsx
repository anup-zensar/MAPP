import { useQueryClient } from '@tanstack/react-query';
import { Pencil } from 'lucide-react';
import { useState } from 'react';
import { useNavigate } from 'react-router';
import { ApiError } from '@/api/errors';
import { useCurrentUser } from '@/session/useSession';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { Avatar } from '@/ui/Avatar/Avatar';
import { Button } from '@/ui/Button/Button';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import { problemOf } from '@/ui/FormPanel/problem-of';
import { SearchField } from '@/ui/SearchField/SearchField';
import { SidePanel } from '@/ui/SidePanel/SidePanel';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { TeamPills } from '@/ui/TeamPill/TeamPill';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { useFormatters } from '@/format/useFormatters';
import { formFailure } from './feedback';
import { OwnerImpactDialog } from './OwnerImpactDialog';
import {
  fetchLatestPerson,
  useCheckPersonImpact,
  useCreatePerson,
  usePerson,
  useUpdatePerson,
} from './api';
import type { ImpactWarning, Person, PersonDetails, PersonUpdate } from './api';
import { RecordLine } from './RecordLine';
import { removedIds } from './removed-ids';
import { TeamField } from './TeamField';
import { PersonRolesFact } from './access/PersonRolesFact';
import styles from '@/ui/settings-table.module.css';
import { personStatusItem } from './person-actions';
import type { PersonActions } from './person-actions';
import { approvesInText, ownsText } from './texts';
import panel from './panels.module.css';
import { CanChange } from '@/ui/CanChange/CanChange';

/** Above this many teams, the details offer "Search teams". */
const SEARCH_ABOVE = 10;

export interface PersonDetailsPanelProps {
  personId: string;
  actions: PersonActions;
  onClose: () => void;
}

/**
 * A person's details (US1-3): avatar, status, e-mail, job title, teams as links, what they own and
 * where they approve, and who created and last changed the record. The one panel footer of
 * Settings: **Edit** (primary) and ⋯ with Deactivate or Reactivate — for yourself, Deactivate stays
 * in the menu with the reason it can't be used (FR-006).
 */
export function PersonDetailsPanel({ personId, actions, onClose }: PersonDetailsPanelProps) {
  const person = usePerson(personId);
  const user = useCurrentUser();

  if (person.data) {
    const data = person.data;
    const isYou = data.id === user.id;
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
              <ActionsMenu label={data.name} items={[personStatusItem(data, isYou, actions)]} />
            </>
          </CanChange>
        }
      >
        <PersonDetailsBody person={data} />
      </SidePanel>
    );
  }

  const notFound = person.error instanceof ApiError && person.error.kind === 'notFound';
  return (
    <SidePanel
      open
      onOpenChange={(open) => !open && onClose()}
      title="Person"
      footer={<Button onClick={onClose}>Back to People</Button>}
    >
      {person.isPending && <Skeleton lines={6} height="18px" />}
      {notFound && <p className={styles.gone}>{person.error?.message}</p>}
      {person.isError && !notFound && (
        <ErrorState error={person.error} onRetry={() => void person.refetch()} />
      )}
    </SidePanel>
  );
}

function PersonDetailsBody({ person }: { person: PersonDetails }) {
  const [search, setSearch] = useState('');
  const query = search.trim().toLowerCase();
  const teams = person.teams.filter((team) => team.name.toLowerCase().includes(query));

  return (
    <div className={styles.details}>
      <div className={styles.statusRow}>
        <Avatar initials={person.initials} size="lg" />
        <StatusChip tone={person.active ? 'positive' : 'neutral'}>
          {person.active ? 'Active' : 'Inactive'}
        </StatusChip>
      </div>
      <dl className={styles.facts}>
        <div>
          <dt>E-mail</dt>
          <dd>{person.email}</dd>
        </div>
        <div>
          <dt>Job title</dt>
          <dd>{person.jobTitle || '—'}</dd>
        </div>
        <div>
          <dt>Teams</dt>
          <dd className={panel.section}>
            {person.teams.length === 0 ? (
              <span className={styles.muted}>
                Not in any team yet. A person needs a team to own a plan.
              </span>
            ) : (
              <>
                {person.teams.length > SEARCH_ABOVE && (
                  <SearchField
                    label="Search teams"
                    placeholder="Search teams"
                    value={search}
                    onChange={setSearch}
                  />
                )}
                <TeamPills teams={teams} linked />
              </>
            )}
          </dd>
        </div>
        <div>
          <dt>Owns</dt>
          <dd>{ownsText(person.usage)}</dd>
        </div>
        <div>
          <dt>Approves in</dt>
          <dd>{approvesInText(person.usage)}</dd>
        </div>
        <PersonRolesFact personId={person.id} />
      </dl>
      <RecordLine record={person} />
    </div>
  );
}

export interface PersonFormPanelProps {
  /** The person to edit; without one, the panel adds a new person. */
  person?: Person | PersonDetails;
  onClose: () => void;
}

/**
 * "New person" or "Edit person" (US1-1, US1-2, US1-4): name, e-mail, job title and teams. When
 * teams were removed, the API's impact check comes first and its warnings are confirmed in
 * OwnerImpactDialog (plan "Save flow").
 */
export function PersonFormPanel({ person, onClose }: PersonFormPanelProps) {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const createPerson = useCreatePerson();
  const updatePerson = useUpdatePerson();
  const checkImpact = useCheckPersonImpact();
  // The edit-form rule of feature 001: the person is copied into the form once, when it opens.
  const [start] = useState(() => ({
    name: person?.name ?? '',
    email: person?.email ?? '',
    jobTitle: person?.jobTitle ?? '',
    teamIds: person?.teams.map((team) => team.id) ?? [],
  }));
  const [name, setName] = useState(start.name);
  const [email, setEmail] = useState(start.email);
  const [jobTitle, setJobTitle] = useState(start.jobTitle);
  const [teamIds, setTeamIds] = useState(start.teamIds);
  const [version, setVersion] = useState(person?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const [warnings, setWarnings] = useState<ImpactWarning[]>([]);

  const isDirty =
    name !== start.name ||
    email !== start.email ||
    jobTitle !== start.jobTitle ||
    teamIds.join() !== start.teamIds.join();
  // Only the fields that changed, and the version (D-203, D-210): after a conflict, Save again
  // replaces only what this person changed.
  const body: PersonUpdate = {
    ...(name !== start.name ? { name } : {}),
    ...(email !== start.email ? { email } : {}),
    ...(jobTitle !== start.jobTitle ? { jobTitle } : {}),
    ...(teamIds.join() !== start.teamIds.join() ? { teamIds } : {}),
    version,
  };

  function done(saved: PersonDetails, verb: 'created' | 'saved') {
    show({
      message: `Person '${saved.name}' ${verb}`,
      action: {
        label: 'View',
        altText: 'Open them from the People list',
        onClick: () => void navigate(`/settings/people/${encodeURIComponent(saved.id)}`),
      },
    });
    onClose();
  }

  async function fail(error: unknown) {
    const result = formFailure(error, 'person', dateFormat);
    setProblems(result.problems);
    setFailure(result.failure);
    if (result.conflict && person) {
      // Typed values stay; the form takes the version with the other person's change, so Save
      // again replaces it on purpose.
      const latest = await fetchLatestPerson(queryClient, person.id);
      if (latest) {
        setVersion(latest.version);
      }
    }
  }

  async function save() {
    setSaving(true);
    try {
      if (!person) {
        done(await createPerson.mutateAsync({ name, email, jobTitle, teamIds }), 'created');
        return;
      }
      if (removedIds(start.teamIds, teamIds).length > 0) {
        const impact = await checkImpact.mutateAsync({ id: person.id, body });
        if (impact.warnings.length > 0) {
          setProblems([]);
          setFailure(null);
          setWarnings(impact.warnings);
          return;
        }
      }
      done(await updatePerson.mutateAsync({ id: person.id, body }), 'saved');
    } catch (error) {
      await fail(error);
    } finally {
      setSaving(false);
    }
  }

  async function saveAfterWarnings() {
    if (!person) {
      return;
    }
    setSaving(true);
    try {
      const saved = await updatePerson.mutateAsync({ id: person.id, body });
      setWarnings([]);
      done(saved, 'saved');
    } catch (error) {
      setWarnings([]);
      await fail(error);
    } finally {
      setSaving(false);
    }
  }

  return (
    <>
      <FormPanel
        title={person ? 'Edit person' : 'New person'}
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
          autoComplete="off"
          value={name}
          onChange={setName}
          error={problemOf(problems, 'name')}
        />
        <TextField
          label="E-mail"
          name="email"
          type="email"
          marking="Required"
          autoComplete="off"
          hint={
            person
              ? 'Their sign-in name — the next sign-in uses the new e-mail.'
              : 'Their sign-in name. They can sign in straight away with the demo password.'
          }
          value={email}
          onChange={setEmail}
          error={problemOf(problems, 'email')}
        />
        <TextField
          label="Job title"
          name="jobTitle"
          autoComplete="off"
          value={jobTitle}
          onChange={setJobTitle}
          error={problemOf(problems, 'jobTitle')}
        />
        <TeamField
          label="Teams"
          name="teamIds"
          hint="Only active teams can be added."
          value={teamIds}
          onChange={setTeamIds}
          error={problemOf(problems, 'teamIds')}
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
