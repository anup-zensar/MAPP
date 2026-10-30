import { useLayoutEffect, useRef, useState } from 'react';
import styles from './RequiredLegend.module.css';

/** "* Required", shown once per form by its first legend; screen readers skip it. */
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
