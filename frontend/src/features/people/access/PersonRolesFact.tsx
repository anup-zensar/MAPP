import { Link } from 'react-router';
import styles from '@/ui/settings-table.module.css';
import { usePersonRoles } from './accessApi';
import { personRolesPath } from './accessPaths';
import { rolesText } from './accessWords';

/**
 * The "Roles" line of a person's details panel (014 plan, "Changes to existing web features"): each
 * role with its scope ("Planner · My teams") and **Change roles**, which opens the person's roles on
 * Settings › Roles and access. Only people who may manage access can read someone's roles, so for
 * everyone else — and while the roles can't be read — the line is left out.
 */
export function PersonRolesFact({ personId }: { personId: string }) {
  const roles = usePersonRoles(personId);
  if (!roles.data) {
    return null;
  }
  return (
    <div>
      <dt>Roles</dt>
      <dd className={styles.usage}>
        <span>{rolesText(roles.data.roles)}</span>
        <Link className={styles.inlineLink} to={personRolesPath(personId)}>
          Change roles
        </Link>
      </dd>
    </div>
  );
}
