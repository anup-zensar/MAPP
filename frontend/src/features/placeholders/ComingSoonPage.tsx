import { House } from 'lucide-react';
import type { ComingSoonSection } from '@/app/navigation';
import { LinkButton } from '@/components';
import { usePageTitle } from '@/lib/usePageTitle';
import { CentredMessage } from './CentredMessage';
import styles from './ComingSoonPage.module.css';

export interface ComingSoonPageProps {
  /** The section a later feature builds (navigation.ts). */
  section: ComingSoonSection;
}

/**
 * The page of a section that a later feature builds (FR-015): its name, one sentence on what it
 * will do, and a way back to Home — never an error. It has the same centred layout as Page not
 * found (UI audit M-02).
 */
export function ComingSoonPage({ section }: ComingSoonPageProps) {
  usePageTitle(section.label);

  return (
    <CentredMessage
      icon={<section.icon />}
      title={section.label}
      action={
        <LinkButton to="/" variant="primary" icon={<House />}>
          Go to Home
        </LinkButton>
      }
    >
      <p>{section.comingSoon}</p>
      <p className={styles.soon}>Coming soon</p>
      <p>This part of NorthStar isn&apos;t built yet.</p>
    </CentredMessage>
  );
}
