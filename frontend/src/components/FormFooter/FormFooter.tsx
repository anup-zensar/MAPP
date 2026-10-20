import type { ReactNode } from 'react';
import styles from './FormFooter.module.css';

export interface FormFooterProps {
  /**
   * The form's buttons in order of importance: Save (primary), Save and add another, Cancel
   * (design system §6.2). They line up from the right, so Save ends up rightmost.
   */
  children: ReactNode;
}

/**
 * The sticky footer of a full-page form (feature 004; plans, activities, line items): its buttons
 * stay in view at the bottom of the window while the form scrolls. Save is never disabled.
 */
export function FormFooter({ children }: FormFooterProps) {
  return <div className={styles.footer}>{children}</div>;
}
