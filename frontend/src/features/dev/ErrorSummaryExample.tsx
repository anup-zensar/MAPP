import { useState } from 'react';
import { Button } from '@/ui/Button/Button';
import { FormErrorSummary } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import styles from './ComponentGalleryPage.module.css';

/**
 * The problems the error summary lists. One list for the whole page: the summary takes the focus
 * whenever it is given a new list, so a list made anew on every render would keep taking it.
 */
const SUMMARY_ERRORS: FieldProblem[] = [
  { field: 'campaignName', message: 'Enter a name.' },
  { field: 'budgetOwner', message: 'Choose a budget owner.' },
];

/**
 * The error summary appears when asked for: it takes the focus when it appears, which on page
 * load would move the focus, and the scroll position, away from the top of the page.
 */
export function ErrorSummaryExample() {
  const [shown, setShown] = useState(false);

  return (
    <div className={styles.stack}>
      <div>
        <Button onClick={() => setShown(true)}>Show the error summary</Button>
      </div>
      {shown && <FormErrorSummary errors={SUMMARY_ERRORS} />}
    </div>
  );
}
