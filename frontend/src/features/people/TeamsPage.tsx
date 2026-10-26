import { Plus, UsersRound } from 'lucide-react';
import { useState } from 'react';
import { useNavigate, useParams } from 'react-router';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { Avatar } from '@/ui/Avatar/Avatar';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { TeamPill } from '@/ui/TeamPill/TeamPill';
import { ListToolbar } from '@/ui/ListToolbar/ListToolbar';
import styles from '@/ui/settings-table.module.css';
import { filterRecords, useListFilters } from '@/ui/useListFilters';
import { useStatusFlow } from '@/ui/useStatusFlow';
import { formatWhole } from '@/format/decimal';
import { usePageTitle } from '@/ui/usePageTitle';
import { CanChange, SettingsViewOnlyNote } from '@/ui/CanChange/CanChange';
import { useCanChange } from '@/ui/useCanChange';
import { useShowFailure } from './feedback';
import { useSetTeamStatus, useTeams } from './api';
import type { Team } from './api';
import { teamMoreItems } from './team-actions';
import type { TeamActions } from './team-actions';
import { DeleteTeamDialog, TeamDetailsPanel, TeamFormPanel } from './TeamPanels';
import { usedByText } from './texts';
import page from './lists.module.css';

const TEAM_NOUN = { one: 'team', many: 'teams' };

/** How many member avatars a row shows before the count. */
const AVATARS_SHOWN = 5;

/** What setting a team inactive does, said before it applies. */
const TEAM_INACTIVE_EFFECT =
  "New plans can't choose an inactive team; plans using it keep it, shown as '(inactive)'.";

/** The form panel that is open. */
type OpenForm = { kind: 'new' } | { kind: 'edit'; team: Team };

/**
 * Settings › Teams (feature 003, US2, FR-010 – FR-015): every team with its description, members,
 * the plans using it and its status; the Settings list toolbar, the details panel at
 * /settings/teams/:teamId, New team, Edit, Set team inactive / Reactivate and Delete with Undo.
 */
export function TeamsPage() {
  usePageTitle('Teams');
  const { teamId } = useParams();
  const navigate = useNavigate();
  const status = useTeamStatus();
  const [form, setForm] = useState<OpenForm | null>(null);
  const [deleting, setDeleting] = useState<Team | null>(null);

  const openDetails = (team: Team) =>
    void navigate(`/settings/teams/${encodeURIComponent(team.id)}`);
  const closeDetails = () => void navigate('/settings/teams');

  const actions: TeamActions = {
    edit: (team) => setForm({ kind: 'edit', team }),
    setStatus: (team, active) => (active ? status.reactivate(team) : status.setInactive(team)),
    remove: setDeleting,
  };

  return (
    <>
      <PageHeader
        breadcrumb={[{ label: 'Settings', to: '/settings' }, { label: 'Teams' }]}
        title="Teams"
        description="Groups of people that plans are assigned to. A plan's owner must belong to one of its teams."
        actions={
          <CanChange permission="people.manage">
            <Button variant="primary" icon={<Plus />} onClick={() => setForm({ kind: 'new' })}>
              New team
            </Button>
          </CanChange>
        }
      />
      <SettingsViewOnlyNote permission="people.manage" />
      <TeamsCard onNew={() => setForm({ kind: 'new' })} onOpen={openDetails} actions={actions} />

      {teamId && !form && (
        <TeamDetailsPanel teamId={teamId} actions={actions} onClose={closeDetails} />
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
          setInactive={(team) => status.applyInactive(team).then(() => undefined)}
          onDeleted={(team) => {
            if (team.id === teamId) {
              closeDetails();
            }
          }}
          onClose={() => setDeleting(null)}
        />
      )}
      {status.dialog}
    </>
  );
}

/**
 * Set team inactive and Reactivate with the one set-inactive behaviour (design system §6.7): a
 * team that plans use asks first and says which; an unused one is set inactive at once. Both offer
 * Undo.
 */
function useTeamStatus() {
  const setTeamStatus = useSetTeamStatus();
  const showFailure = useShowFailure('team');

  return useStatusFlow<Team>({
    words: { title: 'Team', noun: 'team' },
    usage: (team) =>
      team.plans.length > 0
        ? [`${usedByText(team.plans.length)}: ${team.plans.map((plan) => plan.name).join(', ')}.`]
        : [],
    effect: () => [TEAM_INACTIVE_EFFECT],
    request: (team, active, version) => setTeamStatus.mutateAsync({ id: team.id, active, version }),
    onFailure: showFailure,
  });
}

interface TeamsCardProps {
  onNew: () => void;
  onOpen: (team: Team) => void;
  actions: TeamActions;
}

/** The toolbar and the table, with the list's loading, empty, no-match and error states. */
function TeamsCard({ onNew, onOpen, actions }: TeamsCardProps) {
  const teams = useTeams();
  const filters = useListFilters();

  if (!teams.data) {
    return (
      <Card>
        {teams.isError ? (
          <ErrorState error={teams.error} onRetry={() => void teams.refetch()} />
        ) : (
          <Skeleton lines={5} height="28px" />
        )}
      </Card>
    );
  }

  const all = teams.data.items;
  if (all.length === 0) {
    return (
      <Card>
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
      </Card>
    );
  }

  const { shown, counts } = filterRecords(
    all,
    filters,
    (team) => `${team.name} ${team.description}`,
  );

  return (
    <Card>
      <ListToolbar
        search={{
          label: 'Search team name or description',
          placeholder: 'Search team name or description',
          value: filters.q,
          onChange: filters.setQ,
        }}
        status={{ value: filters.status, onChange: filters.setStatus, counts }}
        count={{ shown: shown.length, total: all.length, noun: TEAM_NOUN }}
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
                <th scope="col">Team</th>
                <th scope="col">Description</th>
                <th scope="col" className={styles.number}>
                  Members
                </th>
                <th scope="col" className={styles.number}>
                  Plans
                </th>
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
  const canChange = useCanChange('people.manage');
  return (
    <tr>
      <th scope="row" className={`${styles.nameCell} ${styles.nowrap}`}>
        <TeamPill team={team} to={`/settings/teams/${encodeURIComponent(team.id)}`} />
      </th>
      <td>
        <span className={page.clamp}>{team.description || '—'}</span>
      </td>
      <td className={styles.number}>
        <span className={page.members}>
          {team.members.length > 0 && (
            <ul className={page.avatars} aria-hidden="true">
              {team.members.slice(0, AVATARS_SHOWN).map((member) => (
                <li key={member.id}>
                  <Avatar initials={member.initials} size="sm" />
                </li>
              ))}
            </ul>
          )}
          {formatWhole(team.members.length)}
        </span>
      </td>
      <td className={styles.number}>{formatWhole(team.plans.length)}</td>
      <td className={styles.nowrap}>
        <StatusChip tone={team.active ? 'positive' : 'neutral'}>
          {team.active ? 'Active' : 'Inactive'}
        </StatusChip>
      </td>
      <td className={styles.actionsCell}>
        <ActionsMenu
          label={team.name}
          items={[
            { label: 'View', onSelect: () => onOpen(team) },
            ...(canChange
              ? [
                  { label: 'Edit', onSelect: () => actions.edit(team) },
                  ...teamMoreItems(team, actions),
                ]
              : []),
          ]}
        />
      </td>
    </tr>
  );
}
