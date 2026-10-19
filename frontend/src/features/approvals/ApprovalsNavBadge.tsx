export interface ApprovalsNavBadgeProps {
  /** The sidebar is the icon rail: the badge sits on the icon. */
  collapsed: boolean;
}

/**
 * The count of activities waiting for the signed-in person on the sidebar's Approvals item (008
 * plan, "Routes and navigation": a CountBadge, "<n> waiting for you") — STUB of the shared
 * foundation: nothing yet. Feature 008 replaces this file; navigation.ts already gives it to the
 * Approvals item and the Sidebar renders it inside the link.
 */
export function ApprovalsNavBadge({ collapsed }: ApprovalsNavBadgeProps) {
  void collapsed;
  return null;
}
