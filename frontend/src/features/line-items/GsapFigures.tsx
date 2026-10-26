import { CircleAlert } from 'lucide-react';
import type { ReactNode } from 'react';
import { Link } from 'react-router';
import { InfoTip } from '@/ui/InfoTip/InfoTip';
import { ProgressBar } from '@/ui/ProgressBar/ProgressBar';
import { cx } from '@/ui/cx';
import { recordHref } from '@/ui/record-links';
import {
  aboveByText,
  aboveOnText,
  accruedTip,
  FIGURE_LABEL,
  FIGURE_TIP,
  meterText,
  NO_PLANNED_COST,
} from './gsap-financials';
import type { FinancialFiguresData } from './gsap-financials';
import styles from './GsapFigures.module.css';
import { formatMonth, moneyText } from './line-item-math';

export interface GsapFiguresProps {
  figures: FinancialFiguresData;
  /** The latest accrual month in the data ("2026-09"), named in the Accrued label. */
  accrualMonth: string | null;
}

/**
 * The five GSAP figures of a record (015 FR-011, FR-013, FR-014): Committed with its meter "38% of
 * planned cost committed", Invoiced, Paid, Accrued (of the latest accrual month) and Not yet
 * committed, each with its ⓘ; "Committed above planned cost by X" in red words, and for an
 * activity "Committed above planned cost on n line items" linking to the first. Amounts are in the
 * reader's display currency, as the API answers them.
 */
export function GsapFigures({ figures, accrualMonth }: GsapFiguresProps) {
  const month = accrualMonth ? formatMonth(accrualMonth) : null;
  const over = figures.committedAbovePlanned;
  const notYet = figures.notYetCommitted;
  const first = figures.firstLineItemAbovePlanned;

  return (
    <div className={styles.figures}>
      <ul className={styles.tiles} aria-label="Financial figures">
        <Tile label={FIGURE_LABEL.committed} tip={FIGURE_TIP.committed} value={figures.committed}>
          {figures.committedPercent === null ? (
            <span>{NO_PLANNED_COST}</span>
          ) : (
            <>
              <ProgressBar
                label="Planned cost committed"
                value={figures.committedPercent}
                tone={figures.committedPercent > 100 ? 'problem' : undefined}
              />
              <span>{meterText(figures.committedPercent)}</span>
            </>
          )}
        </Tile>
        <Tile label={FIGURE_LABEL.invoiced} tip={FIGURE_TIP.invoiced} value={figures.invoiced} />
        <Tile label={FIGURE_LABEL.paid} tip={FIGURE_TIP.paid} value={figures.paid} />
        <Tile
          label={month ? `${FIGURE_LABEL.accrued} (${month})` : FIGURE_LABEL.accrued}
          tip={accruedTip(month)}
          value={figures.accrued}
        />
        <Tile
          label={FIGURE_LABEL.notYetCommitted}
          tip={FIGURE_TIP.notYetCommitted}
          value={notYet}
          problem={over}
        >
          {notYet === null && <span>{NO_PLANNED_COST}</span>}
        </Tile>
      </ul>
      {over && notYet && (
        <p className={styles.problem}>
          <CircleAlert aria-hidden="true" />
          {aboveByText(moneyText({ ...notYet, amount: notYet.amount.replace(/^-/, '') }))}
        </p>
      )}
      {figures.lineItemsAbovePlanned > 0 && first && (
        <p className={styles.problem}>
          <CircleAlert aria-hidden="true" />
          <Link to={recordHref('lineItem', first.id)} className={styles.problemLink}>
            {aboveOnText(figures.lineItemsAbovePlanned)}
          </Link>
        </p>
      )}
    </div>
  );
}

interface TileProps {
  label: string;
  tip: string;
  value: { amount: string; currency: string } | null;
  problem?: boolean;
  children?: ReactNode;
}

function Tile({ label, tip, value, problem = false, children }: TileProps) {
  return (
    <li className={styles.tile}>
      <span className={styles.label}>
        {label}
        <InfoTip label={label} text={tip} />
      </span>
      <span className={cx(styles.value, problem && styles.negative)}>
        {value ? moneyText(value) : '—'}
      </span>
      {children && <span className={styles.detail}>{children}</span>}
    </li>
  );
}
