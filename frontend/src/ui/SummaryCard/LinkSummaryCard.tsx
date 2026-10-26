import type { CSSProperties, ReactNode } from 'react';
import { Link } from 'react-router';
import { cx } from '@/lib/cx';
import { InfoTip } from '../InfoTip/InfoTip';
import type { StatusTone } from '../StatusChip/StatusChip';
import { Tooltip } from '../Tooltip/Tooltip';
import styles from './LinkSummaryCard.module.css';

export interface LinkSummaryCardProps {
  /** A lucide icon in the ink colour on a lightly tinted tile. */
  icon: ReactNode;
  /** The colour of the tile, as CSS: "var(--info)". */
  tint: string;
  label: string;
  /** A plain explanation of the label, shown in an info tip. */
  tip: string;
  /** The number or amount, already formatted ("6", "$1.2M", "—"). */
  value: string;
  /** The complete value when `value` is shortened ("$1,200,000.00" for "$1.2M"). */
  fullValue?: string;
  /** Colours the value, only when the colour means something (amber: something waits). */
  tone?: StatusTone;
  /** Lines under the value: counts, a budget-used bar. */
  detail?: ReactNode;
  /** The list the card opens; "#id" scrolls to a part of this page. */
  to: string;
  /** Ends the link's name: "<label>: <value> — open the Plans list". */
  linkLabel: string;
}

/**
 * The link variant of SummaryCard (feature 009, D-197): a summary card that opens its list, as on
 * Home (009 FR-005, FR-007). The whole card is one link, named "<label>: <value> — <link label>";
 * its icon sits on a tinted tile, and the info tip stays above the link.
 */
export function LinkSummaryCard({
  icon,
  tint,
  label,
  tip,
  value,
  fullValue,
  tone,
  detail,
  to,
  linkLabel,
}: LinkSummaryCardProps) {
  const isShortened = fullValue !== undefined && fullValue !== value;
  const name = `${label}: ${isShortened ? `${value} (${fullValue})` : value} — ${linkLabel}`;
  // The link covers the whole card (its ::after); the visible value is its text.
  const link = to.startsWith('#') ? (
    <a href={to} aria-label={name} className={styles.link}>
      {value}
    </a>
  ) : (
    <Link to={to} aria-label={name} className={styles.link}>
      {value}
    </Link>
  );

  return (
    <div className={styles.card} style={{ '--tint': tint } as CSSProperties}>
      <p className={styles.label}>
        <span className={styles.icon} aria-hidden="true">
          {icon}
        </span>
        {label}
        <span className={styles.tip}>
          <InfoTip label={label} text={tip} />
        </span>
      </p>
      <p className={cx(styles.value, tone && styles[tone])}>
        {isShortened ? <Tooltip content={fullValue}>{link}</Tooltip> : link}
      </p>
      {detail && <div className={styles.detail}>{detail}</div>}
    </div>
  );
}
