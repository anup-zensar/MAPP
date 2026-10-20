import { CircleAlert } from 'lucide-react';
import { useCurrentUser } from '@/app/useSession';
import type { components } from '@/api/schema';
import { PersonLabel, StatusChip } from '@/components';
import type { DateFormat } from '@/lib/format';
import { useFormatters } from '@/lib/useFormatters';
import type { ApprovalPerson, ApprovalRound } from './approvalsApi';
import {
  chainState,
  escalatedText,
  levelRuleLines,
  names,
  progressText,
  reminderSentText,
  sourceText,
  wasLeftOutText,
} from './messages';
import type { ChainState } from './messages';
import styles from './ApprovalChain.module.css';

type WorkflowLevel = components['schemas']['ApprovalWorkflowLevel'];
type ChainLevel = components['schemas']['ApprovalChainLevel'];

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
  /** The level's rules and what happened to it, one line each, in secondary ink. */
  lines: string[];
  /** Each approver's own state on a level where all must approve ("Approved", "Waiting"). */
  approverState?: (person: ApprovalPerson) => string | undefined;
  /** "No approver" and its advice (017 data-model §4). */
  noApprover?: string;
}

/**
 * The approval chain (008 FR-024, 017 FR-033): one row per level, in order — the badge "L1", the
 * level's name, its approvers ("(you)" for the signed-in person, "(inactive)"), where they came
 * from (line manager, role), and its state as a status chip with what follows it ("Approved" by
 * Carlos Mendes on 12 Mar 2026, "Waiting", "Not needed — only when …", "Approved automatically —
 * …"). Under it: "1 of 3 approved · waiting for …" on a level where all must approve, the reminder
 * and escalation, the people left out and why, and "No approver" with its advice. Each state word
 * has one colour (design system §4.1).
 */
export function ApprovalChain(props: ApprovalChainProps) {
  const user = useCurrentUser();
  const { dateFormat } = useFormatters();
  const rows: ChainRow[] =
    'round' in props
      ? props.round.levels.map((level) => roundRow(level, props.round, dateFormat))
      : props.upcomingLevels.map((level) => ({
          number: level.number,
          label: level.label,
          approvers: level.approvers,
          lines: levelRuleLines(level),
        }));

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
                    <PersonLabel
                      person={person}
                      you={person.id === user.id}
                      detail={approverDetail(row, person)}
                    />
                  </li>
                ))}
              </ul>
            )}
            {row.lines.length > 0 && (
              <ul role="list" className={styles.lines}>
                {row.lines.map((line) => (
                  <li key={line}>{line}</li>
                ))}
              </ul>
            )}
            {row.noApprover && (
              <p className={styles.warning}>
                <CircleAlert aria-hidden="true" />
                <span>
                  <strong>No approver.</strong> {row.noApprover}
                </span>
              </p>
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

/**
 * What follows an approver's name: their own state on a level where all must approve, and 014's
 * flag when they hold no role that approves (FR-028).
 */
function approverDetail(row: ChainRow, person: ApprovalPerson): string | undefined {
  const parts = [
    row.approverState?.(person),
    person.canApprove === false ? "(can't approve — no approving role)" : undefined,
  ].filter(Boolean);
  return parts.length > 0 ? parts.join(' ') : undefined;
}

/** A level of a round with its state and the lines of 017 (absent fields read as before 017). */
function roundRow(level: ChainLevel, round: ApprovalRound, dateFormat: DateFormat): ChainRow {
  const all = level.mode === 'all' && (level.requiredCount ?? 0) > 1;
  const approvedIds = new Set(
    (level.decisions ?? [])
      .filter((decision) => decision.action === 'approved')
      .map((decision) =>
        decision.capacity === 'delegate' && decision.asDelegateOf
          ? decision.asDelegateOf.id
          : decision.actor.id,
      ),
  );
  const waiting = level.approvers.filter((person) => !approvedIds.has(person.id));
  const open = level.state === 'waiting' || level.state === 'notReached';

  const lines = [
    sourceText(level.source),
    all
      ? [
          progressText(level.approvedCount ?? approvedIds.size, level.requiredCount ?? 0),
          level.state === 'waiting' && waiting.length > 0 ? `waiting for ${names(waiting)}` : '',
        ]
          .filter(Boolean)
          .join(' · ')
      : '',
    level.remindedAt ? reminderSentText(level.remindedAt, dateFormat) : '',
    level.escalatedAt && level.escalatedTo
      ? escalatedText(level.escalatedTo.name, level.escalatedAt, dateFormat)
      : '',
    ...(level.excluded ?? []).map(wasLeftOutText),
  ].filter(Boolean);

  return {
    number: level.number,
    label: level.label,
    approvers: level.approvers,
    state: chainState(level, round.outcome, dateFormat),
    lines,
    approverState:
      all && level.state !== 'notReached'
        ? (person) => (approvedIds.has(person.id) ? 'Approved' : open ? 'Waiting' : undefined)
        : undefined,
    noApprover: level.noApprover && open ? level.advice || 'Nobody can approve this level.' : '',
  };
}
