import { Link } from 'react-router';
import styles from '@/features/settings/shared/SettingsTable.module.css';
import { usePersonRoles } from './accessApi';
import { assignmentText } from './accessWords';

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
  const assignments = roles.data.roles;
  return (
    <div>
      <dt>Roles</dt>
      <dd className={styles.usage}>
        <span>
          {assignments.length > 0 ? assignments.map(assignmentText).join(', ') : 'No role'}
        </span>
        <Link
          className={styles.inlineLink}
          to={`/settings/access?tab=people&person=${encodeURIComponent(personId)}`}
        >
          Change roles
        </Link>
      </dd>
    </div>
  );
}
