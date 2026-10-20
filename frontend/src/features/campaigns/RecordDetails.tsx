import { Plus } from 'lucide-react';
import type { ReactNode } from 'react';
import { Link, useNavigate } from 'react-router';
import { useAttributeForm } from '@/api/settingsQueries';
import {
  ActionsMenu,
  Breadcrumb,
  CompletenessChip,
  ErrorState,
  LevelTag,
  LinkButton,
  PersonLabel,
  ShellAttributeValues,
  SidePanel,
  Skeleton,
} from '@/components';
import type { ActionsMenuItem } from '@/components';
import { historyHref } from '@/features/activity-log/showHistory';
import { localDate } from '@/lib/format';
import { fieldHref, recordHref } from '@/lib/recordLinks';
import { useFormatters } from '@/lib/useFormatters';
import { BudgetCostSection, GoalsSection } from './BudgetCost';
import { usePlan } from './campaignApi';
import type { Campaign, Initiative } from './campaignApi';
import { addInitiativeHref } from './campaignRows';
import { InitiativeActivitiesSection, initiativeExtraActions } from './extensions';
import { Fact, Facts, PanelSection } from './PanelSection';
import styles from './RecordDetails.module.css';
import { ApprovalProgressChip, OutsideYearNote, TimingChip } from './StatusChips';
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
 * chips, facts, budget and cost, goals, description, Shell attributes and what is below it, with
 * **Edit** and **⋯ More**. Selecting a row opens this, never the form.
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
      ? [
          {
            label: ADD_INITIATIVE,
            onSelect: () => void navigate(addInitiativeHref(planId, record.id)),
          },
        ]
      : []),
    ...recordItems,
    {
      label: VIEW_HISTORY,
      onSelect: () => void navigate(historyHref(record.id)),
    },
    ...(isCampaign
      ? []
      : initiativeExtraActions({ id: record.id, name: record.name, planId }, navigate)),
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
            <LinkButton to={`${href}/edit`} variant="primary">
              {EDIT}
            </LinkButton>
            <ActionsMenu label={record.name} items={items} />
          </>
        }
      >
        <div className={styles.details}>
          <Breadcrumb
            items={[
              { label: 'Plans', to: '/plans' },
              { label: record.plan.name, to: planHref },
              ...(initiative
                ? [
                    {
                      label: initiative.campaign.name,
                      to: recordHref('campaign', initiative.campaign.id, planId),
                    },
                  ]
                : []),
              { label: record.name },
            ]}
          />
          <div className={styles.chips}>
            <LevelTag level={level} />
            <ApprovalProgressChip progress={record.approvalProgress} />
            <TimingChip timing={record.timing} />
            {/* #completeness: where Home's "Show fields" leads (feature 009). */}
            <span id="completeness">
              <CompletenessChip
                missing={record.completeness.missing.map((field) => ({
                  id: field.field,
                  label: field.label,
                }))}
                onSelectField={(field) => void navigate(fieldHref(level, record.id, field, planId))}
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
            {initiative && (
              <Fact label={CAMPAIGN}>
                <Link to={recordHref('campaign', initiative.campaign.id, planId)}>
                  {initiative.campaign.name}
                </Link>
              </Fact>
            )}
            <Fact label={OWNER}>{record.owner ? <PersonLabel person={record.owner} /> : '—'}</Fact>
            <Fact label={DATES}>{dateRange(record.startDate, record.endDate)}</Fact>
          </Facts>
          <p className={styles.created}>
            {createdLine(
              record.createdBy.name,
              date(localDate(record.createdAt)),
              record.updatedBy.name,
              date(localDate(record.updatedAt)),
            )}
          </p>
          {/* Amounts in the reader's display currency (D-220). */}
          <BudgetCostSection
            level={level}
            budget={record.display.budget}
            budgetAndCost={record.display.budgetAndCost}
            parentRemainingBudget={record.display.parentRemainingBudget}
          />
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
          <LinkButton to={addInitiativeHref(planId, campaign.id)} icon={<Plus />} size="sm">
            {ADD_INITIATIVE}
          </LinkButton>
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
