import { useQueryClient } from '@tanstack/react-query';
import { useId, useState } from 'react';
import { useNavigate } from 'react-router';
import { ApiError } from '@/api/errors';
import { useCurrentUser } from '@/app/useSession';
import {
  Avatar,
  Button,
  ConfirmDialog,
  ErrorState,
  FormPanel,
  problemOf,
  SearchField,
  SidePanel,
  Skeleton,
  StatusChip,
  TeamPills,
  TextField,
  useToast,
} from '@/components';
import type { FieldProblem, SaveFailure } from '@/components';
import { useFormatters } from '@/lib/useFormatters';
import { formFailure } from './feedback';
import { OwnerImpactDialog } from './OwnerImpactDialog';
import {
  fetchLatestPerson,
  useCheckPersonImpact,
  useCreatePerson,
  usePerson,
  useUpdatePerson,
} from './peopleApi';
import type { ImpactWarning, Person, PersonDetails, PersonUpdate } from './peopleApi';
import { RecordLine } from './RecordLine';
import { removedIds } from './removedIds';
import { TeamField } from './TeamField';
import { DEACTIVATE_TIP, deactivateSentences, SELF_DEACTIVATION, usageText } from './texts';
import styles from './panels.module.css';

/** Above this many teams, the details offer "Search teams". */
const SEARCH_ABOVE = 10;

/** What the details panel lets people do with the person. */
export interface PersonActions {
  edit: (person: Person | PersonDetails) => void;
  deactivate: (person: Person | PersonDetails) => void;
  reactivate: (person: Person | PersonDetails) => void;
}

export interface PersonDetailsPanelProps {
  personId: string;
  actions: PersonActions;
  /** A status change is on its way: its button shows it. */
  statusBusy: boolean;
  onClose: () => void;
}

/**
 * A person's details (US1-3): avatar, status, e-mail, job title, teams as links, what they own and
 * approve, and who created and last changed the record; Edit and Deactivate or Reactivate. For
 * yourself, Deactivate stays visible with the reason it can't be used (FR-006).
 */
export function PersonDetailsPanel({
  personId,
  actions,
  statusBusy,
  onClose,
}: PersonDetailsPanelProps) {
  const person = usePerson(personId);
  const user = useCurrentUser();
  const reasonId = useId();

  if (person.data) {
    const data = person.data;
    const isYou = data.id === user.id;
    return (
      <SidePanel
        open
        onOpenChange={(open) => !open && onClose()}
        title={data.name}
        footer={
          <div className={styles.footerActions}>
            <Button variant="primary" onClick={() => actions.edit(data)}>
              Edit
            </Button>
            {data.active ? (
              <Button
                aria-disabled={isYou || undefined}
                aria-describedby={reasonId}
                aria-busy={statusBusy || undefined}
                onClick={() => !isYou && actions.deactivate(data)}
              >
                Deactivate
              </Button>
            ) : (
              <Button aria-busy={statusBusy || undefined} onClick={() => actions.reactivate(data)}>
                Reactivate
              </Button>
            )}
            {data.active && (
              // Always visible: why you can't deactivate yourself, or why there is no Delete.
              <p id={reasonId} className={styles.reasonLine}>
                {isYou ? SELF_DEACTIVATION : DEACTIVATE_TIP}
              </p>
            )}
          </div>
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
      footer={
        <Button variant="primary" onClick={onClose}>
          Close
        </Button>
      }
    >
      {person.isPending && <Skeleton lines={6} height="18px" />}
      {notFound && <p className={styles.notFound}>{person.error?.message}</p>}
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
      <div className={styles.top}>
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
          <dd className={styles.section}>
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
          <dt>Owns and approves</dt>
          <dd>{usageText(person.usage)}</dd>
        </div>
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
  const body: PersonUpdate = { name, email, jobTitle, teamIds, version };

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

export interface DeactivateDialogProps {
  person: Person | PersonDetails;
  /** Deactivates the person from the version shown; resolves when done. */
  deactivate: (person: PersonDetails) => Promise<void>;
  onClose: () => void;
}

/**
 * "Deactivate <name>?" (US1-5, FR-005): loads what the person owns and approves first — a grey
 * line stands in for it meanwhile, and Deactivate person waits for it — then says who keeps their
 * name. Cancel has the focus.
 */
export function DeactivateDialog({ person, deactivate, onClose }: DeactivateDialogProps) {
  const details = usePerson(person.id);
  const [busy, setBusy] = useState(false);
  const [first, ...rest] = deactivateSentences({
    name: person.name,
    usage: details.data?.usage ?? [],
  });

  async function confirm() {
    // Ignored until the person's usage has arrived, and while the change is on its way.
    if (!details.data || busy) {
      return;
    }
    setBusy(true);
    await deactivate(details.data);
    onClose();
  }

  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={`Deactivate ${person.name}?`}
      confirmLabel="Deactivate person"
      tone="danger"
      busy={busy}
      onConfirm={() => void confirm()}
    >
      <p>{first}</p>
      {details.isPending && <Skeleton height="14px" />}
      {details.isError && (
        <ErrorState error={details.error} onRetry={() => void details.refetch()} />
      )}
      {details.data && rest.map((sentence) => <p key={sentence}>{sentence}</p>)}
    </ConfirmDialog>
  );
}
