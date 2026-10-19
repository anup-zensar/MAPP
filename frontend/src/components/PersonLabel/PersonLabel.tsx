import type { ReactNode } from 'react';
import { cx } from '@/lib/cx';
import { Avatar } from '../Avatar/Avatar';
import styles from './PersonLabel.module.css';

/** What a person label needs: a PersonRef, a Person or the signed-in person will do. */
export interface PersonLabelPerson {
  name: string;
  initials: string;
  active: boolean;
  jobTitle?: string;
}

export interface PersonLabelProps {
  person: PersonLabelPerson;
  /** Adds " (you)": the person is the one signed in. */
  you?: boolean;
  /** A second, smaller line: a job title, an e-mail, the person's teams. */
  detail?: ReactNode;
  size?: 'sm' | 'md';
}

/**
 * A person wherever NorthStar names one (feature 003, SC-003): avatar, full name, " (you)" and
 * " (inactive)" written out — never shown by colour alone — and an optional second line.
 */
export function PersonLabel({ person, you = false, detail, size = 'sm' }: PersonLabelProps) {
  return (
    <span className={cx(styles.label, styles[size])}>
      <Avatar initials={person.initials} size={size} />
      <span className={styles.text}>
        <span className={styles.name}>
          {person.name}
          {you && ' (you)'}
          {!person.active && ' (inactive)'}
        </span>
        {detail && <span className={styles.detail}>{detail}</span>}
      </span>
    </span>
  );
}
