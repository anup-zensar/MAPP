import { formatWhole } from '@/format/decimal';
import { Skeleton } from '../Skeleton/Skeleton';
import styles from './SettingsCount.module.css';

export interface SettingsCountProps {
  /** How many records of the section are active; undefined while unknown. */
  count: number | undefined;
  /** The count is still on its way: a grey bar holds its place. */
  loading: boolean;
  /** How the count reads: "<n> active" unless given ("4 roles", "2 edited"). */
  text?: (count: number) => string;
}

/**
 * "<n> active" at the foot of a Settings card (002 FR-001, plan "Settings cards"), the same for
 * 002's sections, whose counts come from the overview, and for the People, Teams (003) and Approval
 * workflows (008) cards, whose features count their own records: a grey bar while it loads,
 * nothing when it could not be loaded — the card stays a usable link.
 */
export function SettingsCount({ count, loading, text }: SettingsCountProps) {
  if (loading) {
    return <Skeleton width="64px" />;
  }
  if (count === undefined) {
    return null;
  }
  return (
    <span className={styles.count}>{text ? text(count) : `${formatWhole(count)} active`}</span>
  );
}
