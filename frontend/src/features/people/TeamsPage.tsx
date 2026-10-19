import { Plus, UsersRound } from 'lucide-react';
import { useState } from 'react';
import { useNavigate, useParams } from 'react-router';
import {
  ActionsMenu,
  Avatar,
  Breadcrumb,
  Button,
  Card,
  EmptyState,
  ErrorState,
  PageHeader,
  SearchField,
  SegmentedFilter,
  Skeleton,
  StatusChip,
  TeamPill,
  useToast,
} from '@/components';
import { usePageTitle } from '@/lib/usePageTitle';
import { useShowFailure } from './feedback';
import { useSetTeamStatus, useTeams } from './peopleApi';
import type { Team } from './peopleApi';
import { DeleteTeamDialog, TeamDetailsPanel, TeamFormPanel } from './TeamPanels';
import type { TeamActions } from './TeamPanels';
import { membersText, plansText, teamsCountText } from './texts';
import styles from './lists.module.css';

type StatusFilter = 'all' | 'active' | 'inactive';

const STATUS_OPTIONS: { value: StatusFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'active', label: 'Active' },
  { value: 'inactive', label: 'Inactive' },
];

/** How many member avatars a row shows before the count. */
const AVATARS_SHOWN = 5;

/** The form panel that is open. */
type OpenForm = { kind: 'new' } | { kind: 'edit'; team: Team };

/**
 * Settings › Teams (feature 003, US2, FR-010 – FR-015): every team with its description, members,
 * the plans using it and its status; search, status filter, the details panel at
 * /settings/teams/:teamId, New team, Edit, Set team inactive / Reactivate and Delete with Undo.
 */
export function TeamsPage() {
  usePageTitle('Teams');
  const { teamId } = useParams();
  const navigate = useNavigate();
  const { show } = useToast();
  const showFailure = useShowFailure('team');
  const setTeamStatus = useSetTeamStatus();
  const [form, setForm] = useState<OpenForm | null>(null);
  const [deleting, setDeleting] = useState<Team | null>(null);

  const openDetails = (team: Team) =>
    void navigate(`/settings/teams/${encodeURIComponent(team.id)}`);
  const closeDetails = () => void navigate('/settings/teams');

  async function setStatus(team: Team, active: boolean) {
    // One status change at a time; another click meanwhile is ignored.
    if (setTeamStatus.isPending) {
      return;
    }
    try {
      const saved = await setTeamStatus.mutateAsync({ id: team.id, active, version: team.version });
      show({ message: `Team '${saved.name}' ${active ? 'reactivated' : 'set inactive'}` });
    } catch (error) {
      showFailure(error);
    }
  }

  const actions: TeamActions = {
    edit: (team) => setForm({ kind: 'edit', team }),
    setStatus: (team, active) => void setStatus(team, active),
    remove: setDeleting,
  };

  return (
    <>
      <Breadcrumb items={[{ label: 'Settings', to: '/settings' }, { label: 'Teams' }]} />
      <PageHeader
        eyebrow="Settings"
        title="Teams"
        description="Groups of people that plans are assigned to. A plan's owner must belong to one of its teams."
        actions={
          <Button variant="primary" icon={<Plus />} onClick={() => setForm({ kind: 'new' })}>
            New team
          </Button>
        }
      />
      <TeamsCard onNew={() => setForm({ kind: 'new' })} onOpen={openDetails} actions={actions} />

      {teamId && !form && (
        <TeamDetailsPanel
          teamId={teamId}
          actions={actions}
          statusBusy={setTeamStatus.isPending}
          onClose={closeDetails}
        />
      )}
      {form && (
        <TeamFormPanel
          team={form.kind === 'edit' ? form.team : undefined}
          onClose={() => setForm(null)}
        />
      )}
      {deleting && (
        <DeleteTeamDialog
          team={deleting}
          setInactive={(team) => setStatus(team, false)}
          onDeleted={(team) => {
            if (team.id === teamId) {
              closeDetails();
            }
          }}
          onClose={() => setDeleting(null)}
        />
      )}
    </>
  );
}

interface TeamsCardProps {
  onNew: () => void;
  onOpen: (team: Team) => void;
  actions: TeamActions;
}

/** The toolbar and the table, with the list's loading, empty, no-match and error states. */
function TeamsCard({ onNew, onOpen, actions }: TeamsCardProps) {
  const teams = useTeams();
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState<StatusFilter>('all');

  const all = teams.data?.items ?? [];
  const query = search.trim().toLowerCase();
  const shown = all.filter(
    (team) =>
      (status === 'all' || team.active === (status === 'active')) &&
      `${team.name} ${team.description}`.toLowerCase().includes(query),
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
            label="Search team name or description"
            placeholder="Search team name or description"
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
        {teams.data && <p className={styles.count}>{teamsCountText(shown.length, all.length)}</p>}
      </div>

      {teams.isPending && <Skeleton lines={5} height="28px" />}
      {teams.isError && <ErrorState error={teams.error} onRetry={() => void teams.refetch()} />}
      {teams.data && all.length === 0 && (
        <EmptyState
          icon={<UsersRound />}
          title="No teams yet — create your first team"
          action={
            // Secondary: the page header already has the one primary button.
            <Button icon={<Plus />} onClick={onNew}>
              New team
            </Button>
          }
        />
      )}
      {teams.data && all.length > 0 && shown.length === 0 && (
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
                <th scope="col">Team</th>
                <th scope="col">Description</th>
                <th scope="col">Members</th>
                <th scope="col">Used by</th>
                <th scope="col">Status</th>
                <th scope="col">
                  <span className="visually-hidden">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {shown.map((team) => (
                <TeamRow key={team.id} team={team} onOpen={onOpen} actions={actions} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}

interface TeamRowProps {
  team: Team;
  onOpen: (team: Team) => void;
  actions: TeamActions;
}

function TeamRow({ team, onOpen, actions }: TeamRowProps) {
  return (
    <tr>
      <th scope="row">
        <TeamPill team={team} to={`/settings/teams/${encodeURIComponent(team.id)}`} />
      </th>
      <td>
        <span className={styles.clamp}>{team.description || '—'}</span>
      </td>
      <td>
        <span className={styles.members}>
          {team.members.length > 0 && (
            <ul className={styles.avatars} aria-hidden="true">
              {team.members.slice(0, AVATARS_SHOWN).map((member) => (
                <li key={member.id}>
                  <Avatar initials={member.initials} size="sm" />
                </li>
              ))}
            </ul>
          )}
          {membersText(team.members.length)}
        </span>
      </td>
      <td className={styles.nowrap}>{plansText(team.plans.length)}</td>
      <td>
        <StatusChip tone={team.active ? 'positive' : 'neutral'}>
          {team.active ? 'Active' : 'Inactive'}
        </StatusChip>
      </td>
      <td className={styles.actionsCell}>
        <ActionsMenu
          label={team.name}
          items={[
            { label: 'View', onSelect: () => onOpen(team) },
            { label: 'Edit', onSelect: () => actions.edit(team) },
            {
              label: team.active ? 'Set team inactive' : 'Reactivate',
              onSelect: () => actions.setStatus(team, !team.active),
            },
            {
              label: 'Delete',
              tone: 'danger',
              separatorBefore: true,
              onSelect: () => actions.remove(team),
            },
          ]}
        />
      </td>
    </tr>
  );
}
