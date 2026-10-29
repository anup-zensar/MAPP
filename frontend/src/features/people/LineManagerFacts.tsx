import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import styles from '@/ui/settings-table.module.css';
import { usePeople } from './api';
import type { PersonDetails } from './api';
import { reportsText } from './texts';

// The details answer has no manager, so it is read from the people list on the same page.
export function LineManagerFacts({ person }: { person: PersonDetails }) {
  const people = usePeople();
  const manager = people.data?.items.find((candidate) => candidate.id === person.id)?.manager;
  const reports = reportsText(person.usage);
  return (
    <>
      <div>
        <dt>Line manager</dt>
        <dd>
          {manager ? (
            <PersonLabel person={manager} />
          ) : (
            <span className={styles.muted}>No line manager</span>
          )}
        </dd>
      </div>
      {reports && (
        <div>
          <dt>Line manager of</dt>
          <dd>{reports}</dd>
        </div>
      )}
    </>
  );
}
