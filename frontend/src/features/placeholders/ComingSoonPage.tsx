import { House } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { LinkButton } from '@/ui/Button/Button';
import { usePageTitle } from '@/ui/usePageTitle';
import { CentredMessage } from './CentredMessage';
import styles from './ComingSoonPage.module.css';

export interface ComingSoonSection {
  label: string;
  icon: LucideIcon;
  comingSoon: string;
}

export interface ComingSoonPageProps {
  section: ComingSoonSection;
}

/** A section's name, what it will do and a way back to Home: never an error. */
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
      <p>This part of SP&A One isn&apos;t built yet.</p>
    </CentredMessage>
  );
}
