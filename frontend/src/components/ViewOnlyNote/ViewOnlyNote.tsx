import { Lock } from 'lucide-react';
import { useAccess } from '@/lib/access';
import styles from './ViewOnlyNote.module.css';
import { viewOnlyText } from './viewOnlyText';
import type { ViewOnlyArea } from './viewOnlyText';

export interface ViewOnlyNoteProps {
  area: ViewOnlyArea;
}

/**
 * The one note that replaces every change action of a page where the signed-in person may change
 * nothing (feature 014, FR-011, D-230): a lock, "View only", their roles and the administrators to
 * ask, from GET /api/me. Pages show it instead of a page full of disabled buttons.
 */
export function ViewOnlyNote({ area }: ViewOnlyNoteProps) {
  const access = useAccess();
  const roleNames = access.roles.map((r) => r.role.name);
  const administratorNames = access.administrators.map((a) => a.name);

  return (
    <p className={styles.note} role="note">
      <Lock aria-hidden="true" />
      <span>{viewOnlyText(area, roleNames, administratorNames)}</span>
    </p>
  );
}
