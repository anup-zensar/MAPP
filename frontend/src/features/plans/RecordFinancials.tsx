import { TriangleAlert } from 'lucide-react';
import { useId } from 'react';
import { Link } from 'react-router';
import { ApiError } from '@/api/errors';
import type { components } from '@/api/generated/schema';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { InfoTip } from '@/ui/InfoTip/InfoTip';
import { ProgressBar } from '@/ui/ProgressBar/ProgressBar';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { recordHref } from '@/ui/record-links';
import { useFormatters } from '@/format/useFormatters';
import { financialsHref, useRecordFinancials } from './financials-api';
import { FinancialsBreakdown } from './FinancialsBreakdown';
import styles from './RecordFinancials.module.css';

type FinancialFigures = components['schemas']['FinancialFigures'];
type Level = components['schemas']['AttributeLevel'];

const LEVEL_WORD: Record<Level, string> = {
  plan: 'plan',
  campaign: 'campaign',
  initiative: 'initiative',
  activity: 'activity',
  lineItem: 'line item',
  purchaseRequest: 'purchase request',
  purchaseRequestLine: 'PR line',
};

const TIPS = {
  committed: 'The value of purchase orders GSAP has issued for these line items.',
  invoiced: 'Supplier invoices GSAP has received against them, less credit notes.',
  paid: 'Payments GSAP has released — Shell’s actuals.',
  notYetCommitted: 'Planned cost minus committed.',
};

export interface RecordFinancialsProps {
  recordId: string;
  planId: string;
  /** 'page' on the plan page (a card, h2), 'panel' in a campaign or initiative panel (h3). */
  variant: 'page' | 'panel';
}

/**
 * **Financials** of a plan, campaign or initiative (015 FR-018, FR-019; D-250, D-305), open where
 * people work: Committed, Invoiced, Paid, Accrued and Not yet committed with the share of the
 * planned cost committed, the red "Committed above planned cost …" warnings in words, the figures
 * of each campaign, initiative or activity below, and View all n transactions. Nothing shows while
 * the API does not serve 015 yet.
 *
 * Lane f1 keeps it in the plans feature until 015's `features/financials/FinancialsCard` lands.
 */
export function RecordFinancials({ recordId, planId, variant }: RecordFinancialsProps) {
  const headingId = useId();
  const { displayCurrency, dateTime } = useFormatters();
  const query = useRecordFinancials(recordId, displayCurrency);
  if (query.error instanceof ApiError && query.error.kind === 'notFound') {
    return null;
  }
  const Heading = variant === 'page' ? 'h2' : 'h3';
  const data = query.data;
  const level = data?.record.type ?? 'plan';

  function content() {
    if (query.error instanceof ApiError && query.error.kind === 'forbidden') {
      return (
        <p className={styles.note}>
          You don’t have access to financial data — ask an administrator.
        </p>
      );
    }
    if (query.isError) {
      return <ErrorState error={query.error} onRetry={() => void query.refetch()} />;
    }
    if (!data) {
      return <Skeleton lines={3} />;
    }
    return (
      <>
        <Figures figures={data.figures} accrualMonth={data.source.accrualMonth} variant={variant} />
        <Warnings figures={data.figures} planId={planId} />
        {data.figures.transactionCount === 0 && (
          <p className={styles.note}>
            No purchase orders or invoices yet — they appear here once GSAP sends them for this{' '}
            {LEVEL_WORD[level]}’s line items.
          </p>
        )}
        {data.breakdownLevel && data.breakdown.length > 0 && (
          <FinancialsBreakdown view={data} planId={planId} />
        )}
        <div className={styles.footer}>
          {data.figures.transactionCount > 0 && (
            <Link to={financialsHref(data.record.id)}>
              View all {data.figures.transactionCount} transactions in Financials
            </Link>
          )}
          <span className={styles.source}>
            {data.source.dataAsOf
              ? `GSAP data as of ${dateTime(data.source.dataAsOf)}${data.source.feedName ? ` · ${data.source.feedName}` : ''}${data.source.fileName ? ` '${data.source.fileName}'` : ''}`
              : 'No GSAP data loaded yet.'}
          </span>
        </div>
      </>
    );
  }

  return (
    <section
      aria-labelledby={headingId}
      className={variant === 'page' ? styles.card : styles.section}
      id={variant === 'page' ? 'financials' : `financials-${recordId}`}
    >
      <div className={styles.header}>
        <Heading id={headingId} className={styles.title}>
          Financials
        </Heading>
        <InfoTip
          label="Financials"
          text={`Purchase orders, invoices, payments and accruals from GSAP for this ${LEVEL_WORD[level]}’s line items.`}
        />
      </div>
      {content()}
    </section>
  );
}

