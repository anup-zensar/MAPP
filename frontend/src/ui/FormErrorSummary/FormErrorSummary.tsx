import { useEffect, useRef } from 'react';
import { RequiredLegend } from '../RequiredLegend/RequiredLegend';
import styles from './FormErrorSummary.module.css';

export interface FieldProblem {
  field: string;
  message: string;
}

export interface FormErrorSummaryProps {
  errors: FieldProblem[];
  fieldId?: (field: string) => string;
}

/** Lists every problem as a link to its field and takes the focus, so the problems are read out. */
export function FormErrorSummary({ errors, fieldId = (field) => field }: FormErrorSummaryProps) {
  const summary = useRef<HTMLDivElement>(null);

  // A new list means a new failed attempt: bring the person here again.
  useEffect(() => {
    summary.current?.focus();
  }, [errors]);

  if (errors.length === 0) {
    return <RequiredLegend />;
  }

  const title =
    errors.length === 1
      ? '1 problem needs your attention'
      : `${errors.length} problems need your attention`;

  function goToField(field: string) {
    document.getElementById(fieldId(field))?.focus();
  }

  return (
    <>
      <div ref={summary} tabIndex={-1} className={styles.summary}>
        <div role="alert">
          <h2 className={styles.title}>{title}</h2>
          <ul className={styles.list}>
            {errors.map((error, index) => (
              <li key={`${error.field}-${index}`}>
                <a
                  href={`#${fieldId(error.field)}`}
                  onClick={(event) => {
                    // Move the focus ourselves: following the #link would change the page address.
                    event.preventDefault();
                    goToField(error.field);
                  }}
                >
                  {error.message}
                </a>
              </li>
            ))}
          </ul>
        </div>
      </div>
      <RequiredLegend />
    </>
  );
}
