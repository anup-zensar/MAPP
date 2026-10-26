import { Pencil } from 'lucide-react';
import { useState } from 'react';
import type { ReactNode } from 'react';
import { useNavigate } from 'react-router';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { LinkButton } from '@/ui/Button/Button';
import { CompletenessChip } from '@/ui/CompletenessChip/CompletenessChip';
import { LevelTag } from '@/ui/LevelTag/LevelTag';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import type { ActionsMenuItem } from '@/ui/ActionsMenu/ActionsMenu';
import type { ActivityApproval, ShowSubmitProblems } from '@/features/approvals';
import { DecisionButton, DecisionDialog, SubmitButton } from '@/features/approvals';
import { allows, gateItem } from '@/session/gate-action';
import { recordHref } from '@/ui/record-links';
import { useFormatters } from '@/format/useFormatters';
import type { Activity } from './activitiesApi';
import styles from './ActivityHeader.module.css';
import { ActivityStatusChip } from '@/ui/ActivityStatusChip/ActivityStatusChip';
import { activityEditHref, datesText, lineItemsAttentionText } from './activityText';

export interface ActivityHeaderProps {
  activity: Activity;
  /** The activity's approval (feature 008), once loaded: who may decide, and how. */
  approval: ActivityApproval | undefined;
  /** The More menu's actions of feature 006 (Copy, Move, View history, Delete). */
  moreActions: ActionsMenuItem[];
  /** Opens the Ready to submit panel and moves the focus there. */
  onOpenReady: () => void;
  /** Submit found problems: show how many and list them. */
  onShowProblems: ShowSubmitProblems;
}

/** What the owner sees in More when the rules let them approve their own activity (FR·A-01). */
const OWN_APPROVAL_HINT =
  'You own this activity — approving it yourself skips an independent check.';

/**
 * The top of an activity's page (FR-019, design system §2.3, §6.3) through the one page header:
 * breadcrumb, name, and the meta row — level tag, status, completeness, ID, owner, dates. The
 * actions follow the person's main job here, with one primary button (UI audit X-04, AC-01, AC-03):
 *
 * - an approver of the current level, on a submitted activity they don't own: **Approve** (primary)
 *   and **Decline** side by side, Edit in More;
 * - everyone else: **Edit** (primary), and **Submit for approval** / **Re-submit for approval**
 *   while the activity is Created or Declined;
 * - the owner of a submitted activity is never offered Approve as a button; where the rules let
 *   them approve (D-020) it is in More with a warning (functional review A-01).
 */
export function ActivityHeader({
  activity,
  approval,
  moreActions,
  onOpenReady,
  onShowProblems,
}: ActivityHeaderProps) {
  const navigate = useNavigate();
  const format = useFormatters();
  const [ownApproval, setOwnApproval] = useState(false);
  const { plan, campaign, initiative } = activity;
  const editHref = activityEditHref(plan.id, activity.id);
  const approvalActivity = {
    id: activity.id,
    name: activity.name,
    status: activity.status,
    version: activity.version,
  };

  const canDecide =
    activity.status === 'submitted' &&
    approval !== undefined &&
    approval.actions.approve.available &&
    !approval.viewer.isOwner;
  const ownerMayApprove =
    activity.status === 'submitted' &&
    approval !== undefined &&
    approval.actions.approve.available &&
    approval.viewer.isOwner;

  // 014 FR-011: Edit and Submit only for a person who may do them; the server's reason in the menu.
  const access = activity.access;
  const mayEdit = allows(access?.edit);
  const edit = mayEdit ? (
    <LinkButton to={editHref} variant="primary" icon={<Pencil />}>
      Edit
    </LinkButton>
  ) : null;
  let actions: ReactNode = edit;
  let more = moreActions;
  if (canDecide && approval) {
    actions = (
      <>
        <DecisionButton mode="decline" approval={approval} />
        <DecisionButton mode="approve" approval={approval} variant="primary" />
      </>
    );
    more = [
      ...gateItem({ label: 'Edit', onSelect: () => void navigate(editHref) }, access?.edit),
      ...moreActions,
    ];
  } else if (activity.status === 'created' || activity.status === 'declined') {
    actions = (
      <>
        {allows(access?.submit) && (
          <SubmitButton
            activity={approvalActivity}
            onShowProblems={onShowProblems}
            variant="secondary"
          />
        )}
        {edit}
      </>
    );
  } else if (ownerMayApprove) {
    more = [
      { label: 'Approve', hint: OWN_APPROVAL_HINT, onSelect: () => setOwnApproval(true) },
      ...moreActions,
    ];
  }

  return (
    <>
      <PageHeader
        breadcrumb={[
          { label: 'Plans', to: '/plans' },
          { label: plan.name, to: recordHref('plan', plan.id) },
          { label: campaign.name, to: recordHref('campaign', campaign.id, plan.id) },
          { label: initiative.name, to: recordHref('initiative', initiative.id, plan.id) },
          { label: activity.name },
        ]}
        title={activity.name}
        meta={
          <>
            <LevelTag level="activity" />
            <ActivityStatusChip status={activity.displayStatus} />
            <CompletenessChip
              missing={activity.readiness.activity.map((problem) => ({
                id: problem.field,
                label: problem.message,
              }))}
              onSelectField={(field) =>
                void navigate(activityEditHref(plan.id, activity.id, field))
              }
              itemsBelow={activity.completeness.lineItemsNeedingAttention}
              itemsBelowText={lineItemsAttentionText}
              onItemsBelowSelect={onOpenReady}
            />
            <span className={styles.fact}>{activity.id}</span>
            {activity.owner ? (
              <PersonLabel person={activity.owner} />
            ) : (
              <span className={styles.fact}>No owner yet</span>
            )}
            <span className={styles.fact}>
              {datesText(activity.startDate, activity.endDate, activity.ongoing, format)}
            </span>
          </>
        }
        actions={
          <>
            {/* ⋯ first, then the secondary button, the primary last (design system §2.3). */}
            {more.length > 0 && <ActionsMenu label={activity.name} items={more} />}
            {actions}
          </>
        }
      />
      {ownApproval && approval && (
        <DecisionDialog mode="approve" approval={approval} onClose={() => setOwnApproval(false)} />
      )}
    </>
  );
}
