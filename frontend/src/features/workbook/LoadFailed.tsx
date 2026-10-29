import { forbiddenMessage } from '@/session/access';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { NoAccess } from '@/ui/NoAccess/NoAccess';

/** A page whose data did not load: the no-access page for a refusal, otherwise Try again. */
export function LoadFailed({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  const refused = forbiddenMessage(error);
  if (refused) return <NoAccess message={refused} />;
  return <ErrorState error={error} onRetry={onRetry} />;
}
