import { House } from 'lucide-react';
import type { ComingSoonSection } from '@/app/navigation';
import { Card, EmptyState, LinkButton, PageHeader } from '@/components';
import { usePageTitle } from '@/lib/usePageTitle';

export interface ComingSoonPageProps {
  /** The section a later feature builds (navigation.ts). */
  section: ComingSoonSection;
}

/**
 * The page of a section that a later feature builds (FR-015): its name, one sentence on what it
 * will do, and a way back to Home — never an error.
 */
export function ComingSoonPage({ section }: ComingSoonPageProps) {
  usePageTitle(section.label);

  return (
    <>
      <PageHeader eyebrow={section.label} title={section.label} description={section.comingSoon} />
      <Card>
        <EmptyState
          icon={<section.icon />}
          title="Coming soon"
          text="This part of NorthStar isn't built yet."
          action={
            <LinkButton to="/" variant="primary" icon={<House />}>
              Go to Home
            </LinkButton>
          }
        />
      </Card>
    </>
  );
}
