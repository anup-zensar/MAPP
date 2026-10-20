import styles from './PlanTypeTag.module.css';
import { PLAN_TYPE_LABELS, planTypeOf } from './planCurrency';
import type { PlanType } from './planCurrency';

/**
 * The plan type as a quiet tag (016 FR-032): "Operational", "Strategic", "Cross-market". It is a
 * fact about the plan, not a status, so it is outlined and never counts as one of a row's two
 * status chips (design system §4.1).
 */
export function PlanTypeTag({ plan }: { plan: { planType?: PlanType } }) {
  return <span className={styles.tag}>{PLAN_TYPE_LABELS[planTypeOf(plan)]}</span>;
}
