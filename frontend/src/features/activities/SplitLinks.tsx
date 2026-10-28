import { Split } from 'lucide-react';
import { Link } from 'react-router';
import { FieldHelpTip } from '@/features/settings';
import type { Activity } from './api';
import { activityHref } from './activity-text';
import styles from './SplitLinks.module.css';
import { splitFromText, splitIntoText } from './split-text';
import type { SplitLink } from './split-api';

/** "Split from 'UK launch event' (A-1012)" and "Split into …" links, "(deleted)" ones without a link. */
export function SplitLinks({ activity }: { activity: Activity }) {
  const from = activity.splitFrom ?? null;
  const into = activity.splitInto ?? [];
  if (!from && into.length === 0) {
    return null;
  }
  const tip = <FieldHelpTip helpKey="activity.splitFrom" label="Split from / Split into" />;
  return (
    <ul className={styles.links} aria-label="Split">
      {from && (
        <li>
          <Split aria-hidden="true" />
          <Target link={from} text={splitFromText(from.name, from.id, from.deleted)} />
          {tip}
        </li>
      )}
      {into.map((link, index) => (
        <li key={link.id}>
          <Split aria-hidden="true" />
          <Target link={link} text={splitIntoText(link.name, link.id, link.deleted)} />
          {!from && index === 0 && tip}
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
