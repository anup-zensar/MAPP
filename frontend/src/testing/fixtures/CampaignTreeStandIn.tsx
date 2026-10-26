import { Outlet, useParams } from 'react-router';
import { CELL_KEYS } from '@/features/plans/campaigns/campaign-rows';
import { useCampaignTreeContent } from '@/features/plans/campaigns/useCampaignTreeContent';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { TreeTable } from '@/ui/TreeTable/TreeTable';

/** Stands in for feature 004's plan page: the tree with what useCampaignTreeContent gives it. */
export function CampaignTreeStandIn() {
  const { planId = '' } = useParams();
  const content = useCampaignTreeContent(planId);
  if (content.isLoading) {
    return <p>Loading rows</p>;
  }
  if (content.error) {
    return (
      <button type="button" onClick={content.refetch}>
        Try again
      </button>
    );
  }
  return (
    <>
      <div>{content.toolbarActions}</div>
      <ActionsMenu label="FY2026 Lubricants Europe" items={content.planRowActions} />
      <TreeTable
        label="Plan tree"
        columns={CELL_KEYS.map((key) => ({ key, header: key }))}
        nodes={content.nodes}
        expandedIds={new Set(content.nodes.map((node) => node.id))}
        onToggle={() => {}}
        footer={content.footer}
      />
      {content.nodes.length === 0 && <div data-testid="empty">{content.emptyAction}</div>}
      {content.dialogs}
      <Outlet />
    </>
  );
}
