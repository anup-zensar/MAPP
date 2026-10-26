import { Plus } from 'lucide-react';
import type { ReactNode } from 'react';
import { Link, useNavigate } from 'react-router';
import { useAttributeForm } from '@/features/settings';
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
import { allows, gateItem } from '@/session/gate-action';
import { localDate } from '@/format/intl';
import { recordHref } from '@/ui/record-links';
import { useFormatters } from '@/format/useFormatters';
import { completenessFieldHref } from '../phasing/phasing-links';
import { RecordFinancials } from '../RecordFinancials';
import { BudgetByMonthCard } from '../phasing/BudgetByMonthCard';
import { AttentionChip } from './AttentionChip';
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
  CAMPAIGN,
  createdLine,
  DATES,
  DESCRIPTION,
  EDIT,
  ID,
  initiativesHeading,
  NO_ACTIVITIES_YET,
  NO_INITIATIVES,
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
  const initiative = isCampaign ? null : (record as Initiative);

  const items: ActionsMenuItem[] = [
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
            </span>
          </div>
          {isCampaign && (record as Campaign).outsidePlanYear && (
            <OutsideYearNote year={record.plan.year} />
          )}
          <Facts>
            <Fact label={ID}>{record.id}</Fact>
            <Fact label={PLAN}>
              <Link to={planHref}>{record.plan.name}</Link>
            </Fact>
            {initiative && (
              <Fact label={CAMPAIGN}>
                <Link to={recordHref('campaign', initiative.campaign.id, planId)}>
                  {initiative.campaign.name}
                </Link>
              </Fact>
            )}
            <Fact label={OWNER}>{record.owner ? <PersonLabel person={record.owner} /> : '—'}</Fact>
            <Fact label={DATES}>
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
          {/* Amounts in the plan currency, the reader's display currency beneath when it
              differs (016 FR-013; before 016 both are the same). */}
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
          {/* D-305: 016's Budget by month, then 015's Financials, both open. */}
          <BudgetByMonthCard
            level={level}
            planId={planId}
            record={record}
            year={record.plan.year}
            currency={record.budget?.currency ?? 'USD'}
            variant="panel"
          />
          <RecordFinancials recordId={record.id} planId={planId} variant="panel" />
          <GoalsSection goals={record.display.goals} />
          <PanelSection title={DESCRIPTION}>
            <p className={styles.text}>{record.description || '—'}</p>
          </PanelSection>
          <AttributeValues level={level} record={record} planId={planId} />
          {isCampaign ? (
            <CampaignInitiatives campaign={record as Campaign} planId={planId} />
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

/** "Initiatives (n)": each with its link, dates, approval progress and budget (FR-016). */
function CampaignInitiatives({ campaign, planId }: { campaign: Campaign; planId: string }) {
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
