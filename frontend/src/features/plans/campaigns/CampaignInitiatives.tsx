import { Plus } from 'lucide-react';
import { Link } from 'react-router';
import { LinkButton } from '@/ui/Button/Button';
import { recordHref } from '@/ui/record-links';
import { allows } from '@/session/gate-action';
import { useFormatters } from '@/format/useFormatters';
import type { Campaign } from './api';
import { addInitiativeHref } from './campaign-rows';
import { PanelSection } from './PanelSection';
import styles from './RecordDetails.module.css';
import { ApprovalProgressChip } from './StatusChips';
import { ADD_INITIATIVE, initiativesHeading, NO_INITIATIVES } from './texts';

/** "Initiatives (n)" in a campaign's panel: each with its link, dates, approval progress and budget. */
export function CampaignInitiatives({ campaign, planId }: { campaign: Campaign; planId: string }) {
  const { dateRange, money } = useFormatters();
  return (
    <PanelSection title={initiativesHeading(campaign.initiatives.length)}>
      {campaign.initiatives.length === 0 ? (
        <div className={styles.empty}>
          <p className={styles.text}>{NO_INITIATIVES}</p>
          {allows(campaign.access?.addChild) && (
            <LinkButton to={addInitiativeHref(planId, campaign.id)} icon={<Plus />} size="sm">
              {ADD_INITIATIVE}
            </LinkButton>
          )}
        </div>
      ) : (
        <ul className={styles.children}>
          {campaign.initiatives.map((item) => (
            <li key={item.id} className={styles.child}>
              <div className={styles.childMain}>
                <Link to={recordHref('initiative', item.id, planId)}>{item.name}</Link>
                <span className={styles.muted}>{dateRange(item.startDate, item.endDate)}</span>
              </div>
              <div className={styles.childSide}>
                <ApprovalProgressChip progress={item.approvalProgress} />
                <span>{money(item.display.budget)}</span>
              </div>
            </li>
          ))}
        </ul>
      )}
    </PanelSection>
  );
}
