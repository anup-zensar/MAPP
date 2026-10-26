import { House, MapPinOff } from 'lucide-react';
import { LinkButton } from '@/components';
import { usePageTitle } from '@/lib/usePageTitle';
import { CentredMessage } from './CentredMessage';

/**
 * Shown for an address MAPP does not know (FR-015, US2 scenario 6), with a way back to
 * Home. The not-found route puts it in the layout that fits the visitor.
 */
export function NotFoundPage() {
  usePageTitle('Page not found');

  return (
    <CentredMessage
      icon={<MapPinOff />}
      title="Page not found"
      action={
        <LinkButton to="/" variant="primary" icon={<House />}>
          Go to Home
        </LinkButton>
      }
    >
      <p>There&apos;s no page at this address. Check the address, or go back to Home.</p>
    </CentredMessage>
  );
}
