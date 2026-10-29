import type { ReactNode } from 'react';
import { Link } from 'react-router';
import { PurchaseRequestStatusChip } from '@/features/purchase-requests';
import { useFormatters } from '@/format/useFormatters';
import { Card } from '@/ui/Card/Card';
import { InfoTip } from '@/ui/InfoTip/InfoTip';
import { purchaseRequestHref } from '@/ui/record-links';
import type { SpaView } from './api';
import { BUDGET, PROCUREMENT, percentText } from './spa-text';
import styles from './Spa.module.css';

function Fact({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className={styles.fact}>
      <dt className={styles.label}>{label}</dt>
      <dd className={styles.factValue}>{children}</dd>
    </div>
  );
}

/** The scope's budget against planned cost, and how much is linked to finance budget items (013 FR-020). */
export function SpaBudget({ view }: { view: SpaView }) {
  const { money } = useFormatters();
  const b = view.budget;
  return (
    <Card title={BUDGET.title} actions={<InfoTip label={BUDGET.title} text={BUDGET.tip} />}>
      <dl className={styles.facts}>
        <Fact label={BUDGET.budget}>
          {b.budget ? money(b.budget) : `— · ${b.noBudgetReason ?? ''}`}
        </Fact>
        <Fact label="Planned cost">{money(b.plannedCost)}</Fact>
        <Fact label="Committed">{money(b.committed)}</Fact>
        <Fact label={BUDGET.plannedVsBudget}>
          <span className={b.overBudget ? styles.problem : undefined}>
            {percentText(b.plannedVsBudgetPercent, true)}
            {b.overBudget && ` · ${BUDGET.overBudget}`}
          </span>
        </Fact>
        <Fact label={BUDGET.linked}>{money(b.linked)}</Fact>
        <Fact label={BUDGET.notLinked}>{money(b.notLinked)}</Fact>
      </dl>
    </Card>
  );
}

/** The purchase requests on the scope's line items and their amounts by status (013 FR-021). */
export function SpaProcurement({ view }: { view: SpaView }) {
  const { money } = useFormatters();
  const p = view.procurement;
  return (
    <Card
      title={PROCUREMENT.title}
      actions={<InfoTip label={PROCUREMENT.title} text={PROCUREMENT.tip} />}
    >
      <dl className={styles.facts}>
        <Fact label={PROCUREMENT.count}>{p.count}</Fact>
        <Fact label={PROCUREMENT.approved}>{money(p.approved)}</Fact>
        <Fact label={PROCUREMENT.pendingApproval}>{money(p.pendingApproval)}</Fact>
        <Fact label={PROCUREMENT.draft}>{money(p.draft)}</Fact>
        <Fact label={PROCUREMENT.invoiced}>{money(p.invoiced)}</Fact>
      </dl>
      {p.items.length === 0 ? (
        <p className={styles.note}>{PROCUREMENT.none}</p>
      ) : (
        <ul className={styles.prList}>
          {p.items.map((pr) => (
            <li key={pr.id} className={styles.prItem}>
              <Link to={purchaseRequestHref(pr.id)}>{pr.name}</Link>
              <span className={styles.muted}>
                {pr.id}
                {pr.vendor && ` · ${pr.vendor}`}
              </span>
              <PurchaseRequestStatusChip status={pr.status} />
              <span className={styles.numeric}>
                {money(pr.amount)}
                {!pr.counted && ` · ${PROCUREMENT.notCounted}`}
              </span>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}
