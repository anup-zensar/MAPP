import type { ReactNode } from 'react';
import styles from './FormFooter.module.css';

export interface FormFooterProps {
  /** The form's buttons: Save (primary), Save and add another, Cancel (design system §6.2). */
  children: ReactNode;
}

/**
 * The sticky footer of a full-page form (feature 004; plans, activities, line items): its buttons
 * stay in view at the bottom of the window while the form scrolls. Save is never disabled.
 */
export function FormFooter({ children }: FormFooterProps) {
  return <div className={styles.footer}>{children}</div>;
}
