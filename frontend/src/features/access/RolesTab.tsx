import { Plus, ShieldCheck } from 'lucide-react';
import { Link, useNavigate } from 'react-router';
import {
  ActionsMenu,
  Card,
  EmptyState,
  ErrorState,
  LinkButton,
  Skeleton,
  StatusChip,
} from '@/components';
import type { ActionsMenuItem } from '@/components';
import list from '@/features/settings/shared/SettingsTable.module.css';
import { formatWhole } from '@/lib/decimal';
import { useRoles } from './accessApi';
import type { Role } from './accessApi';
import { SCOPE_WORDS } from './accessWords';
import { roleEditPath, rolePath, roleCopyPath, NEW_ROLE_PATH } from './accessPaths';
import { deleteLock, useRoleDelete } from './useRoleDelete';
import styles from './access.module.css';

/**
 * The Roles tab (US4, FR-018): every role with its description, default scope and how many people
 * hold it, "Built in" for the seeded ones; **New role** (empty), and in each row's ⋯ View, Edit,
 * Copy as new role and Delete. The Administrator role can only be viewed and copied.
 */
export function RolesTab() {
  const roles = useRoles();
  const navigate = useNavigate();
  const remover = useRoleDelete();

  if (!roles.data) {
    return (
      <Card>
        {roles.isError ? (
          <ErrorState error={roles.error} onRetry={() => void roles.refetch()} />
        ) : (
          <Skeleton lines={5} height="28px" />
        )}
      </Card>
    );
  }

  const items = roles.data.items;
  const newRole = (
    <LinkButton to={NEW_ROLE_PATH} variant="primary" icon={<Plus />}>
      New role
    </LinkButton>
  );

  return (
    <Card>
      {items.length === 0 ? (
        <EmptyState
          icon={<ShieldCheck />}
          title="No roles yet — create the first role"
          action={newRole}
        />
      ) : (
        <>
          <div className={list.toolbar}>
            <p className={list.resultCount}>
              {`${formatWhole(items.length)} ${items.length === 1 ? 'role' : 'roles'}`}
            </p>
            <div className={styles.toolbarEnd}>{newRole}</div>
          </div>
          <div className={list.tableScroll}>
            <table className={list.table}>
              <thead>
                <tr>
                  <th scope="col">Role</th>
                  <th scope="col">Description</th>
                  <th scope="col">Default scope</th>
                  <th scope="col" className={list.number}>
                    People
                  </th>
                  <th scope="col">
                    <span className="visually-hidden">Actions</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {items.map((role) => (
                  <tr key={role.id}>
                    <th scope="row" className={`${list.nameCell} ${list.nowrap}`}>
                      <Link className={list.inlineLink} to={rolePath(role.id)}>
                        {role.name}
                      </Link>{' '}
                      {role.builtIn && <StatusChip tone="neutral">Built in</StatusChip>}
                    </th>
                    <td>{role.description || '—'}</td>
                    <td className={list.nowrap}>{SCOPE_WORDS[role.defaultScope]}</td>
                    <td className={list.number}>{formatWhole(role.people.length)}</td>
                    <td className={list.actionsCell}>
                      <ActionsMenu
                        label={role.name}
                        items={roleMenuItems(role, {
                          view: () => void navigate(rolePath(role.id)),
                          edit: () => void navigate(roleEditPath(role.id)),
                          copy: () => void navigate(roleCopyPath(role.id)),
                          remove: () => remover.remove(role),
                        })}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
      {remover.dialog}
    </Card>
  );
}

interface RoleMenuActions {
  view: () => void;
  edit: () => void;
  copy: () => void;
  remove: () => void;
}

/** View · Edit · Copy as new role · Delete (after a line, with why it can't be used). */
function roleMenuItems(role: Role, actions: RoleMenuActions): ActionsMenuItem[] {
  const isAdministrator = role.builtIn === 'administrator';
  const reason = deleteLock(role);
  return [
    { label: 'View', onSelect: actions.view },
    ...(isAdministrator ? [] : [{ label: 'Edit', onSelect: actions.edit }]),
    { label: 'Copy as new role', onSelect: actions.copy },
    ...(isAdministrator
      ? []
      : [
          {
            label: 'Delete',
            tone: 'danger' as const,
            separatorBefore: true,
            hint: reason || undefined,
            onSelect: actions.remove,
          },
        ]),
  ];
}
