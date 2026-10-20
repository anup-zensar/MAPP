import { CircleAlert, Users } from 'lucide-react';
import { useState } from 'react';
import { useSearchParams } from 'react-router';
import { useCurrentUser } from '@/app/useSession';
import {
  Button,
  Card,
  EmptyState,
  ErrorState,
  PersonLabel,
  SearchField,
  SegmentedFilter,
  Skeleton,
  TeamPills,
} from '@/components';
import list from '@/features/settings/shared/SettingsTable.module.css';
import { formatWhole } from '@/lib/decimal';
import { usePeopleAccess, useRoles } from './accessApi';
import type { PersonRoles } from './accessApi';
import { assignmentText, SCOPE_EMPTY } from './accessWords';
import { PersonRolesPanel } from './PersonRolesPanel';
import styles from './access.module.css';

/** The role filter: every person, one role's holders, or the people without a role. */
const ALL = 'all';
const NO_ROLE = 'none';

/**
 * The People tab (US3 scenario 1, FR-017): everyone with their teams and roles ("Planner · My
 * teams"), a search, a role filter that includes "No role", and **Edit roles**, which opens the
 * person's roles panel (also from `?person=U-…`).
 */
export function PeopleAccessTab() {
  const people = usePeopleAccess();
  const roles = useRoles();
  const user = useCurrentUser();
  const [params, setParams] = useSearchParams();
  const [query, setQuery] = useState('');
  const [role, setRole] = useState(ALL);
  const openId = params.get('person') ?? '';

  function open(personId: string) {
    setParams((previous) => {
      const next = new URLSearchParams(previous);
      if (personId) {
        next.set('person', personId);
      } else {
        next.delete('person');
      }
      return next;
    });
  }

  if (!people.data) {
    return (
      <Card>
        {people.isError ? (
          <ErrorState error={people.error} onRetry={() => void people.refetch()} />
        ) : (
          <Skeleton lines={6} height="28px" />
        )}
      </Card>
    );
  }

  const all = people.data.items;
  const found = matches(all, query);
  const shown = found.filter((person) =>
    role === ALL
      ? true
      : role === NO_ROLE
        ? person.roles.length === 0
        : person.roles.some((assignment) => assignment.role.id === role),
  );
  const roleOptions = [
    { value: ALL, label: 'All', count: found.length },
    ...(roles.data?.items ?? []).map((candidate) => ({
      value: candidate.id,
      label: candidate.name,
      count: found.filter((person) =>
        person.roles.some((assignment) => assignment.role.id === candidate.id),
      ).length,
    })),
    {
      value: NO_ROLE,
      label: 'No role',
      count: found.filter((person) => person.roles.length === 0).length,
    },
  ];

  return (
    <Card>
      {all.length === 0 ? (
        <EmptyState
          icon={<Users />}
          title="Nobody can sign in yet — add people in Settings › People"
        />
      ) : (
        <>
          <div className={list.toolbar}>
            <div className={list.search}>
              <SearchField
                label="Search name or job title"
                placeholder="Search name or job title"
                value={query}
                onChange={setQuery}
              />
            </div>
            <SegmentedFilter label="Role" options={roleOptions} value={role} onChange={setRole} />
            <p className={list.resultCount} aria-live="polite">
              {`${formatWhole(shown.length)} of ${formatWhole(all.length)} ${all.length === 1 ? 'person' : 'people'}`}
            </p>
          </div>
          {shown.length === 0 ? (
            <div className={list.noMatch}>
              <p>Nothing matches these filters</p>
              <Button
                onClick={() => {
                  setQuery('');
                  setRole(ALL);
                }}
              >
                Clear filters
              </Button>
            </div>
          ) : (
            <div className={list.tableScroll}>
              <table className={list.table}>
                <thead>
                  <tr>
                    <th scope="col">Person</th>
                    <th scope="col">Teams</th>
                    <th scope="col">Roles</th>
                    <th scope="col">
                      <span className="visually-hidden">Actions</span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {shown.map((person) => (
                    <PersonRow
                      key={person.person.id}
                      person={person}
                      isYou={person.person.id === user.id}
                      onEdit={() => open(person.person.id)}
                    />
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
      {openId && (
        <PersonRolesPanel
          key={openId}
          personId={openId}
          isYou={openId === user.id}
          onClose={() => open('')}
        />
      )}
    </Card>
  );
}

function matches(people: PersonRoles[], query: string): PersonRoles[] {
  const q = query.trim().toLowerCase();
  if (!q) {
    return people;
  }
  return people.filter((person) =>
    `${person.person.name} ${person.person.jobTitle}`.toLowerCase().includes(q),
  );
}

interface PersonRowProps {
  person: PersonRoles;
  isYou: boolean;
  onEdit: () => void;
}

function PersonRow({ person, isYou, onEdit }: PersonRowProps) {
  return (
    <tr>
      <th scope="row" className={`${list.nameCell} ${list.nowrap}`}>
        <PersonLabel person={person.person} you={isYou} />
      </th>
      <td>
        {person.teams.length > 0 ? (
          <TeamPills teams={person.teams} />
        ) : (
          <span className={list.muted}>No team</span>
        )}
      </td>
      <td>
        {person.roles.length === 0 ? (
          <span className={list.muted}>No role</span>
        ) : (
          <ul className={styles.roleChips} aria-label={`Roles of ${person.person.name}`}>
            {person.roles.map((assignment) => (
              <li key={assignment.id}>
                <span className={styles.roleChip}>{assignmentText(assignment)}</span>
                {assignment.scopeEmpty && (
                  <span className={styles.flag}>
                    <CircleAlert aria-hidden="true" />
                    {SCOPE_EMPTY}
                  </span>
                )}
              </li>
            ))}
          </ul>
        )}
      </td>
      <td className={`${list.actionsCell} ${list.nowrap}`}>
        <Button
          size="sm"
          variant="tertiary"
          aria-label={`Edit roles of ${person.person.name}`}
          onClick={onEdit}
        >
          Edit roles
        </Button>
      </td>
    </tr>
  );
}
