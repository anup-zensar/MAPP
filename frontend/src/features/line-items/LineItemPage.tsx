import { Pencil } from 'lucide-react';
import { useNavigate, useParams } from 'react-router';
import { useAttributeForm } from '@/api/settingsQueries';
import {
  ActionsMenu,
  Card,
  CompletenessChip,
  ErrorState,
  LevelTag,
  LinkButton,
  PageHeader,
  PersonLabel,
  ShellAttributeValues,
  Skeleton,
  SummaryCard,
} from '@/components';
import { HistorySection } from '@/features/activity-log/HistorySection';
import { showHistory } from '@/features/activity-log/showHistory';
import { formatRate, toCents } from '@/lib/decimal';
import { localDate } from '@/lib/format';
import { fieldHref, recordHref } from '@/lib/recordLinks';
import { useFormatters } from '@/lib/useFormatters';
import { usePageTitle } from '@/lib/usePageTitle';
import { Detail, DetailsList } from './DetailsList';
import { LineItemFinancials } from './FinancialsSection';
import styles from './LineItemPage.module.css';
import { activityBreadcrumb, activityLineItemsHref, planIdOf } from './lineItemLinks';
import { centsInput, formatMonth, moneyText } from './lineItemMath';
import { LineItemLoadFailed, LineItemLoading } from './LineItemPageStates';
import { useLineItem } from './lineItemsApi';
import type { LineItem, LineItemActivity, UnassignedBalance } from './lineItemsApi';
import {
  bookkeepingText,
  conversionText,
  currencyLabel,
  customTotalText,
  DELETE,
  DUPLICATE,
  EDIT,
  inactiveItemText,
  LABEL,
  NOT_SPREAD_YET,
  OUTSIDE_TIP,
  SECTION_SPREAD,
  SPREAD_LABELS,
  VIEW_HISTORY,
} from './lineItemText';
import { RateTip } from './MoneyPair';
import { SwbsChip } from './SwbsChip';
import { useLineItemActions } from './useLineItemActions';

/** The activity's unassigned planned cost, as the line item page's third summary card says it. */
const UNASSIGNED_LABEL = "Activity's unassigned planned cost";
const UNASSIGNED_TIP =
  "The activity's planned cost minus all its line items' planned costs (Sys_Gen_Balance). Aim for 0.00.";

/**
 * A line item's page (US4, FR-026), at /line-items/:lineItemId, on the record template of design
 * system §2.4 B and as wide as its activity's page (UI audit LI-01): the header with the level tag,
 * completeness and SWBS chips, ID and owner in the meta row, ⋯ (Duplicate, View history, Delete)
 * and Edit; the summary cards (planned cost, spread, the activity's unassigned planned cost); the
 * Details and Finance coding cards, each value in the one key–value layout ("—" when empty); GSAP's
 * Financials (feature 015); the spread; then its History (feature 011).
 */
export function LineItemPage() {
  const { lineItemId = '' } = useParams();
  const navigate = useNavigate();
  const details = useLineItem(lineItemId);
  const actions = useLineItemActions((activity) => navigate(activityLineItemsHref(activity)));
  usePageTitle(details.data?.lineItem.name ?? 'Line item');

  if (details.isError) {
    return (
      <LineItemLoadFailed
        title="Line item"
        error={details.error}
        onRetry={() => void details.refetch()}
      />
    );
  }
  if (!details.data) {
    return <LineItemLoading title="Line item" />;
  }

  const { lineItem, activity, balance } = details.data;
  const href = recordHref('lineItem', lineItem.id);

  return (
    <>
      <PageHeader
        breadcrumb={[...activityBreadcrumb(activity), { label: lineItem.name }]}
        title={lineItem.name}
        meta={
          <>
            <LevelTag level="lineItem" />
            <CompletenessChip
              missing={lineItem.missing.map((field) => ({ id: field.field, label: field.label }))}
              onSelectField={(field) => void navigate(fieldHref('lineItem', lineItem.id, field))}
            />
            <SwbsChip status={lineItem.swbsStatus} />
            <span className={styles.fact}>{lineItem.id}</span>
            {lineItem.owner ? (
              <PersonLabel person={lineItem.owner} />
            ) : (
              <span className={styles.fact}>No owner yet</span>
            )}
          </>
        }
        actions={
          <>
            {/* ⋯ first, the primary last (design system §2.3, UI audit LI-02). */}
            <ActionsMenu
              label={lineItem.name}
              items={[
                { label: DUPLICATE, onSelect: () => actions.duplicate(lineItem, activity) },
                // Brings the History section below into view and focuses it (feature 011).
                { label: VIEW_HISTORY, onSelect: () => showHistory() },
                {
                  label: DELETE,
                  tone: 'danger',
                  separatorBefore: true,
                  onSelect: () => actions.remove(lineItem, activity),
                },
              ]}
            />
            <LinkButton to={`${href}/edit`} variant="primary" icon={<Pencil />}>
              {EDIT}
            </LinkButton>
          </>
        }
      />
      <div className={styles.page}>
        <Figures lineItem={lineItem} balance={balance} />
        <div className={styles.columns}>
          <DetailsCard lineItem={lineItem} />
          <FinanceCard lineItem={lineItem} activity={activity} />
        </div>
        {/* GSAP's money for this line item, open (015 FR-017). */}
        <LineItemFinancials lineItemId={lineItem.id} />
        <SpreadCard lineItem={lineItem} />
        <HistorySection
          recordType="lineItem"
          recordId={lineItem.id}
          recordName={lineItem.name}
          level="lineItem"
        />
      </div>
      {actions.dialogs}
    </>
  );
}

