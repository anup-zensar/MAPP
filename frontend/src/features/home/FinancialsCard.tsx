import { ChevronRight, ReceiptText, TriangleAlert, Unlink } from 'lucide-react';
import type { ReactNode } from 'react';
import { Link } from 'react-router';
import { LinkButton } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { InfoTip } from '@/ui/InfoTip/InfoTip';
import { ProgressBar } from '@/ui/ProgressBar/ProgressBar';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { cx } from '@/lib/cx';
import { useFormatters } from '@/lib/useFormatters';
import styles from './FinancialsCard.module.css';
import { useFinancialOverview } from './homeApi';
import type { FinancialOverview, HomeScope } from './homeApi';
import { committedAboveHref, FINANCIALS_HREF, NOT_LINKED_HREF } from './homeLinks';
import {
  FINANCIALS_CARD,
  aboveThePlannedText,
  accruedLabel,
  accruedTip,
  aboveByText,
  financialsTitle,
  notLinkedText,
  ofPlannedText,
  percentText,
  sourceText,
} from './homeText';
import { SectionError } from './SectionError';

/**
 * Home's Financials card (015 US6, FR-025), right under the summary cards: for the current year
 * and the Home scope, Committed against planned cost with its meter, Invoiced, Paid and Accrued;
 * then "n activities committed above planned cost" (All work with that filter) and "n transactions
 * not linked to a line item" (Financials › Not linked), Open Financials and the source line. It
 * loads and fails on its own, like the other Home cards (009 FR-022).
 */
export function FinancialsCard({ scope }: { scope: HomeScope }) {
  const { displayCurrency } = useFormatters();
  const overview = useFinancialOverview(scope, displayCurrency);
  const open = (
    <LinkButton to={FINANCIALS_HREF} size="sm" variant="secondary">
      {FINANCIALS_CARD.open}
    </LinkButton>
  );

  if (overview.isPending) {
    return (
      <Card title="Financials" actions={open}>
        <Skeleton lines={3} height="18px" />
      </Card>
    );
  }
  if (overview.isError) {
    return (
      <Card title="Financials" actions={open}>
        <SectionError onRetry={() => void overview.refetch()} />
      </Card>
    );
  }
  return <Overview overview={overview.data} open={open} />;
}

function Overview({ overview, open }: { overview: FinancialOverview; open: ReactNode }) {
  const { money, dateTime } = useFormatters();
  const { figures, source, year } = overview;
  const empty = figures.transactionCount === 0;

  // Without any GSAP data the empty state says so; the source line would only repeat it.
  const subtitle = source.dataAsOf ? sourceText(source, dateTime) : undefined;

  return (
    <Card title={financialsTitle(year)} subtitle={subtitle} actions={open}>
      {empty ? (
        <EmptyState
          icon={<ReceiptText />}
          title={source.dataAsOf ? FINANCIALS_CARD.empty : FINANCIALS_CARD.noData}
        />
      ) : (
        <dl className={styles.figures}>
          <Figure
            label={FINANCIALS_CARD.committed}
            tip={FINANCIALS_CARD.committedTip}
            value={money(figures.committed)}
            problem={figures.committedAbovePlanned}
          >
            <Committed overview={overview} />
          </Figure>
          <Figure
            label={FINANCIALS_CARD.invoiced}
            tip={FINANCIALS_CARD.invoicedTip}
            value={money(figures.invoiced)}
          />
          <Figure
            label={FINANCIALS_CARD.paid}
            tip={FINANCIALS_CARD.paidTip}
            value={money(figures.paid)}
          />
          <Figure
            label={accruedLabel(source.accrualMonth)}
            tip={accruedTip(source.accrualMonth)}
            value={money(figures.accrued)}
          />
        </dl>
      )}
      {(overview.activitiesAbovePlanned > 0 || overview.notLinked > 0) && (
        <ul className={styles.links}>
          {overview.activitiesAbovePlanned > 0 && (
            <li>
              <Link to={committedAboveHref(year)} className={cx(styles.link, styles.problem)}>
                <TriangleAlert aria-hidden="true" className={styles.icon} />
                {aboveThePlannedText(overview.activitiesAbovePlanned)}
                <ChevronRight aria-hidden="true" className={styles.chevron} />
              </Link>
            </li>
          )}
          {overview.notLinked > 0 && (
            <li>
              <Link to={NOT_LINKED_HREF} className={styles.link}>
                <Unlink aria-hidden="true" className={styles.icon} />
                {notLinkedText(overview.notLinked)}
                <ChevronRight aria-hidden="true" className={styles.chevron} />
              </Link>
            </li>
          )}
        </ul>
      )}
    </Card>
  );
}

interface FigureProps {
  label: string;
  tip: string;
  value: string;
  /** Committed above planned cost: the value in red (with words beside it, not colour alone). */
  problem?: boolean;
  children?: ReactNode;
}

/** One figure: its label with an ⓘ, the amount, and an optional line under it. */
function Figure({ label, tip, value, problem = false, children }: FigureProps) {
  return (
    <div className={styles.figure}>
      <dt className={styles.label}>
        {label}
        <InfoTip label={label} text={tip} />
      </dt>
      <dd className={cx(styles.value, problem && styles.problem)}>{value}</dd>
      {children && <dd className={styles.detail}>{children}</dd>}
    </div>
  );
}

/**
 * Committed's lines: "of $460,000.00 planned" and the meter with "8%" after it; above planned cost
 * the meter and the percentage turn red and "Committed above planned cost by $2,000.00" says so in
 * words (015 FR-013); "No planned cost yet" without a planned cost.
 */
function Committed({ overview }: { overview: FinancialOverview }) {
  const { money } = useFormatters();
  const { plannedCost, committedPercent, committedAbovePlanned, notYetCommitted } =
    overview.figures;
  if (!plannedCost) {
    return <span>{FINANCIALS_CARD.noPlannedCost}</span>;
  }
  return (
    <>
      <span>{ofPlannedText(money(plannedCost))}</span>
      {committedPercent !== null && (
        <span className={styles.meter}>
          <ProgressBar
            value={committedPercent}
            label={FINANCIALS_CARD.meterLabel}
            tone={committedAbovePlanned ? 'problem' : undefined}
          />
          <span className={cx(styles.percent, committedAbovePlanned && styles.problem)}>
            {percentText(committedPercent)}
          </span>
        </span>
      )}
      {committedAbovePlanned && notYetCommitted?.amount.startsWith('-') && (
        <span className={styles.problem}>
          {aboveByText(money({ ...notYetCommitted, amount: notYetCommitted.amount.slice(1) }))}
        </span>
      )}
    </>
  );
}
