import { Pencil } from 'lucide-react';
import type { ReactNode } from 'react';
import { useNavigate, useParams } from 'react-router';
import { useAttributeForm } from '@/api/settingsQueries';
import {
  Breadcrumb,
  Card,
  CompletenessChip,
  ErrorState,
  LevelTag,
  LinkButton,
  PageHeader,
  ShellAttributeValues,
  Skeleton,
} from '@/components';
import { formatRate, toCents } from '@/lib/decimal';
import { localDate } from '@/lib/format';
import { fieldHref, recordHref } from '@/lib/recordLinks';
import { useFormatters } from '@/lib/useFormatters';
import { usePageTitle } from '@/lib/usePageTitle';
import { FromFinanceSection } from './FromFinanceSection';
import styles from './LineItemPage.module.css';
import { activityBreadcrumb, activityLineItemsHref, planIdOf } from './lineItemLinks';
import { centsInput, formatMonth, moneyText } from './lineItemMath';
import { LineItemLoadFailed, LineItemLoading } from './LineItemPageStates';
import { useLineItem, usePlanBusinessUnit } from './lineItemsApi';
import type { LineItem, LineItemActivity } from './lineItemsApi';
import {
  bookkeepingText,
  conversionText,
  currencyLabel,
  customTotalText,
  DELETE,
  DUPLICATE,
  EDIT,
  EMPTY,
  EYEBROW,
  inactiveItemText,
  LABEL,
  NOT_SPREAD_YET,
  OUTSIDE_TIP,
  previewLine,
  SECTION_BASICS,
  SECTION_BUDGET,
  SECTION_COST,
  SECTION_FINANCE_CODING,
  SECTION_GSAP,
  SECTION_SPREAD,
  SPREAD_LABELS,
  SWBS_LATER,
  VIEW_HISTORY,
} from './lineItemText';
import { MoreMenu } from './MoreMenu';
import { SwbsChip } from './SwbsChip';
import { useLineItemActions } from './useLineItemActions';

