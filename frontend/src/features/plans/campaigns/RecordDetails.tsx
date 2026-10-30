import type { ReactNode } from 'react';
import { Link, useNavigate } from 'react-router';
import { RefusesOverspending } from '@/features/overspend';
import { FieldHelpTip, useAttributeForm } from '@/features/settings';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { LinkButton } from '@/ui/Button/Button';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { LevelTag } from '@/ui/LevelTag/LevelTag';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { ShellAttributeValues } from '@/ui/ShellAttributeFields/ShellAttributeValues';
import { SidePanel } from '@/ui/SidePanel/SidePanel';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import type { ActionsMenuItem } from '@/ui/ActionsMenu/ActionsMenu';
import { HistorySection, panelHistoryId, showHistory } from '@/features/activity-log';
import { allows, gateItem, moreItemFor } from '@/session/gate-action';
import { localDate } from '@/format/intl';
import { recordHref } from '@/ui/record-links';
import { useFormatters } from '@/format/useFormatters';
import { completenessFieldHref } from '../phasing/phasing-links';
import { RecordFinancials } from '../RecordFinancials';
import { BudgetByMonthCard } from '../phasing/BudgetByMonthCard';
import { AttentionChip } from './AttentionChip';
import { CampaignInitiatives } from './CampaignInitiatives';
import { BudgetCostSection, GoalsSection } from './BudgetCost';
import { usePlan } from './api';
import type { Campaign, Initiative } from './api';
import { addInitiativeHref } from './campaign-rows';
import { InitiativeActivitiesSection, initiativeExtraActions } from './extensions';
import { Fact, Facts, PanelSection } from './PanelSection';
import styles from './RecordDetails.module.css';
import { ApprovalProgressChip, OutsideYearNote } from './StatusChips';
import {
  activitiesLine,
  ADD_INITIATIVE,
  APPROVAL_PROGRESS_LABEL,
  CAMPAIGN,
  COMPLETENESS,
  createdLine,
  DATES,
  DESCRIPTION,
  EDIT,
  ID,
  NO_ACTIVITIES_YET,
  OWNER,
  PLAN,
  TIMING_WORD,
  VIEW_HISTORY,
} from './texts';
import type { RecordLevel } from './texts';

export interface RecordDetailsProps {
  level: RecordLevel;
  record: Campaign | Initiative;
  planId: string;
  /** Copy and Move to… (US6): after Add initiative, before View history. */
  recordItems?: ActionsMenuItem[];
  /** More items added by the caller (Delete, US5); they come after the panel's own. */
  moreItems?: ActionsMenuItem[];
  /** Dialogs the More items open. */
  dialogs?: ReactNode;
}

/**
 * A campaign's or initiative's details (US3, FR-016, design system §6.3): breadcrumb, level tag,
 * chips, facts, budget and cost, goals, description, Shell attributes, what is below it and its
 * History (feature 011), with **Edit** and **⋯ More**. Selecting a row opens this, never the form.
 */
