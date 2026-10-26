import { Link } from 'react-router';
import { useCurrentUser } from '@/session/useSession';
import { Card } from '@/ui/Card/Card';
import { ACCESS_PATH } from './accessPaths';
import { assignmentText, orList, summariseAccess } from './accessWords';
import styles from './access.module.css';

/**
 * "Your access" on My profile (FR-024, plan "My profile"): your roles with their scope, what they
 * allow in plain language, and whom to ask for a change. It comes with the session, so it shows
 * at once; an older answer without access leaves the card out.
 */
export function YourAccessCard() {
  const user = useCurrentUser();
  const access = user.access;
  if (!access) {
    return null;
  }
  const others = access.administrators.filter((admin) => admin.id !== user.id);
  const youAdminister = others.length < access.administrators.length;
  const ask = orList(others.map((admin) => admin.name));

  return (
    <Card title="Your access">
      <div className={styles.rules}>
        {access.roles.length === 0 ? (
          <p className={styles.note}>
            {`You don't have access to any plans yet. Ask an administrator${ask ? ` (${ask})` : ''} to give you a role.`}
          </p>
        ) : (
          <>
            <ul className={styles.roleChips} aria-label="Your roles">
              {access.roles.map((assignment) => (
                <li key={assignment.id}>
                  <span className={styles.roleChip}>{assignmentText(assignment)}</span>
                </li>
              ))}
            </ul>
            <ul className={styles.summary}>
              {summariseAccess(access.permissions).map((line) => (
                <li key={line}>{line}</li>
              ))}
            </ul>
          </>
        )}
        {youAdminister ? (
          <p className={styles.note}>
            You change roles in{' '}
            <Link to={ACCESS_PATH} className={styles.inlineSentenceLink}>
              Settings › Roles and access
            </Link>
            .
          </p>
        ) : (
          ask && <p className={styles.note}>{`To change your access, ask ${ask}.`}</p>
        )}
      </div>
    </Card>
  );
}
