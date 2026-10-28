import { CircleAlert } from 'lucide-react';
import { ApiError } from '@/api/errors';
import { LinkButton } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { forbiddenMessage } from '@/session/access';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { NoAccess } from '@/ui/NoAccess/NoAccess';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { ACTIVITY_GONE, GO_TO_BIN, GO_TO_PLANS } from './line-item-text';

// The breadcrumb row is kept while loading, so the title never moves.
function Header({ title }: { title: string }) {
  return (
    <PageHeader breadcrumb={[{ label: 'Plans', to: '/plans' }, { label: title }]} title={title} />
  );
}

export function LineItemLoading({ title }: { title: string }) {
  return (
    <>
      <Header title={title} />
      <Card>
        <Skeleton lines={6} />
      </Card>
    </>
  );
}

// Gone (D-154): the API's message with Go to the recycle bin or Go to Plans; refused by the
// person's roles: the no-access page; anything else: Try again.
export function LineItemLoadFailed({
  title,
  error,
  onRetry,
}: {
  title: string;
  error: unknown;
  onRetry: () => void;
}) {
  const refused = forbiddenMessage(error);
  if (refused) {
    return <NoAccess message={refused} />;
  }
  const gone = error instanceof ApiError && error.kind === 'notFound';
  return (
    <>
      <Header title={title} />
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
