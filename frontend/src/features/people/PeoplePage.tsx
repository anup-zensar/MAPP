import { useQueryClient } from '@tanstack/react-query';
import { Plus, Users } from 'lucide-react';
import { useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router';
import { useCurrentUser } from '@/app/useSession';
import {
  ActionsMenu,
  Button,
  Card,
  EmptyState,
  ErrorState,
  PageHeader,
  PersonLabel,
  Skeleton,
  StatusChip,
  TeamPills,
  useToast,
} from '@/components';
import { ListToolbar } from '@/features/settings/shared/ListToolbar';
import styles from '@/features/settings/shared/SettingsTable.module.css';
import lists from './lists.module.css';
import { filterRecords, useListFilters } from '@/features/settings/shared/useListFilters';
import { useStatusFlow } from '@/features/settings/shared/useStatusFlow';
import { usePageTitle } from '@/lib/usePageTitle';
import { useShowFailure } from './feedback';
import { fetchLatestPerson, usePeople, useSetPersonStatus } from './peopleApi';
import type { Person, PersonDetails } from './peopleApi';
import { personStatusItem } from './personActions';
import type { PersonActions } from './personActions';
import { PersonDetailsPanel, PersonFormPanel } from './PersonPanels';
import { deactivateSentences, isInUse, SELF_DEACTIVATION } from './texts';
import { CanChange, SettingsViewOnlyNote } from '../settings/shared/CanChange';
import { useCanChange } from '../settings/shared/useCanChange';

const PERSON_NOUN = { one: 'person', many: 'people' };

/** The form panel that is open. */
type OpenForm = { kind: 'new' } | { kind: 'edit'; person: Person | PersonDetails };

/**
 * Settings › People (feature 003, US1, FR-001 – FR-009): everyone who can sign in, with e-mail, job
 * title, teams and status; the Settings list toolbar (search, Active · Inactive · All, "6 of 6
 * people"), the details panel at /settings/people/:personId, New person, Edit, Deactivate and
 * Reactivate. People are never deleted (FR-008).
 */
export function PeoplePage() {
  usePageTitle('People');
  const { personId } = useParams();
  const navigate = useNavigate();
  const user = useCurrentUser();
  const { show } = useToast();
  const status = usePersonStatus();
  const [form, setForm] = useState<OpenForm | null>(null);

  const actions: PersonActions = {
    edit: (person) => setForm({ kind: 'edit', person }),
    deactivate: (person) =>
      person.id === user.id ? show({ message: SELF_DEACTIVATION }) : status.setInactive(person),
    reactivate: (person) => status.reactivate(person),
  };

  return (
    <>
      <PageHeader
        breadcrumb={[{ label: 'Settings', to: '/settings' }, { label: 'People' }]}
        title="People"
        description="Everyone who can sign in to MAPP, with their job title and teams."
        actions={
          <CanChange permission="people.manage">
            <Button variant="primary" icon={<Plus />} onClick={() => setForm({ kind: 'new' })}>
              New person
            </Button>
          </CanChange>
        }
      />
      <SettingsViewOnlyNote permission="people.manage" />
      <PeopleCard
        currentUserId={user.id}
        onNew={() => setForm({ kind: 'new' })}
        onOpen={(person) => void navigate(`/settings/people/${encodeURIComponent(person.id)}`)}
        actions={actions}
      />

      {personId && !form && (
        <PersonDetailsPanel
          personId={personId}
          actions={actions}
          onClose={() => void navigate('/settings/people')}
        />
      )}
      {form && (
        <PersonFormPanel
          person={form.kind === 'edit' ? form.person : undefined}
          onClose={() => setForm(null)}
        />
      )}
      {status.dialog}
    </>
  );
}

/**
 * Deactivate and Reactivate with the one set-inactive behaviour (design system §6.7): a person who
 * owns or approves something is asked about first, with what they own and approve (FR-005);
 * anyone else is deactivated at once. Both offer Undo.
 */
function usePersonStatus() {
  const queryClient = useQueryClient();
  const setPersonStatus = useSetPersonStatus();
  const showFailure = useShowFailure('person');

  return useStatusFlow<Person | PersonDetails>({
    words: { title: 'Person', noun: 'person', person: true },
    usage: async (person) => {
      // What they own and approve, as stored now.
      const details = await fetchLatestPerson(queryClient, person.id);
      const usage = details?.usage ?? [];
      return isInUse(usage) ? deactivateSentences({ name: person.name, usage }) : [];
    },
    effect: () => [],
    request: (person, active, version) =>
      setPersonStatus.mutateAsync({ id: person.id, active, version }),
    onFailure: showFailure,
  });
}

interface PeopleCardProps {
  currentUserId: string;
  onNew: () => void;
  onOpen: (person: Person) => void;
  actions: PersonActions;
}

/** The toolbar and the table, with the list's loading, empty, no-match and error states. */
function PeopleCard({ currentUserId, onNew, onOpen, actions }: PeopleCardProps) {
  const people = usePeople();
  const filters = useListFilters();

  if (!people.data) {
    return (
      <Card>
        {people.isError ? (
          <ErrorState error={people.error} onRetry={() => void people.refetch()} />
        ) : (
          <Skeleton lines={5} height="28px" />
        )}
      </Card>
    );
  }

  const all = people.data.items;
  if (all.length === 0) {
    return (
      <Card>
        <EmptyState
          icon={<Users />}
          title="No people yet — add the first person"
          action={
            // Secondary: the page header already has the one primary button.
            <Button icon={<Plus />} onClick={onNew}>
              New person
            </Button>
          }
        />
      </Card>
    );
  }

  const { shown, counts } = filterRecords(
    all,
    filters,
    (person) => `${person.name} ${person.email} ${person.jobTitle}`,
  );

  return (
    <Card>
      <ListToolbar
        search={{
          label: 'Search name, e-mail or job title',
          placeholder: 'Search name, e-mail or job title',
          value: filters.q,
          onChange: filters.setQ,
        }}
        status={{ value: filters.status, onChange: filters.setStatus, counts }}
        count={{ shown: shown.length, total: all.length, noun: PERSON_NOUN }}
      />
      {shown.length === 0 ? (
        <div className={styles.noMatch}>
          <p>Nothing matches these filters</p>
          <Button onClick={filters.clear}>Clear filters</Button>
        </div>
      ) : (
        // On narrow screens the table scrolls sideways inside its card (design system §2).
        <div className={styles.tableScroll}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th scope="col">Name</th>
                <th scope="col">E-mail</th>
                <th scope="col">Job title</th>
                <th scope="col">Teams</th>
                <th scope="col">Status</th>
                <th scope="col">
                  <span className="visually-hidden">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {shown.map((person) => (
                <PersonRow
                  key={person.id}
                  person={person}
                  isYou={person.id === currentUserId}
                  onOpen={onOpen}
                  actions={actions}
                />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}

interface PersonRowProps {
  person: Person;
  isYou: boolean;
  onOpen: (person: Person) => void;
  actions: PersonActions;
}

function PersonRow({ person, isYou, onOpen, actions }: PersonRowProps) {
  const canChange = useCanChange('people.manage');
  return (
    <tr>
      <th scope="row" className={`${styles.nameCell} ${styles.nowrap}`}>
        <Link
          className={styles.personLink}
          to={`/settings/people/${encodeURIComponent(person.id)}`}
        >
          <PersonLabel person={person} you={isYou} />
        </Link>
      </th>
      <td className={`${styles.muted} ${styles.nowrap}`}>{person.email}</td>
      <td className={styles.nowrap}>{person.jobTitle || '—'}</td>
      <td className={lists.teamsCell}>
        {person.teams.length > 0 ? (
          <TeamPills teams={person.teams} />
        ) : (
          <span className={styles.muted}>No team</span>
        )}
      </td>
      <td className={styles.nowrap}>
        <StatusChip tone={person.active ? 'positive' : 'neutral'}>
          {person.active ? 'Active' : 'Inactive'}
        </StatusChip>
      </td>
      <td className={styles.actionsCell}>
        <ActionsMenu
          label={person.name}
          items={[
            { label: 'View', onSelect: () => onOpen(person) },
            ...(canChange
              ? [
                  { label: 'Edit', onSelect: () => actions.edit(person) },
                  personStatusItem(person, isYou, actions),
                ]
              : []),
          ]}
        />
      </td>
    </tr>
  );
}
