import { useOutlet, useParams } from 'react-router';
import { useCampaignTreeContent } from '@/features/campaigns/useCampaignTreeContent';
import { BuildingPage } from '@/features/placeholders/BuildingPage';

/**
 * A plan's page (feature 004, US3), the layout route of /plans/:planId — STUB of the shared
 * foundation. Feature 004 replaces this file with the real page; the route stays.
 *
 * Already wired for feature 005, so neither feature changes the other's files:
 * - `useCampaignTreeContent(planId)` (features/campaigns) gives the campaign and initiative rows
 *   (the plan node's children in PlanTree), **Add campaign** for the toolbar, the plan row's ⋯
 *   menu and the empty tree, the totals row (TreeTable footer) and the row dialogs;
 * - the `<Outlet />` shows 005's campaign and initiative side panels (child routes).
 * The tab names the plan unless a panel on top names itself.
 */
export function PlanPage() {
  const { planId = '' } = useParams();
  const campaigns = useCampaignTreeContent(planId);
  const panel = useOutlet();

  return (
    <BuildingPage
      eyebrow="Plans"
      title="Plan"
      tabTitle={panel ? null : 'Plan'}
      description={planId}
      breadcrumb={[{ label: 'Plans', to: '/plans' }, { label: planId }]}
    >
      {campaigns.dialogs}
      {panel}
    </BuildingPage>
  );
}
