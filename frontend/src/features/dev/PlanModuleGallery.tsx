import { Breadcrumb } from '@/ui/Breadcrumb/Breadcrumb';
import { Card } from '@/ui/Card/Card';
import { CompletenessChip } from '@/ui/CompletenessChip/CompletenessChip';
import { LevelTag } from '@/ui/LevelTag/LevelTag';
import { LockedNote } from '@/ui/LockedNote/LockedNote';
import { NoAccess } from '@/ui/NoAccess/NoAccess';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { ProgressBar } from '@/ui/ProgressBar/ProgressBar';
import { SummaryCard } from '@/ui/SummaryCard/SummaryCard';
import { TeamPills } from '@/ui/TeamPill/TeamPill';
import { ViewOnlyNote } from '@/ui/ViewOnlyNote/ViewOnlyNote';
import { ALEX, PRIYA, TEAMS } from './gallery-data';
import styles from './ComponentGalleryPage.module.css';
import { AttributesExample } from './AttributesExample';
import { ChoiceExample } from './ChoiceExample';
import { DatesExample } from './DatesExample';
import { FormExamples } from './FormExamples';
import { ListToolbarExample } from './ListToolbarExample';
import { NumberFieldsExample } from './NumberFieldsExample';
import { PickersExample } from './PickersExample';
import { SplitExample } from './SplitExample';
import { TabsExample } from './TabsExample';
import { TextFieldsExample } from './TextFieldsExample';
import { TreeExample } from './TreeExample';

/** The levels of the plan tree, each with its tag. */
const LEVELS = ['plan', 'campaign', 'initiative', 'activity', 'lineItem'] as const;

/**
 * The shared building blocks of the Plan module (features 002–008, build guide §4) in the
 * component gallery, one card each, with live state so they can be tried.
 */
export function PlanModuleGallery() {
  return (
    <>
      <Card title="Breadcrumb">
        <Breadcrumb
          items={[
            { label: 'Plans', to: '/dev/components' },
            { label: 'FY2026 Lubricants Europe', to: '/dev/components' },
            { label: 'Helix launch' },
          ]}
        />
      </Card>
      <ListToolbarExample />
      <TabsExample />
      <Card title="LevelTag and ProgressBar">
        <div className={styles.stack}>
          <div className={styles.row}>
            {LEVELS.map((level) => (
              <LevelTag key={level} level={level} />
            ))}
          </div>
          <ProgressBar value={62} label="Budget used (62 %)" />
          <ProgressBar value={125} label="Budget used (125 %)" tone="problem" />
        </div>
      </Card>
      <Card title="Items below and summary detail">
        <div className={styles.stack}>
          <div className={styles.row}>
            <CompletenessChip missing={[]} itemsBelow={3} />
            <CompletenessChip
              missing={[]}
              itemsBelow={[
                {
                  id: 'C-1002',
                  label: "Campaign 'Retail growth'",
                  href: '/dev/components',
                },
              ]}
            />
          </div>
          <div className={styles.summaries}>
            <SummaryCard
              label="Remaining budget"
              value="-$50,000.00"
              tone="problem"
              detail="Over-allocated by $50,000.00"
            />
            <SummaryCard
              label="Budget used"
              value="62 %"
              detail={<ProgressBar value={62} label="Budget used" />}
            />
          </div>
        </div>
      </Card>
      <Card title="LockedNote">
        <LockedNote>The code can't change once the currency is saved.</LockedNote>
      </Card>
      <Card title="ViewOnlyNote (feature 014)">
        <div className={styles.stack}>
          <ViewOnlyNote area="plan" />
          <ViewOnlyNote area="settings" />
        </div>
      </Card>
      <Card title="NoAccess (feature 014)">
        <NoAccess message="You don't have access to this plan. Ask an administrator if you need it." />
      </Card>
      <NumberFieldsExample />
      <TextFieldsExample />
      <ChoiceExample />
      <SplitExample />
      <DatesExample />
      <Card title="PersonLabel and TeamPills">
        <div className={styles.stack}>
          <PersonLabel person={PRIYA} you detail="Marketer" size="md" />
          <PersonLabel person={{ ...ALEX, active: false }} />
          <TeamPills teams={TEAMS} linked />
        </div>
      </Card>
      <PickersExample />
      <AttributesExample />
      <TreeExample />
      <FormExamples />
    </>
  );
}
