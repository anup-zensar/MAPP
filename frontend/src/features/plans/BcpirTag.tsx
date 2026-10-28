import styles from './PlanTypeTag.module.css';

/** The BCPIR marker of a plan (013 FR-002): a fact about the plan, outlined like the plan type. */
export function BcpirTag({ plan }: { plan: { bcpir?: boolean } }) {
  if (!plan.bcpir) {
    return null;
  }
  return <span className={styles.tag}>BCPIR</span>;
}
