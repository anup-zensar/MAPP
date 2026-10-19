import { Link } from 'react-router';
import type { components } from '@/api/schema';
import styles from './TeamPill.module.css';

/** A team as other records show it (contract TeamRef). */
export type TeamRef = components['schemas']['TeamRef'];

export interface TeamPillProps {
  team: TeamRef;
  /** Makes the pill a link, usually to /settings/teams/<id>. */
  to?: string;
}

/**
 * A team's name in a neutral pill with its colour dot (feature 003, D-025, D-201): the dot is
 * decoration, the name is the text; " (inactive)" is written out.
 */
export function TeamPill({ team, to }: TeamPillProps) {
  const content = (
    <>
      <span
        className={styles.dot}
        style={{ background: `var(--team-${team.colour})` }}
        aria-hidden="true"
      />
      {team.name}
      {!team.active && ' (inactive)'}
    </>
  );
  return to ? (
    <Link to={to} className={styles.pill}>
      {content}
    </Link>
  ) : (
    <span className={styles.pill}>{content}</span>
  );
}

export interface TeamPillsProps {
  teams: TeamRef[];
  /** Links each pill to its team's page in Settings. */
  linked?: boolean;
}

/** Several team pills that wrap onto more lines (plan cards, people lists, My profile). */
export function TeamPills({ teams, linked = false }: TeamPillsProps) {
  return (
    <ul className={styles.pills}>
      {teams.map((team) => (
        <li key={team.id}>
          <TeamPill
            team={team}
            to={linked ? `/settings/teams/${encodeURIComponent(team.id)}` : undefined}
          />
        </li>
      ))}
    </ul>
  );
}
