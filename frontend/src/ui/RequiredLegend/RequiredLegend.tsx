import { useLayoutEffect, useRef, useState } from 'react';
import styles from './RequiredLegend.module.css';

/**
 * "* Required" (038 FR-064, D-358): shown when its form, or its dialog without a form, has a field
 * required to save, and only by the first legend in it, so a form shows it once. FormErrorSummary,
 * FormDialog and ConfirmDialog place it before their fields. Screen readers skip it: they hear
 * "required" with each such field.
 */
export function RequiredLegend() {
  const legend = useRef<HTMLParagraphElement>(null);
  const [shown, setShown] = useState(false);

  useLayoutEffect(() => {
    const scope =
      legend.current?.closest('form') ??
      legend.current?.closest('[role="dialog"], [role="alertdialog"]');
    if (!scope) {
      return;
    }
    const check = () =>
      setShown(
        scope.querySelector('[data-required-legend]') === legend.current &&
          scope.querySelector('[aria-required="true"]') !== null,
      );
    check();
    // Fields come and go with the form's choices; the legend follows them.
    const observer = new MutationObserver(check);
    observer.observe(scope, {
      subtree: true,
      childList: true,
      attributes: true,
      attributeFilter: ['aria-required'],
    });
    return () => observer.disconnect();
  }, []);

  return (
    <p
      ref={legend}
      data-required-legend=""
      hidden={!shown}
      aria-hidden="true"
      className={styles.legend}
    >
      {shown && (
        <>
          <span className={styles.asterisk}>*</span> Required
        </>
      )}
    </p>
  );
}
