import { CircleAlert } from 'lucide-react';
import { ApiError } from '@/api/errors';
import { Card, EmptyState, ErrorState, LinkButton, PageHeader, Skeleton } from '@/components';
import { ACTIVITY_GONE, EYEBROW, GO_TO_BIN, GO_TO_PLANS } from './lineItemText';

/** The page's header while it loads, or when what it shows is gone. */
export function LineItemLoading({ title }: { title: string }) {
  return (
    <>
      <PageHeader eyebrow={EYEBROW} title={title} />
      <Card>
        <Skeleton lines={6} />
      </Card>
    </>
  );
}

/**
 * A load that failed: the line item or its activity is gone (D-154) — the API's message with Go to
 * the recycle bin or Go to Plans — or another failure with Try again.
 */
export function LineItemLoadFailed({
  title,
  error,
  onRetry,
}: {
  title: string;
  error: unknown;
  onRetry: () => void;
}) {
  const gone = error instanceof ApiError && error.kind === 'notFound';
  return (
    <>
      <PageHeader eyebrow={EYEBROW} title={title} />
      <Card>
        {gone ? (
          <EmptyState
            icon={<CircleAlert />}
            title={error.message}
            action={
              error.message === ACTIVITY_GONE ? (
                <LinkButton to="/recycle-bin" variant="primary">
                  {GO_TO_BIN}
                </LinkButton>
              ) : (
                <LinkButton to="/plans" variant="primary">
                  {GO_TO_PLANS}
                </LinkButton>
              )
            }
          />
        ) : (
          <ErrorState error={error} onRetry={onRetry} />
        )}
      </Card>
    </>
  );
}
