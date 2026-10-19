import { useCurrentUser } from '@/app/useSession';
import type { components } from '@/api/schema';
import { PersonLabel } from '@/components';
import { cx } from '@/lib/cx';
import { useFormatters } from '@/lib/useFormatters';
import type { ApprovalPerson, ApprovalRound } from './approvalsApi';
import { chainStateText } from './messages';
import styles from './ApprovalChain.module.css';

type WorkflowLevel = components['schemas']['ApprovalWorkflowLevel'];
type ChainState = components['schemas']['ApprovalChainLevel']['state'];

export type ApprovalChainProps =
  /** The chain of a round: each level with its state. */
  | { round: ApprovalRound }
  /** The levels of the chosen workflow while the activity was never submitted (no states). */
  | { upcomingLevels: WorkflowLevel[] };

interface ChainRow {
  number: number;
  label: string;
  approvers: ApprovalPerson[];
  state?: ChainState;
  stateText?: string;
}

/**
 * The approval chain (FR-024): one row per level, in order — the badge "L1", the level's name,
 * its approvers ("(you)" for the signed-in person, "(inactive)") and its state ("Approved by
 * Carlos Mendes on 12 Mar 2026", "Waiting", "Not reached yet", …).
 */
export function ApprovalChain(props: ApprovalChainProps) {
  const user = useCurrentUser();
  const { dateFormat } = useFormatters();
  const rows: ChainRow[] =
    'round' in props
      ? props.round.levels.map((level) => ({
          ...level,
          stateText: chainStateText(level, props.round.outcome, dateFormat),
        }))
      : props.upcomingLevels;

  return (
    <ol role="list" className={styles.chain}>
      {rows.map((row) => (
        <li key={row.number} className={styles.level}>
          <span className={styles.badge} aria-hidden="true">
            {`L${row.number}`}
          </span>
          <div className={styles.main}>
            <p className={styles.label}>
              {row.label}
              {/* The badge is decoration: screen readers hear the level's number here. */}
              {row.label !== `Level ${row.number}` && (
                <span className="visually-hidden">{` (level ${row.number})`}</span>
              )}
            </p>
            <ul role="list" className={styles.approvers}>
              {row.approvers.map((person) => (
                <li key={person.id}>
                  <PersonLabel person={person} you={person.id === user.id} />
                </li>
              ))}
            </ul>
          </div>
          {row.stateText && (
            <p className={cx(styles.state, row.state && styles[row.state])}>{row.stateText}</p>
          )}
        </li>
      ))}
    </ol>
  );
}