/**
 * A line item's details (US4, FR-026), at /line-items/:lineItemId: where it lives, its chips, the
 * facts, who created and changed it, Edit and More (Duplicate, View history, Delete), then every
 * value read-only in the form's order, "—" for empty ones.
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

  const { lineItem, activity } = details.data;
  const href = recordHref('lineItem', lineItem.id);

  return (
    <>
      <Breadcrumb items={[...activityBreadcrumb(activity), { label: lineItem.name }]} />
      <PageHeader
        eyebrow={EYEBROW}
        title={lineItem.name}
        actions={
          <>
            <LinkButton to={`${href}/edit`} variant="primary" icon={<Pencil />}>
              {EDIT}
            </LinkButton>
            <MoreMenu
              items={[
                { label: DUPLICATE, onSelect: () => actions.duplicate(lineItem, activity) },
                {
                  label: VIEW_HISTORY,
                  onSelect: () =>
                    void navigate(
                      `/activity-log?${new URLSearchParams({ recordType: 'lineItem', recordId: lineItem.id })}`,
                    ),
                },
                {
                  label: DELETE,
                  tone: 'danger',
                  separatorBefore: true,
                  onSelect: () => actions.remove(lineItem, activity),
                },
              ]}
            />
          </>
        }
      />
      <Summary
        lineItem={lineItem}
        onSelectField={(field) => void navigate(fieldHref('lineItem', lineItem.id, field))}
      />
      <Values lineItem={lineItem} activity={activity} />
      {actions.dialogs}
    </>
  );
}

function Summary({
  lineItem,
  onSelectField,
}: {
  lineItem: LineItem;
  onSelectField: (field: string) => void;
}) {
  const { date } = useFormatters();
  return (
    <div className={styles.summary}>
      <div className={styles.chips}>
        <LevelTag level="lineItem" />
        <CompletenessChip
          missing={lineItem.missing.map((field) => ({ id: field.field, label: field.label }))}
          onSelectField={onSelectField}
        />
        <SwbsChip status={lineItem.swbsStatus} />
      </div>
      <div className={styles.factsRow}>
        <Fact label="ID">{lineItem.id}</Fact>
        <Fact label={LABEL.owner}>{lineItem.owner?.name ?? EMPTY}</Fact>
        <Fact label={LABEL.plannedCost}>{costText(lineItem)}</Fact>
        <Fact label={LABEL.spread}>{SPREAD_LABELS[lineItem.spread.method]}</Fact>
      </div>
      <p className={styles.bookkeeping}>
        {bookkeepingText(
          lineItem.createdBy.name,
          date(localDate(lineItem.createdAt)),
          lineItem.updatedBy.name,
          date(localDate(lineItem.updatedAt)),
        )}
      </p>
    </div>
  );
}

function Values({ lineItem, activity }: { lineItem: LineItem; activity: LineItemActivity }) {
  const attributeForm = useAttributeForm('lineItem');
  const planId = planIdOf(activity);
  const businessUnit = usePlanBusinessUnit(planId);
  const item = lineItem.financeBudgetItem;
  const { spread } = lineItem;

  return (
    <div className={styles.sections}>
      <Section title={SECTION_BASICS}>
        <Fact label={LABEL.name}>{lineItem.name}</Fact>
        <Fact label={LABEL.owner}>
          {lineItem.owner
            ? `${lineItem.owner.name}${lineItem.owner.active ? '' : ' (inactive)'}`
            : EMPTY}
        </Fact>
        <Fact label={LABEL.description}>{lineItem.description || EMPTY}</Fact>
      </Section>

      <Section title={SECTION_COST}>
        <Fact label={LABEL.currency}>
          {currencyLabel(lineItem.currency.code, lineItem.currency.name, lineItem.currency.active)}
        </Fact>
        <Fact label={LABEL.plannedCost}>{costText(lineItem)}</Fact>
      </Section>

      <Section title={SECTION_SPREAD}>
        <Fact label={LABEL.spread}>{SPREAD_LABELS[spread.method]}</Fact>
        {spread.state === 'notSpread' ? (
          <p className={styles.note}>{NOT_SPREAD_YET}</p>
        ) : (
          <ul className={styles.months} aria-label="Month amounts">
            {spread.months.map((month) => (
              <li key={month.month}>
                {previewLine(formatMonth(month.month), moneyText(month.amount))}
                {month.outsideActivityDates && (
                  <span className={styles.outside}> — {OUTSIDE_TIP}</span>
                )}
              </li>
            ))}
          </ul>
        )}
        {spread.method === 'custom' && spread.customTotal && (
          <p className={styles.note}>{customTotal(lineItem)}</p>
        )}
      </Section>

      <Section title={SECTION_FINANCE_CODING}>
        {attributeForm.data ? (
          <ShellAttributeValues
            form={attributeForm.data}
            values={lineItem.attributes}
            businessUnit={businessUnit.data ?? []}
            planLink={planId ? recordHref('plan', planId) : undefined}
          />
        ) : attributeForm.isError ? (
          <ErrorState error={attributeForm.error} onRetry={() => void attributeForm.refetch()} />
        ) : (
          <Skeleton lines={2} />
        )}
        <Fact label={LABEL.contractedAmount}>
          {lineItem.contractedAmount ? moneyText(lineItem.contractedAmount) : EMPTY}
        </Fact>
      </Section>

      <Section title={SECTION_BUDGET}>
        <Fact label={LABEL.financeBudgetYear}>{lineItem.financeBudgetYear ?? EMPTY}</Fact>
        <Fact label={LABEL.financeBudgetItem}>
          {item
            ? `${[item.name, item.budgetName, item.code].filter(Boolean).join(' · ')}${item.active ? '' : ' (inactive)'}`
            : EMPTY}
        </Fact>
        {item && !item.active && <p className={styles.problem}>{inactiveItemText(item.name)}</p>}
      </Section>

      <Section title={SECTION_GSAP}>
        <Fact label={LABEL.requestSwbs}>{lineItem.requestSwbs ? 'Yes' : 'No'}</Fact>
        <Fact label={LABEL.swbsCode}>{SWBS_LATER}</Fact>
      </Section>

      <Card>
        <FromFinanceSection />
      </Card>
    </div>
  );
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <Card title={title}>
      <div className={styles.facts}>{children}</div>
    </Card>
  );
}

function Fact({ label, children }: { label: string; children: ReactNode }) {
  return (
    <dl className={styles.fact}>
      <dt>{label}</dt>
      <dd>{children}</dd>
    </dl>
  );
}

/** "46,000.00 EUR = 50,000.00 USD at 0.92 EUR per USD", or "50,000.00 USD". */
function costText(lineItem: LineItem): string {
  const local = moneyText(lineItem.plannedCost);
  if (lineItem.currency.code === 'USD') {
    return local;
  }
  return `${local} ${conversionText(
    centsInput(toCents(lineItem.plannedCostUsd.amount)),
    formatRate(lineItem.currency.ratePerUsd),
    lineItem.currency.code,
  )}`;
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
