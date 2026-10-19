import type { CSSProperties, ReactNode } from 'react';
import { Link } from 'react-router';
import { InfoTip, Tooltip } from '@/components';
import type { StatusTone } from '@/components';
import { cx } from '@/lib/cx';
import { cardLinkName } from './homeText';
import styles from './LinkSummaryCard.module.css';

export interface LinkSummaryCardProps {
  /** A lucide icon on a tinted tile (ZenReach home). */
  icon: ReactNode;
  /** The colour of the tile, as CSS: "var(--blue)". */
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
 * A summary card of Home that opens its list (FR-005, FR-007): the whole card is one link, named
 * "<label>: <value> — <link label>". It is the link variant of the shared SummaryCard (plan
 * "Shared parts", D-197), kept in Home's folder until it moves to `@/components`.
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
  const name = cardLinkName(label, isShortened ? `${value} (${fullValue})` : value, linkLabel);
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
