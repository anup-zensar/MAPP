import { BuildingPage } from '@/features/placeholders/BuildingPage';

/**
 * Approvals (feature 008, US3), at /approvals (?tab=waiting|open|decided, ?activity=<id> opens the
 * preview) — STUB of the shared foundation. Feature 008 replaces this file; the route stays.
 */
export function ApprovalsPage() {
  return (
    <BuildingPage
      eyebrow="Approvals"
      title="Approvals"
      description="Approve or decline the activities waiting for you, and follow the ones you sent."
    />
  );
}
