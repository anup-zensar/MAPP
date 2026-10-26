import type { components } from '@/api/generated/schema';
import styles from '@/ui/settings-table.module.css';

type ImpactWarning = components['schemas']['ImpactWarning'];

/** What a change would affect: one warning as a sentence, several as a list. */
export function ImpactWarnings({ warnings }: { warnings: ImpactWarning[] }) {
  const [only, ...others] = warnings;
  if (only && others.length === 0) {
    return <p>{only.message}</p>;
  }
  return (
    <ul className={styles.warnings}>
      {warnings.map((warning) => (
        <li key={warning.message}>{warning.message}</li>
      ))}
    </ul>
  );
}
