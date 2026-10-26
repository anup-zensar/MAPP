import { cx } from '@/lib/cx';
import styles from './LevelTag.module.css';

/** The planning levels (vocabulary: Plan → Campaign → Initiative → Activity → Line item). */
export type PlanningLevel = 'plan' | 'campaign' | 'initiative' | 'activity' | 'lineItem';

const WORDS: Record<PlanningLevel, string> = {
  plan: 'PLAN',
  campaign: 'CAMPAIGN',
  initiative: 'INITIATIVE',
  activity: 'ACTIVITY',
  lineItem: 'LINE ITEM',
};

export interface LevelTagProps {
  level: PlanningLevel;
}

/**
 * A record's level as a small tag on a tint of the level's colour (design system §4): "PLAN",
 * "CAMPAIGN", "INITIATIVE", "ACTIVITY", "LINE ITEM". The word carries the meaning.
 */
export function LevelTag({ level }: LevelTagProps) {
  return <span className={cx(styles.tag, styles[level])}>{WORDS[level]}</span>;
}
