import { House, MapPinOff } from 'lucide-react';
import { LinkButton } from '@/components';
import { usePageTitle } from '@/lib/usePageTitle';
import styles from './NotFoundPage.module.css';

/**
 * Shown for an address NorthStar does not know (FR-015, US2 scenario 6), with a way back to
 * Home. The not-found route puts it in the layout that fits the visitor.
 */
export function NotFoundPage() {
  usePageTitle('Page not found');

  return (
    <div className={styles.page}>
      <div className={styles.icon}>
        <MapPinOff />
      </div>
      <h1 className={styles.title}>Page not found</h1>
      <p className={styles.text}>
        There's no page at this address. Check the address, or go back to Home.
      </p>
      <LinkButton to="/" variant="primary" icon={<House />}>
        Go to Home
      </LinkButton>
    </div>
  );
}
