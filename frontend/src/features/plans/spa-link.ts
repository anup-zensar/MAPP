/** The SP&A view of a plan, or of one of its campaigns (013 FR-005). */
export function spaHref(planId: string, campaignId?: string): string {
  const path = `/plans/${encodeURIComponent(planId)}/spa`;
  return campaignId ? `${path}?${new URLSearchParams({ campaign: campaignId })}` : path;
}
