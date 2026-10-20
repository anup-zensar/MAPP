import { useCurrentUser } from '@/app/useSession';
import type { components } from '@/api/schema';
import { PersonLabel, StatusChip } from '@/components';
import { useFormatters } from '@/lib/useFormatters';
import type { ApprovalPerson, ApprovalRound } from './approvalsApi';
import { chainState } from './messages';
import type { ChainState } from './messages';
import styles from './ApprovalChain.module.css';

type WorkflowLevel = components['schemas']['ApprovalWorkflowLevel'];

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
}

/**
 * The approval chain (FR-024): one row per level, in order — the badge "L1", the level's name,
 * its approvers ("(you)" for the signed-in person, "(inactive)") and its state as a status chip
 * with what follows it ("Approved" by Carlos Mendes on 12 Mar 2026, "Waiting", "Not reached
 * yet", …). Each state word has one colour (design system §4.1).
 */
export function ApprovalChain(props: ApprovalChainProps) {
  const user = useCurrentUser();
  const { dateFormat } = useFormatters();
  const rows: ChainRow[] =
    'round' in props
      ? props.round.levels.map((level) => ({
          ...level,
          state: chainState(level, props.round.outcome, dateFormat),
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
            {row.approvers.length > 0 && (
              <ul role="list" className={styles.approvers}>
                {row.approvers.map((person) => (
                  <li key={person.id}>
                    <PersonLabel person={person} you={person.id === user.id} />
                  </li>
                ))}
              </ul>
            )}
          </div>
          {row.state && (
            <p className={styles.state}>
              <StatusChip tone={row.state.tone}>{row.state.word}</StatusChip>
              {row.state.detail && <span className={styles.detail}> {row.state.detail}</span>}
            </p>
          )}
        </li>
      ))}
    </ol>
  );
}
