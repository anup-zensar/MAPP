import { useCurrentUser } from '@/session/useSession';
import { useFormatters } from '@/format/useFormatters';
import { ApprovalAlert } from './ApprovalAlert';
import { useDelegations } from './delegations-api';

/**
 * The delegations in force today that concern the signed-in person (017 US4, FR-024), above the
 * approvals page's cards, one line each: "You are approving for Daniel Okoro until 14 Nov 2026."
 * or "Carlos Mendes approves for you until 14 Nov 2026." Nothing while none is current, while
 * they load, or when they could not be loaded (the panel says so).
 */
export function DelegationBanner() {
  const user = useCurrentUser();
  const { date } = useFormatters();
  const delegations = useDelegations('mine');
  const lines = (delegations.data?.items ?? [])
    .filter((delegation) => delegation.state === 'current')
    .flatMap((delegation) => {
      const until = date(delegation.endDate);
      if (delegation.delegate.id === user.id) {
        return [`You are approving for ${delegation.person.name} until ${until}.`];
      }
      if (delegation.person.id === user.id) {
        return [`${delegation.delegate.name} approves for you until ${until}.`];
      }
      return [];
    });

  if (lines.length === 0) {
    return null;
  }
  return (
    <ApprovalAlert tone="info">
      {lines.map((line) => (
        <p key={line}>{line}</p>
      ))}
    </ApprovalAlert>
  );
}