/**
 * The summary cards: the planned cost in its own currency (with the plan currency and the reader's
 * currency beside it, and the rates used — 016 FR-009, FR-015), the spread and the activity's
 * unassigned planned cost.
 */
function Figures({ lineItem, balance }: { lineItem: LineItem; balance: UnassignedBalance }) {
  const { spread } = lineItem;
  const months = spread.months.filter((month) => toCents(month.amount.amount) !== 0n);
  const left = balance.balance;
  const negative = left !== null && toCents(left.amount) < 0n;
  return (
    <section className={styles.figures} aria-label="Summary">
      <SummaryCard
        label={LABEL.plannedCost}
        value={moneyText(lineItem.plannedCost)}
        detail={<CostBeside lineItem={lineItem} />}
      />
      <SummaryCard
        label={LABEL.spread}
        value={SPREAD_LABELS[spread.method]}
        detail={
          spread.state === 'notSpread'
            ? NOT_SPREAD_YET
            : months.length === 0
              ? 'Nothing in any month yet'
              : `${months.length === 1 ? '1 month' : `${months.length} months`}: ${formatMonth(
                  months[0].month,
                )}${months.length > 1 ? ` – ${formatMonth(months[months.length - 1].month)}` : ''}`
        }
      />
      <SummaryCard
        label={UNASSIGNED_LABEL}
        tip={UNASSIGNED_TIP}
        value={left ? moneyText(left) : '—'}
        tone={negative ? 'problem' : undefined}
        detail={
          left === null
            ? 'The activity has no planned cost yet'
            : negative
              ? 'Line items exceed the planned cost'
              : toCents(left.amount) === 0n
                ? 'All of the planned cost is assigned'
                : 'Not yet assigned to a line item'
        }
      />
    </section>
  );
}

/**
 * Under the planned cost: "= 46,000.00 GBP" in the plan currency and "≈ 50,000.00 USD" in USD, each
 * only when it differs, with the ⓘ naming the rates (016 FR-009, FR-015); without 016's plan amount,
 * "= 50,000.00 USD at 0.92 EUR per USD".
 */
function CostBeside({ lineItem }: { lineItem: LineItem }) {
  const local = lineItem.plannedCost;
  const plan = lineItem.plannedCostPlan;
  const usd = lineItem.plannedCostUsd;
  if (local.currency === 'USD' && (!plan || plan.currency === 'USD')) {
    return null;
  }
  const lines: string[] = [];
  if (plan && plan.currency !== local.currency) {
    lines.push(`= ${moneyText(plan)}`);
  }
  if (usd.currency !== local.currency && (!plan || plan.currency !== usd.currency)) {
    lines.push(
      plan
        ? `≈ ${moneyText(usd)}`
        : conversionText(
            centsInput(toCents(usd.amount)),
            formatRate(lineItem.currency.ratePerUsd),
            lineItem.currency.code,
          ),
    );
  }
  return (
    <span className={styles.beside}>
      {lines.join(' · ')}
      <RateTip codes={[local.currency, ...(plan ? [plan.currency] : [])]} />
    </span>
  );
}

