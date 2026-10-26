import { Check, Minus, Pencil } from 'lucide-react';
import { useNavigate, useParams } from 'react-router';
import { ApiError } from '@/api/errors';
import { LinkButton } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { LockedNote } from '@/ui/LockedNote/LockedNote';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import list from '@/features/settings/shared/SettingsTable.module.css';
import { usePageTitle } from '@/lib/usePageTitle';
import { usePermissionCatalogue, useRole } from './accessApi';
import type { PermissionCatalogue, Role } from './accessApi';
import { ADMINISTRATOR_LOCK, peopleText, SCOPE_WORDS } from './accessWords';
import { roleCopyPath, roleEditPath, rolesTabPath } from './accessPaths';
import { useRoleDelete } from './useRoleDelete';
import styles from './access.module.css';

/**
 * One role (US4, FR-018) at /settings/access/roles/:roleId: its description, default scope and the
 * people who hold it, then every permission of Table A by group, ticked when the role has it.
 * **Edit** and, in More, Copy as new role and Delete; the Administrator role shows why it can't
 * change and can only be copied.
 */
export function RolePage() {
  const { roleId = '' } = useParams();
  const role = useRole(roleId);
  const catalogue = usePermissionCatalogue();
  usePageTitle(role.data?.name ?? 'Role');

  const breadcrumb = [
    { label: 'Settings', to: '/settings' },
    { label: 'Roles and access', to: rolesTabPath() },
    { label: role.data?.name ?? 'Role' },
  ];

  if (!role.data) {
    const notFound = role.error instanceof ApiError && role.error.kind === 'notFound';
    return (
      <>
        <PageHeader breadcrumb={breadcrumb} title="Role" />
        <Card>
          {role.isPending && <Skeleton lines={6} height="20px" />}
          {notFound && <p className={list.gone}>{role.error?.message}</p>}
          {role.isError && !notFound && (
            <ErrorState error={role.error} onRetry={() => void role.refetch()} />
          )}
        </Card>
      </>
    );
  }

  return (
    <RoleDetails
      role={role.data}
      catalogue={catalogue.data}
      catalogueError={catalogue.error}
      onRetryCatalogue={() => void catalogue.refetch()}
      breadcrumb={breadcrumb}
    />
  );
}

interface RoleDetailsProps {
  role: Role;
  catalogue: PermissionCatalogue | undefined;
  catalogueError: Error | null;
  onRetryCatalogue: () => void;
  breadcrumb: { label: string; to?: string }[];
}

function RoleDetails({
  role,
  catalogue,
  catalogueError,
  onRetryCatalogue,
  breadcrumb,
}: RoleDetailsProps) {
  const navigate = useNavigate();
  const remover = useRoleDelete(() => void navigate(rolesTabPath()));
  const isAdministrator = role.builtIn === 'administrator';
  const granted = new Set(role.permissions);

  return (
    <>
      <PageHeader
        breadcrumb={breadcrumb}
        title={role.name}
        meta={
          <>
            {role.builtIn && <StatusChip tone="neutral">Built in</StatusChip>}
            <span>{`Held by ${peopleText(role.people.length)}`}</span>
          </>
        }
        description={role.description || undefined}
        actions={
          isAdministrator ? (
            <LinkButton to={roleCopyPath(role.id)}>Copy as new role</LinkButton>
          ) : (
            <LinkButton to={roleEditPath(role.id)} variant="primary" icon={<Pencil />}>
              Edit
            </LinkButton>
          )
        }
        moreActions={
          isAdministrator
            ? undefined
            : [
                {
                  label: 'Copy as new role',
                  onSelect: () => void navigate(roleCopyPath(role.id)),
                },
                {
                  label: 'Delete',
                  tone: 'danger',
                  separatorBefore: true,
                  onSelect: () => remover.remove(role),
                },
              ]
        }
      />
      <div className={styles.cards}>
        {isAdministrator && <LockedNote>{ADMINISTRATOR_LOCK}</LockedNote>}
        <Card title="Details">
          <dl className={list.facts}>
            <div>
              <dt>Default scope</dt>
              <dd>{SCOPE_WORDS[role.defaultScope]}</dd>
            </div>
            <div>
              <dt>People</dt>
              <dd>
                {role.people.length === 0 ? (
                  <span className={list.muted}>Nobody holds this role yet.</span>
                ) : (
                  <ul className={styles.summary}>
                    {role.people.map((person) => (
                      <li key={person.id}>
                        <PersonLabel person={person} />
                      </li>
                    ))}
                  </ul>
                )}
              </dd>
            </div>
          </dl>
        </Card>
        <Card title="Permissions">
          {catalogue ? (
            <div className={styles.groups}>
              {catalogue.groups.map((group) => (
                <section key={group.name} className={styles.group} aria-label={group.name}>
                  <h3 className={styles.groupTitle}>{group.name}</h3>
                  <ul className={styles.permissions}>
                    {group.permissions.map((permission) => {
                      const has = granted.has(permission.key);
                      return (
                        <li
                          key={permission.key}
                          className={`${styles.permission} ${has ? styles.granted : styles.notGranted}`}
                        >
                          {has ? <Check aria-hidden="true" /> : <Minus aria-hidden="true" />}
                          <span>
                            {permission.label}
                            <span className="visually-hidden">
                              {has ? ' — allowed' : ' — not allowed'}
                            </span>
                          </span>
                        </li>
                      );
                    })}
                  </ul>
                </section>
              ))}
            </div>
          ) : catalogueError ? (
            <ErrorState error={catalogueError} onRetry={onRetryCatalogue} />
          ) : (
            <Skeleton lines={8} height="18px" />
          )}
        </Card>
      </div>
      {remover.dialog}
    </>
  );
}
