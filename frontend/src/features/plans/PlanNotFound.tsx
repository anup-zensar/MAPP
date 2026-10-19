import { LinkButton, PageHeader } from '@/components';
import { NOT_FOUND_TEXT, NOT_FOUND_TITLE } from './planText';

/** The plan page or edit page of a plan that does not exist (any more). */
export function PlanNotFound() {
  return (
    <PageHeader
      eyebrow="Plans"
      title={NOT_FOUND_TITLE}
      description={NOT_FOUND_TEXT}
      actions={
        <LinkButton to="/plans" variant="primary">
          Go to Plans
        </LinkButton>
      }
    />
  );
}
