import { CircleAlert } from 'lucide-react';
import type { ReactNode } from 'react';
import { ApiError } from '@/api/errors';
import type { BreadcrumbItem } from '@/ui/Breadcrumb/Breadcrumb';
import { LinkButton } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { NoAccess } from '@/ui/NoAccess/NoAccess';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { GO_TO_PLANS } from './page-text';

const PLANS: BreadcrumbItem = { label: 'Plans', to: '/plans' };

function Frame({ title, children }: { title: string; children: ReactNode }) {
  return (
    <>
      <PageHeader breadcrumb={[PLANS, { label: title }]} title={title} />
      <Card>{children}</Card>
    </>
  );
}

export function FormPageLoading({ title }: { title: string }) {
  return (
    <Frame title={title}>
      <Skeleton lines={8} />
    </Frame>
  );
}

/** A load that failed: no access (403), gone or out of reach (404), or anything else with Try again. */
export function FormPageFailed({
  title,
  error,
  onRetry,
}: {
  title: string;
  error: unknown;
  onRetry: () => void;
}) {
  if (error instanceof ApiError && error.kind === 'forbidden') {
    return <NoAccess message={error.message} />;
  }
  return (
    <Frame title={title}>
      {error instanceof ApiError && error.kind === 'notFound' ? (
        <EmptyState
          icon={<CircleAlert />}
          title={error.message}
          action={
            <LinkButton to="/plans" variant="primary">
              {GO_TO_PLANS}
            </LinkButton>
          }
        />
      ) : (
        <ErrorState error={error} onRetry={onRetry} />
      )}
    </Frame>
  );
}

/** The edit form is not offered: the reason, in place of the form (032 FR-056). */
export function FormPageUnavailable({
  title,
  breadcrumb,
  reason,
  backTo,
  backLabel,
}: {
  title: string;
  breadcrumb: BreadcrumbItem[];
  reason: string;
  backTo: string;
  backLabel: string;
}) {
  return (
    <>
      <PageHeader breadcrumb={breadcrumb} title={title} />
      <Card>
        <EmptyState
          icon={<CircleAlert />}
          title={reason}
          action={
            <LinkButton to={backTo} variant="primary">
              {backLabel}
            </LinkButton>
          }
        />
      </Card>
    </>
  );
}
