import { useFormatters } from '@/format/useFormatters';
import { useSession } from '@/session/useSession';

/** What the rest of the app sees of the session: the account's name and a date on another page. */
export function ElsewhereInTheApp() {
  const { user } = useSession();
  const { date } = useFormatters();
  return <p>{`Account: ${user?.name} (${user?.initials}) · a date: ${date('2026-03-12')}`}</p>;
}
