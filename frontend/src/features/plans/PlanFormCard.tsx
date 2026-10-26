import type { ReactNode } from 'react';
import { FormSection } from '@/ui/FormSection/FormSection';
import styles from './PlanForm.module.css';
import type { SectionId } from './plan-form-sections';

/** One section of the form as a white card with an anchor for the rail (design system §5.5). */
export function PlanFormCard({
  id,
  title,
  children,
}: {
  id: SectionId;
  title: string;
  children: ReactNode;
}) {
  return (
    <div id={id} className={styles.card}>
      <FormSection title={title}>{children}</FormSection>
    </div>
  );
}
