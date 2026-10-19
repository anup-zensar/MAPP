import { Plus, Users } from 'lucide-react';
import { useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router';
import { useCurrentUser } from '@/app/useSession';
import {
  ActionsMenu,
  Breadcrumb,
  Button,
  Card,
  EmptyState,
  ErrorState,
  PageHeader,
  PersonLabel,
  SearchField,
  SegmentedFilter,
  Skeleton,
  StatusChip,
  TeamPills,
  useToast,
} from '@/components';
import type { ActionsMenuItem } from '@/components';
import { usePageTitle } from '@/lib/usePageTitle';
import { useShowFailure } from './feedback';
import { usePeople, useSetPersonStatus } from './peopleApi';
import type { Person, PersonDetails } from './peopleApi';
import { DeactivateDialog, PersonDetailsPanel, PersonFormPanel } from './PersonPanels';
import type { PersonActions } from './PersonPanels';
import { DEACTIVATE_TIP, peopleCountText, SELF_DEACTIVATION } from './texts';
import styles from './lists.module.css';

type StatusFilter = 'all' | 'active' | 'inactive';

const STATUS_OPTIONS: { value: StatusFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'active', label: 'Active' },
  { value: 'inactive', label: 'Inactive' },
];

/** The form panel that is open. */
type OpenForm = { kind: 'new' } | { kind: 'edit'; person: Person | PersonDetails };

/**
 * Settings › People (feature 003, US1, FR-001 – FR-009): everyone who can sign in, with e-mail, job
 * title, teams and status; search, status filter, the details panel at /settings/people/:personId,
 * New person, Edit, Deactivate and Reactivate. People are never deleted (FR-008).
 */
export function PeoplePage() {
  usePageTitle('People');
  const { personId } = useParams();
  const navigate = useNavigate();
  const user = useCurrentUser();
  const { show } = useToast();
  const showFailure = useShowFailure('person');
  const setPersonStatus = useSetPersonStatus();
  const [form, setForm] = useState<OpenForm | null>(null);
  const [deactivating, setDeactivating] = useState<Person | PersonDetails | null>(null);

  async function setStatus(person: Person | PersonDetails, active: boolean) {
    // One status change at a time; another click meanwhile is ignored.
    if (setPersonStatus.isPending) {
      return;
    }
    try {
      const saved = await setPersonStatus.mutateAsync({
        id: person.id,
        active,
        version: person.version,
      });
      show({ message: `Person '${saved.name}' ${active ? 'reactivated' : 'deactivated'}` });
    } catch (error) {
      showFailure(error);
    }
  }

  const actions: PersonActions = {
    edit: (person) => setForm({ kind: 'edit', person }),
    deactivate: (person) =>
      person.id === user.id ? show({ message: SELF_DEACTIVATION }) : setDeactivating(person),
    reactivate: (person) => void setStatus(person, true),
  };

  return (
    <>
      <Breadcrumb items={[{ label: 'Settings', to: '/settings' }, { label: 'People' }]} />
      <PageHeader
        eyebrow="Settings"
        title="People"
        description="Everyone who can sign in to NorthStar, with their job title and teams."
        actions={
          <Button variant="primary" icon={<Plus />} onClick={() => setForm({ kind: 'new' })}>
            New person
          </Button>
        }
      />
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
          statusBusy={setPersonStatus.isPending}
          onClose={() => void navigate('/settings/people')}
        />
      )}
      {form && (
        <PersonFormPanel
          person={form.kind === 'edit' ? form.person : undefined}
          onClose={() => setForm(null)}
        />
      )}
      {deactivating && (
        <DeactivateDialog
          person={deactivating}
          deactivate={(person) => setStatus(person, false)}
          onClose={() => setDeactivating(null)}
        />
      )}
    </>
  );
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
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState<StatusFilter>('all');

  const all = people.data?.items ?? [];
  const query = search.trim().toLowerCase();
  const shown = all.filter(
    (person) =>
      (status === 'all' || person.active === (status === 'active')) &&
      `${person.name} ${person.email} ${person.jobTitle}`.toLowerCase().includes(query),
  );
  const clearFilters = () => {
    setSearch('');
    setStatus('all');
  };

  return (
    <Card>
      <div className={styles.toolbar}>
        <div className={styles.search}>
          <SearchField
            label="Search name, e-mail or job title"
            placeholder="Search name, e-mail or job title"
            value={search}
            onChange={setSearch}
          />
        </div>
        <SegmentedFilter
          label="Status"
          options={STATUS_OPTIONS}
          value={status}
          onChange={setStatus}
        />
        {people.data && <p className={styles.count}>{peopleCountText(shown.length, all.length)}</p>}
      </div>

      {people.isPending && <Skeleton lines={5} height="28px" />}
      {people.isError && <ErrorState error={people.error} onRetry={() => void people.refetch()} />}
      {people.data && all.length === 0 && (
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
      )}
      {people.data && all.length > 0 && shown.length === 0 && (
        <EmptyState
          title="Nothing matches these filters"
          action={<Button onClick={clearFilters}>Clear filters</Button>}
        />
      )}
      {shown.length > 0 && (
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
  // There is no Delete: Deactivate sits in its place with the reason (FR-008).
  const statusItem: ActionsMenuItem = person.active
    ? {
        label: 'Deactivate',
        hint: isYou ? SELF_DEACTIVATION : DEACTIVATE_TIP,
        separatorBefore: true,
        onSelect: () => actions.deactivate(person),
      }
    : { label: 'Reactivate', separatorBefore: true, onSelect: () => actions.reactivate(person) };

  return (
    <tr>
      <th scope="row">
        <Link className={styles.open} to={`/settings/people/${encodeURIComponent(person.id)}`}>
          <PersonLabel person={person} you={isYou} />
        </Link>
      </th>
      <td className={styles.secondary}>{person.email}</td>
      <td>{person.jobTitle || '—'}</td>
      <td>
        {person.teams.length > 0 ? (
          <TeamPills teams={person.teams} />
        ) : (
          <span className={styles.secondary}>No team</span>
        )}
      </td>
      <td>
        <StatusChip tone={person.active ? 'positive' : 'neutral'}>
          {person.active ? 'Active' : 'Inactive'}
        </StatusChip>
      </td>
      <td className={styles.actionsCell}>
        <ActionsMenu
          label={person.name}
          items={[
            { label: 'View', onSelect: () => onOpen(person) },
            { label: 'Edit', onSelect: () => actions.edit(person) },
            statusItem,
          ]}
        />
      </td>
    </tr>
  );
}
