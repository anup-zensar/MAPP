import { House, ShieldOff } from 'lucide-react';
import { usePageTitle } from '@/lib/usePageTitle';
import { LinkButton } from '../Button/Button';
import styles from './NoAccess.module.css';

export interface NoAccessProps {
  /**
   * What the server said (FR-008): "You don't have access to this plan. Ask an administrator if you
   * need it." — it names nothing of the record. A page the person's roles do not open says so itself.
   */
  message?: string;
}

/** The sentence shown when no message is given: a page the person's roles do not open (FR-012). */
export const NO_ACCESS_DEFAULT =
  "You don't have access to this page. Ask an administrator if you need it.";

/**
 * The content of the no-access page (feature 014, FR-008, FR-012): "No access", the server's sentence
 * and **Go to Home**. Shown instead of a record the person may not see — a 403 answer to a page load —
 * and at the address of a page their roles do not open.
 */
export function NoAccess({ message = NO_ACCESS_DEFAULT }: NoAccessProps) {
  usePageTitle('No access');

  return (
    <div className={styles.page}>
      <div className={styles.icon}>
        <ShieldOff aria-hidden="true" />
      </div>
      <h1 className={styles.title}>No access</h1>
      <p className={styles.text}>{message}</p>
      <LinkButton to="/" variant="primary" icon={<House />}>
        Go to Home
      </LinkButton>
    </div>
  );
}
