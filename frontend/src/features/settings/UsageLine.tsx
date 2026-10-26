import { Link } from 'react-router';
import type { components } from '@/api/generated/schema';
import { isPlanningType, recordHref } from '@/ui/record-links';
import styles from '@/ui/settings-table.module.css';

type Usage = components['schemas']['Usage'];

/**
 * Where a Settings record is used (FR-003): "Used by 2 plans · Allowed in 3 models", or "Not used
 * yet", then a link to each plan using it (models and finance budgets).
 */
export function UsageLine({ usage }: { usage: Usage }) {
  const links = usage.records.flatMap(({ type, id, name }) =>
    isPlanningType(type) ? [{ type, id, name }] : [],
  );

  return (
    <div className={styles.usage}>
      <p>{usage.summary}</p>
      {links.length > 0 && (
        <ul role="list" className={styles.usageLinks}>
          {links.map((record) => (
            <li key={`${record.type}-${record.id}`}>
              <Link to={recordHref(record.type, record.id)}>{record.name}</Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
