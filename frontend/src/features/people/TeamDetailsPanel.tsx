import { Pencil } from 'lucide-react';
import { useState } from 'react';
import { Link } from 'react-router';
import { ApiError } from '@/api/errors';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { Button } from '@/ui/Button/Button';
import { CanChange } from '@/ui/CanChange/CanChange';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { SearchField } from '@/ui/SearchField/SearchField';
import { SidePanel } from '@/ui/SidePanel/SidePanel';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import list from '@/ui/settings-table.module.css';
import { useTeam } from './api';
import type { Team } from './api';
import styles from './panels.module.css';
import { RecordLine } from './RecordLine';
import { teamMoreItems } from './team-actions';
import type { TeamActions } from './team-actions';
import { COLOURS } from './team-colours';
import { usedByText } from './texts';

/** Above this many members, the details offer "Search members". */
const SEARCH_ABOVE = 10;

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
