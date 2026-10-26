import { TriangleAlert } from 'lucide-react';
import { Link } from 'react-router';
import styles from './ActivityForm.module.css';

export function NoTypesText({ model }: { model: string }) {
  return (
    <>
      The model &apos;{model}&apos; has no activity types yet — an administrator adds them in
      Settings. <Link to="/settings/activity-types">Open Settings</Link>
    </>
  );
}

/** The edge case of a model without activity types, said at the top of the form too. */
export function NoTypesNotice({ model }: { model: string }) {
  return (
    <p role="note" className={styles.notice}>
      <TriangleAlert aria-hidden="true" />
      <span>
        <NoTypesText model={model} />
      </span>
    </p>
  );
}
