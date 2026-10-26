import { Split } from 'lucide-react';
import { Link } from 'react-router';
import type { Activity } from './api';
import { activityHref } from './activity-text';
import styles from './SplitLinks.module.css';
import { splitFromText, splitIntoText } from './split-text';
import type { SplitLink } from './split-api';

/**
 * Where a split activity came from and what was split off it (016 FR-031), under the header: "Split
 * from 'UK launch event' (A-1012)" and "Split into 'Ireland roadshow' (A-1014)", each a link to the
 * other activity — or "(deleted)" without a link when it is in the recycle bin. Nothing when the
 * activity was never split.
 */
export function SplitLinks({ activity }: { activity: Activity }) {
  const from = activity.splitFrom ?? null;
  const into = activity.splitInto ?? [];
  if (!from && into.length === 0) {
    return null;
  }
  return (
    <ul className={styles.links} aria-label="Split">
      {from && (
        <li>
          <Split aria-hidden="true" />
          <Target link={from} text={splitFromText(from.name, from.id, from.deleted)} />
        </li>
      )}
      {into.map((link) => (
        <li key={link.id}>
          <Split aria-hidden="true" />
          <Target link={link} text={splitIntoText(link.name, link.id, link.deleted)} />
        </li>
      ))}
    </ul>
  );
}

function Target({ link, text }: { link: SplitLink; text: string }) {
  return link.deleted ? (
    <span>{text}</span>
  ) : (
    <Link to={activityHref(link.planId, link.id)}>{text}</Link>
  );
}