/** "Mar 2026" for an accrual month "2026-03". */
function monthText(month: string): string {
  const names = [
    'Jan',
    'Feb',
    'Mar',
    'Apr',
    'May',
    'Jun',
    'Jul',
    'Aug',
    'Sep',
    'Oct',
    'Nov',
    'Dec',
  ];
  const [year, number] = month.split('-');
  return `${names[Number(number) - 1] ?? ''} ${year}`.trim();
}

function Figures({
  figures,
  accrualMonth,
  variant,
}: {
  figures: FinancialFigures;
  accrualMonth: string | null;
  variant: 'page' | 'panel';
}) {
  const { money } = useFormatters();
  const accrued = accrualMonth ? `Accrued (${monthText(accrualMonth)})` : 'Accrued';
  const negative =
    figures.notYetCommitted !== null && figures.notYetCommitted.amount.startsWith('-');
  return (
    <div className={variant === 'page' ? styles.tiles : styles.tilesPanel}>
      <div className={styles.tile}>
        <p className={styles.label}>
          Committed
          <InfoTip label="Committed" text={TIPS.committed} />
        </p>
        <p className={styles.value}>{money(figures.committed)}</p>
        {figures.committedPercent === null ? (
          <p className={styles.detail}>No planned cost yet</p>
        ) : (
          <>
            <ProgressBar
              value={figures.committedPercent}
              label="Planned cost committed"
              tone={figures.committedAbovePlanned ? 'problem' : undefined}
            />
            <p className={figures.committedAbovePlanned ? styles.problemText : styles.detail}>
              {figures.committedPercent}% of planned cost committed
            </p>
          </>
        )}
      </div>
      <div className={styles.tile}>
        <p className={styles.label}>
          Invoiced
          <InfoTip label="Invoiced" text={TIPS.invoiced} />
        </p>
        <p className={styles.value}>{money(figures.invoiced)}</p>
      </div>
      <div className={styles.tile}>
        <p className={styles.label}>
          Paid
          <InfoTip label="Paid" text={TIPS.paid} />
        </p>
        <p className={styles.value}>{money(figures.paid)}</p>
      </div>
      <div className={styles.tile}>
        <p className={styles.label}>
          {accrued}
          <InfoTip
            label="Accrued"
            text={`Money owed for work received but not yet invoiced, as GSAP accrued it for ${accrualMonth ? monthText(accrualMonth) : 'the latest month'}.`}
          />
        </p>
        <p className={styles.value}>{money(figures.accrued)}</p>
      </div>
      <div className={styles.tile}>
        <p className={styles.label}>
          Not yet committed
          <InfoTip label="Not yet committed" text={TIPS.notYetCommitted} />
        </p>
        <p className={negative ? `${styles.value} ${styles.problemText}` : styles.value}>
          {money(figures.notYetCommitted)}
        </p>
      </div>
    </div>
  );
}

/** The red warnings in words (015 FR-013). */
function Warnings({ figures, planId }: { figures: FinancialFigures; planId: string }) {
  const { money } = useFormatters();
  const over =
    figures.committedAbovePlanned && figures.notYetCommitted
      ? money({
          amount: figures.notYetCommitted.amount.replace(/^-/, ''),
          currency: figures.notYetCommitted.currency,
        })
      : null;
  const first = figures.firstLineItemAbovePlanned;
  if (!over && figures.lineItemsAbovePlanned === 0) {
    return null;
  }
  return (
    <div className={styles.warnings}>
      {over && (
        <p className={styles.warning}>
          <TriangleAlert aria-hidden="true" />
          Committed above planned cost by {over}
        </p>
      )}
      {figures.lineItemsAbovePlanned > 0 && (
        <p className={styles.warning}>
          <TriangleAlert aria-hidden="true" />
          {first ? (
            <Link to={recordHref(first.type, first.id, planId)}>
              Committed above planned cost on {figures.lineItemsAbovePlanned}{' '}
              {figures.lineItemsAbovePlanned === 1 ? 'line item' : 'line items'}
            </Link>
          ) : (
            <>
              Committed above planned cost on {figures.lineItemsAbovePlanned}{' '}
              {figures.lineItemsAbovePlanned === 1 ? 'line item' : 'line items'}
            </>
          )}
        </p>
      )}
    </div>
  );
}
