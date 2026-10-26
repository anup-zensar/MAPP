import { ChevronRight } from 'lucide-react';
import type { ComponentProps } from 'react';
import { Link } from 'react-router';
import { SummaryCard } from '@/ui/SummaryCard/SummaryCard';
import styles from './SummaryLinkCard.module.css';

export interface SummaryLinkCardProps extends ComponentProps<typeof SummaryCard> {
  /** The list the card opens. */
  to: string;
  /** Ends the link's name: "<label>: <value> — <link label>". */
  linkLabel: string;
}

/**
 * One of Home's four summary cards (design system §2.4 E, §5.3 LinkSummaryCard): the shared
 * SummaryCard — the same look as every other page's summary cards, with no icon tile (UI audit
 * H-03, D-01) — made into one link to its list, with a chevron at the top end. The link lies over
 * the whole card; the ⓘ and a shortened value's button stay above it, so they still open their
 * tips.
 */
export function SummaryLinkCard({ to, linkLabel, ...card }: SummaryLinkCardProps) {
  const value = String(card.value);
  const shortened = card.fullValue !== undefined && card.fullValue !== value;
  const name = `${card.label}: ${shortened ? `${value} (${card.fullValue})` : value} — ${linkLabel}`;

  return (
    <div className={styles.card}>
      <Link to={to} aria-label={name} className={styles.link} />
      <SummaryCard {...card} />
      <ChevronRight aria-hidden="true" className={styles.chevron} />
    </div>
  );
}
