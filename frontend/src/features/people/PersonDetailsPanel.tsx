import { Pencil } from 'lucide-react';
import { useState } from 'react';
import { ApiError } from '@/api/errors';
import { useCurrentUser } from '@/session/useSession';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { Avatar } from '@/ui/Avatar/Avatar';
import { Button } from '@/ui/Button/Button';
import { CanChange } from '@/ui/CanChange/CanChange';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { SearchField } from '@/ui/SearchField/SearchField';
import { SidePanel } from '@/ui/SidePanel/SidePanel';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { TeamPills } from '@/ui/TeamPill/TeamPill';
import styles from '@/ui/settings-table.module.css';
import { PersonRolesFact } from './access/PersonRolesFact';
import { usePerson } from './api';
import type { PersonDetails } from './api';
import panel from './panels.module.css';
import { personStatusItem } from './person-actions';
import type { PersonActions } from './person-actions';
import { RecordLine } from './RecordLine';
import { approvesInText, ownsText } from './texts';

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
