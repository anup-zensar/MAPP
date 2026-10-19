import { Skeleton } from '@/components';
import { formatWhole } from '@/lib/decimal';
import styles from './SettingsCount.module.css';

export interface SettingsCountProps {
  /** How many records of the section are active; undefined while unknown. */
  count: number | undefined;
  /** The count is still on its way: a grey bar holds its place. */
  loading: boolean;
}

/**
 * "<n> active" at the foot of a Settings card (002 FR-001, plan "Settings cards"), the same for
 * 002's sections, whose counts come from the overview, and for the People, Teams (003) and Approval
 * workflows (008) cards, whose features count their own records: a grey bar while it loads,
 * nothing when it could not be loaded — the card stays a usable link.
 */
export function SettingsCount({ count, loading }: SettingsCountProps) {
  if (loading) {
    return <Skeleton width="64px" />;
  }
  if (count === undefined) {
    return null;
  }
  return <span className={styles.count}>{`${formatWhole(count)} active`}</span>;
}