function DetailsCard({ lineItem }: { lineItem: LineItem }) {
  const { date } = useFormatters();
  return (
    <Card title="Details">
      <div className={styles.stack}>
        <DetailsList>
          <Detail label={LABEL.name}>{lineItem.name}</Detail>
          <Detail label={LABEL.owner}>
            {lineItem.owner
              ? `${lineItem.owner.name}${lineItem.owner.active ? '' : ' (inactive)'}`
              : null}
          </Detail>
          <Detail label={LABEL.currency}>
            {currencyLabel(
              lineItem.currency.code,
              lineItem.currency.name,
              lineItem.currency.active,
            )}
          </Detail>
          <Detail label={LABEL.plannedCost} numeric>
            {moneyText(lineItem.plannedCost)}
          </Detail>
          <Detail label={LABEL.description}>
            {lineItem.description ? (
              <span className={styles.description}>{lineItem.description}</span>
            ) : null}
          </Detail>
        </DetailsList>
        <p className={styles.bookkeeping}>
          {bookkeepingText(
            lineItem.createdBy.name,
            date(localDate(lineItem.createdAt)),
            lineItem.updatedBy.name,
            date(localDate(lineItem.updatedAt)),
          )}
        </p>
      </div>
    </Card>
  );
}

/**
 * Finance coding: the line-item Shell attributes under their group's own name (no heading repeats
 * its parent, UI audit LI-03), then the contracted amount, the finance budget and the GSAP request
 * in the same key–value layout.
 */
function FinanceCard({ lineItem, activity }: { lineItem: LineItem; activity: LineItemActivity }) {
  const attributeForm = useAttributeForm('lineItem');
  const planId = planIdOf(activity);
  const item = lineItem.financeBudgetItem;
  return (
    <section aria-label="Finance coding" className={styles.financeCard}>
      {attributeForm.data ? (
        <ShellAttributeValues
          form={attributeForm.data}
          values={lineItem.attributes}
          businessUnit={activity.businessUnitOptionIds}
          planLink={planId ? recordHref('plan', planId) : undefined}
          sectionHeading="h2"
        />
      ) : (
        <>
          <h2>Finance coding</h2>
          {attributeForm.isError ? (
            <ErrorState error={attributeForm.error} onRetry={() => void attributeForm.refetch()} />
          ) : (
            <Skeleton lines={2} />
          )}
        </>
      )}
      <DetailsList>
        <Detail label={LABEL.contractedAmount} numeric>
          {lineItem.contractedAmount ? moneyText(lineItem.contractedAmount) : null}
        </Detail>
        <Detail label={LABEL.financeBudgetYear}>{lineItem.financeBudgetYear}</Detail>
        <Detail label={LABEL.financeBudgetItem}>
          {item
            ? `${[item.name, item.budgetName, item.code].filter(Boolean).join(' · ')}${item.active ? '' : ' (inactive)'}`
            : null}
        </Detail>
        <Detail label={LABEL.requestSwbs}>{lineItem.requestSwbs ? 'Yes' : 'No'}</Detail>
      </DetailsList>
      {item && !item.active && <p className={styles.problem}>{inactiveItemText(item.name)}</p>}
    </section>
  );
}

/** The months of the spread, amounts right-aligned under their header (design system §5.4). */
function SpreadCard({ lineItem }: { lineItem: LineItem }) {
  const { spread } = lineItem;
  const code = lineItem.currency.code;
  return (
    <Card title={SECTION_SPREAD} subtitle={SPREAD_LABELS[spread.method]}>
      {spread.state === 'notSpread' ? (
        <p className={styles.note}>{NOT_SPREAD_YET}</p>
      ) : (
        <div className={styles.stack}>
          <table className={styles.months}>
            <caption className="visually-hidden">Month amounts</caption>
            <thead>
              <tr>
                <th scope="col">Month</th>
                <th scope="col" className={styles.number}>
                  {`Planned cost (${code})`}
                </th>
              </tr>
            </thead>
            <tbody>
              {spread.months.map((month) => (
                <tr key={month.month}>
                  <td>
                    {formatMonth(month.month)}
                    {month.outsideActivityDates && (
                      <span className={styles.outside}> — {OUTSIDE_TIP}</span>
                    )}
                  </td>
                  <td className={styles.number}>{moneyText(month.amount)}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {spread.method === 'custom' && spread.customTotal && (
            <p className={styles.note}>{customTotal(lineItem)}</p>
          )}
        </div>
      )}
    </Card>
  );
}

/** "12,000.00 of 10,000.00 EUR spread — 2,000.00 EUR too much" for a saved custom spread. */
function customTotal(lineItem: LineItem): string {
  const sum = lineItem.spread.customTotal;
  if (!sum) {
    return '';
  }
  const total = toCents(sum.amount);
  const cost = toCents(lineItem.plannedCost.amount);
  const difference = cost - total;
  return customTotalText(centsInput(total), centsInput(cost), lineItem.currency.code, {
    sign: difference === 0n ? 0 : difference > 0n ? 1 : -1,
    amount: centsInput(difference < 0n ? -difference : difference),
  });
}