export function RecordDetails({
  level,
  record,
  planId,
  recordItems = [],
  moreItems = [],
  dialogs,
}: RecordDetailsProps) {
  const navigate = useNavigate();
  const { date, dateRange } = useFormatters();
  const href = recordHref(level, record.id, planId);
  const planHref = `/plans/${planId}`;
  const close = () => void navigate(planHref);
  const isCampaign = 'initiatives' in record;
  const initiative = isCampaign ? null : record;

  const items: ActionsMenuItem[] = [
    ...moreItemFor(
      { label: EDIT, onSelect: () => void navigate(`${href}/edit`) },
      record.access?.edit,
    ),
    ...(isCampaign
      ? gateItem(
          {
            label: ADD_INITIATIVE,
            onSelect: () => void navigate(addInitiativeHref(planId, record.id)),
          },
          record.access?.addChild,
        )
      : []),
    ...recordItems,
    // Brings the panel's History part into view and focuses it (feature 011, FR-011).
    { label: VIEW_HISTORY, onSelect: () => showHistory(panelHistoryId(record.id)) },
    ...(isCampaign
      ? []
      : initiativeExtraActions(
          { id: record.id, name: record.name, planId, addActivity: record.access?.addChild },
          navigate,
        )),
    ...moreItems,
  ];

  return (
    <>
      <SidePanel
        open
        onOpenChange={(open) => !open && close()}
        title={record.name}
        footer={
          <>
            {allows(record.access?.edit) && (
              <LinkButton to={`${href}/edit`} variant="primary">
                {EDIT}
              </LinkButton>
            )}
            <ActionsMenu label={record.name} items={items} />
          </>
        }
      >
        <div className={styles.details}>
          {/* The level tag, the status and one completeness chip (design system §4.1). The page
              under the panel has the breadcrumb; the panel names its parents as values (C-02). */}
          <div className={styles.chips}>
            <LevelTag level={level} />
            <ApprovalProgressChip progress={record.approvalProgress} />
            <FieldHelpTip helpKey={`${level}.approvalProgress`} label={APPROVAL_PROGRESS_LABEL} />
            {/* #completeness: where Home's "Show fields" leads (feature 009). */}
            <span id="completeness">
              <AttentionChip
                missing={record.completeness.missing.map((field) => ({
                  id: field.field,
                  label: field.label,
                }))}
                onSelectField={(field) =>
                  void navigate(completenessFieldHref(level, record.id, field, planId))
                }
                itemsBelow={record.completeness.itemsBelowNeedingAttention}
                onItemsBelowSelect={() =>
                  void navigate(planHref, { state: { highlight: record.id } })
                }
              />
              <FieldHelpTip helpKey="common.completeness" label={COMPLETENESS} />
            </span>
          </div>
          {isCampaign && record.outsidePlanYear && <OutsideYearNote year={record.plan.year} />}
          <Facts>
            <Fact label={ID} helpKey="common.id">
              {record.id}
            </Fact>
            <Fact label={PLAN}>
              <Link to={planHref}>{record.plan.name}</Link>
            </Fact>
            {initiative && (
              <Fact label={CAMPAIGN} helpKey="initiative.campaign">
                <Link to={recordHref('campaign', initiative.campaign.id, planId)}>
                  {initiative.campaign.name}
                </Link>
              </Fact>
            )}
            <Fact label={OWNER} helpKey={`${level}.owner`}>
              {record.owner ? <PersonLabel person={record.owner} /> : '—'}
            </Fact>
            <Fact label={DATES} helpKey={`${level}.dates`}>
              {dateRange(record.startDate, record.endDate)}
              {record.timing && ` · ${TIMING_WORD[record.timing]}`}
            </Fact>
          </Facts>
          <p className={styles.created}>
            {createdLine(
              record.createdBy.name,
              date(localDate(record.createdAt)),
              record.updatedBy.name,
              date(localDate(record.updatedAt)),
            )}
          </p>
          <BudgetCostSection
            level={level}
            budget={record.budget}
            budgetAndCost={record.budgetAndCost}
            parentRemainingBudget={record.parentRemainingBudget}
            display={{
              budget: record.display.budget,
              plannedCost: record.display.budgetAndCost.plannedCost,
            }}
          />
          <RefusesOverspending mode={record.overspend.mode} />
          {/* D-305: 016's Budget by month, then 015's Financials, both open. */}
          <BudgetByMonthCard
            level={level}
            planId={planId}
            record={record}
            year={record.plan.year}
            currency={record.budget?.currency ?? 'USD'}
            access={record.access?.edit}
            variant="panel"
          />
          <RecordFinancials recordId={record.id} planId={planId} variant="panel" />
          <GoalsSection level={level} goals={record.display.goals} />
          <PanelSection
            title={
              <>
                {DESCRIPTION}
                <FieldHelpTip helpKey={`${level}.description`} label={DESCRIPTION} />
              </>
            }
          >
            <p className={styles.text}>{record.description || '—'}</p>
          </PanelSection>
          <AttributeValues level={level} record={record} planId={planId} />
          {isCampaign ? (
            <CampaignInitiatives campaign={record} planId={planId} />
          ) : (
            <>
              <p className={styles.text}>
                {record.counts.activities === 0
                  ? NO_ACTIVITIES_YET
                  : activitiesLine(
                      record.counts.activities,
                      record.counts.approvedActivities,
                      record.counts.lineItems,
                    )}
              </p>
              <InitiativeActivitiesSection planId={planId} initiativeId={record.id} />
            </>
          )}
          <HistorySection
            recordType={level}
            recordId={record.id}
            recordName={record.name}
            level={level}
            id={panelHistoryId(record.id)}
            variant="panel"
          />
        </div>
      </SidePanel>
      {dialogs}
    </>
  );
}

/** The record's Shell attributes, as Settings groups them ("—" when empty, "(inactive)"). */
function AttributeValues({
  level,
  record,
  planId,
}: Pick<RecordDetailsProps, 'level' | 'record' | 'planId'>) {
  const form = useAttributeForm(level);
  const plan = usePlan(planId);
  if (form.isError || plan.isError) {
    return (
      <ErrorState
        error={form.error ?? plan.error}
        onRetry={() => {
          void form.refetch();
          void plan.refetch();
        }}
      />
    );
  }
  if (!form.data || !plan.data) {
    return <Skeleton lines={3} />;
  }
  return (
    <ShellAttributeValues
      form={form.data}
      values={record.attributes}
      businessUnit={plan.data.businessUnit.map((option) => option.id)}
      planLink={`/plans/${planId}`}
      sectionHeading="h3"
    />
  );
}
