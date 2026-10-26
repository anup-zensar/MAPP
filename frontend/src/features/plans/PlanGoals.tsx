import { Card } from '@/ui/Card/Card';
import { InfoTip } from '@/ui/InfoTip/InfoTip';
import { formatWhole } from '@/lib/decimal';
import { useFormatters } from '@/lib/useFormatters';
import { goalRows } from './goalWaterfall';
import styles from './PlanGoals.module.css';
import type { Plan } from './plansApi';
import { STAGE_WORDS } from './planText';
import type { GoalStage } from './planText';

const STAGES: GoalStage[] = ['inq', 'mql', 'revenue'];

/**
 * The plan's goal beside what its activities plan (FR-016): INQ, MQL and Revenue, the typed stage
 * as typed and the others "Calculated" with their formula, and the share of the goal planned.
 */
export function PlanGoals({ plan }: { plan: Plan }) {
  const { money } = useFormatters();
  const { goals } = plan;
  const formulas = new Map(
    goalRows(plan.goalStage, plan.goalAmount ?? '', plan.model).map((row) => [
      row.stage,
      row.formula,
    ]),
  );

  function planGoal(stage: GoalStage) {
    const value = goals.plan[stage];
    const shown =
      value === null ? '—' : typeof value === 'number' ? formatWhole(value) : money(value);
    const reason = goals.plan.reasons[stage];
    return (
      <>
        <span>{shown}</span>
        {value === null && reason && <span className={styles.subtle}>{reason}</span>}
      </>
    );
  }

  /** The stage's name, and "Calculated" with its formula under it for the stages worked out. */
  function stageName(stage: GoalStage) {
    return (
      <span className={styles.stage}>
        <span>{STAGE_WORDS[stage]}</span>
        {stage !== goals.stage && goals.plan[stage] !== null && (
          <span className={styles.calculated}>
            Calculated
            <InfoTip label={`${STAGE_WORDS[stage]} goal`} text={formulas.get(stage) ?? ''} />
          </span>
        )}
      </span>
    );
  }

  const planned = {
    inq: formatWhole(goals.activities.inq),
    mql: formatWhole(goals.activities.mql),
    revenue: money(goals.activities.revenue),
  };

  return (
    <Card title="Goals" subtitle="The plan's goal beside what its activities plan">
      <table className={styles.goalsTable}>
        <thead>
          <tr>
            <th scope="col">Stage</th>
            <th scope="col" className={styles.number}>
              Plan goal
            </th>
            <th scope="col" className={styles.number}>
              In activities
            </th>
          </tr>
        </thead>
        <tbody>
          {STAGES.map((stage) => (
            <tr key={stage}>
              <th scope="row">{stageName(stage)}</th>
              <td className={styles.number}>
                <span className={styles.goalCell}>{planGoal(stage)}</span>
              </td>
              <td className={styles.number}>{planned[stage]}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <p className={styles.goalShare}>
        {plan.goalAmount === null
          ? 'No goal set yet — set a goal amount in Edit.'
          : goals.plannedPercent !== null &&
            `${goals.plannedPercent}% of the ${STAGE_WORDS[goals.stage]} goal planned`}
      </p>
    </Card>
  );
}
