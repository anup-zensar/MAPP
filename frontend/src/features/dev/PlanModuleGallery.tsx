import { useState } from 'react';
import {
  Breadcrumb,
  Button,
  Card,
  CheckboxField,
  ChoiceField,
  ClearValuesDialog,
  CompletenessChip,
  CountBadge,
  DateRangeField,
  DecimalField,
  FilterChips,
  FormDialog,
  FormFooter,
  FormPanel,
  FormSection,
  LevelTag,
  LockedNote,
  MoneyField,
  NoAccess,
  PersonLabel,
  PersonPicker,
  Tabs,
  ProgressBar,
  SearchField,
  SegmentedFilter,
  ShellAttributeFields,
  ShellAttributeValues,
  SplitEditor,
  SummaryCard,
  TeamPicker,
  TeamPills,
  TextAreaField,
  TextField,
  TreeTable,
  valuesToClear,
  ViewOnlyNote,
} from '@/components';
import type { AttributeValues, SplitShare } from '@/components';
import { CAMPAIGN_FORM, COST_CENTRES, PEOPLE, PLAN_TREE, TEAMS } from './galleryData';
import styles from './ComponentGalleryPage.module.css';

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
          <PersonLabel person={PEOPLE[1]} you detail="Marketer" size="md" />
          <PersonLabel person={{ ...PEOPLE[0], active: false }} />
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

function ListToolbarExample() {
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState<'active' | 'inactive' | 'all'>('active');
  const [chips, setChips] = useState([
    { key: 'status', label: 'Status: Draft' },
    { key: 'year', label: 'Year: 2026' },
  ]);
  return (
    <Card title="SearchField, SegmentedFilter and FilterChips">
      <div className={styles.stack}>
        <SearchField
          label="Search models"
          placeholder="Search name"
          value={search}
          onChange={setSearch}
        />
        <SegmentedFilter
          label="Status"
          value={status}
          onChange={setStatus}
          options={[
            { value: 'active', label: 'Active', count: 5 },
            { value: 'inactive', label: 'Inactive', count: 1 },
            { value: 'all', label: 'All', count: 6 },
          ]}
        />
        <FilterChips
          chips={chips}
          onRemove={(key) => setChips(chips.filter((chip) => chip.key !== key))}
          onClearAll={() => setChips([])}
        />
      </div>
    </Card>
  );
}

function TabsExample() {
  const [tab, setTab] = useState<'waiting' | 'open' | 'decided'>('waiting');
  return (
    <Card title="Tabs and CountBadge">
      <Tabs
        label="Approvals"
        value={tab}
        onChange={setTab}
        tabs={[
          {
            value: 'waiting',
            label: 'Waiting for me',
            badge: <CountBadge count={3} label="waiting" />,
          },
          { value: 'open', label: 'All open', badge: <CountBadge count={120} label="open" /> },
          { value: 'decided', label: 'Decided by me' },
        ]}
      >
        <p>The {tab} tab.</p>
      </Tabs>
    </Card>
  );
}

function NumberFieldsExample() {
  const [budget, setBudget] = useState('12.5k');
  const [rate, setRate] = useState('2');
  const [inq, setInq] = useState('250000');
  return (
    <Card title="MoneyField and DecimalField">
      <div className={styles.fields}>
        <MoneyField
          label="Plan budget"
          name="galleryBudget"
          currency="USD"
          marking="Needed to publish"
          value={budget}
          onChange={setBudget}
        />
        <DecimalField
          label="INQ→MQL rate"
          name="galleryRate"
          decimals={2}
          suffix="%"
          value={rate}
          onChange={setRate}
        />
        <DecimalField
          label="Goal amount (INQ)"
          name="galleryInq"
          decimals={0}
          value={inq}
          onChange={setInq}
        />
      </div>
    </Card>
  );
}

function TextFieldsExample() {
  const [description, setDescription] = useState('');
  const [aop, setAop] = useState(false);
  const [budgetName, setBudgetName] = useState('');
  return (
    <Card title="TextAreaField, CheckboxField and suggestions">
      <div className={styles.fields}>
        <TextAreaField
          label="Description"
          name="galleryDescription"
          maxLength={60}
          hint="Counts down in its last 50 characters."
          value={description}
          onChange={setDescription}
        />
        <CheckboxField
          label="Strategic / AOP plan"
          name="galleryAop"
          checked={aop}
          onChange={setAop}
        />
        <CheckboxField
          label="Ongoing (no end date)"
          name="galleryOngoing"
          checked
          onChange={() => {}}
          disabledReason="Ongoing can't be undone — to plan with an end date, create a new activity."
        />
        <TextField
          label="Budget name"
          name="galleryBudgetName"
          hint="Items with the same year and budget name form one finance budget."
          suggestions={['Europe – Retail', 'Asia – B2B']}
          value={budgetName}
          onChange={setBudgetName}
        />
      </div>
    </Card>
  );
}

function ChoiceExample() {
  const [type, setType] = useState('');
  const [channels, setChannels] = useState<string[]>([]);
  const [centre, setCentre] = useState('');
  return (
    <Card title="ChoiceField">
      <div className={styles.fields}>
        <ChoiceField
          label="Activity type"
          name="galleryType"
          marking="Required"
          options={[
            { id: 'AT-1', label: 'Event', active: true },
            { id: 'AT-2', label: 'Paid media', active: true },
            { id: 'AT-3', label: 'Print', active: false },
          ]}
          value={type}
          onChange={setType}
        />
        <ChoiceField
          label="Channels"
          name="galleryChannels"
          multiple
          options={[
            { id: 'SO-TV', label: 'TV', active: true },
            { id: 'SO-RADIO', label: 'Radio', active: true },
            { id: 'SO-WEB', label: 'Web', active: true },
          ]}
          value={channels}
          onChange={setChannels}
        />
        <ChoiceField
          label="Cost centre"
          name="galleryCentre"
          hint="1,217 options: type to search."
          options={COST_CENTRES}
          value={centre}
          onChange={setCentre}
        />
        <ChoiceField
          label="Region"
          name="galleryRegion"
          options={[]}
          value=""
          onChange={() => {}}
          disabledReason="Choose a cluster first."
        />
      </div>
    </Card>
  );
}

function SplitExample() {
  const [split, setSplit] = useState<SplitShare[]>([
    { optionId: 'SO-HELIX', share: '60.00' },
    { optionId: 'SO-RIMULA', share: '30.00' },
  ]);
  return (
    <Card title="SplitEditor">
      <SplitEditor
        label="Brand split"
        name="gallerySplit"
        marking="Needed to submit"
        options={[
          { id: 'SO-HELIX', label: 'Shell Helix', active: true },
          { id: 'SO-RIMULA', label: 'Shell Rimula', active: true },
          { id: 'SO-TELLUS', label: 'Shell Tellus', active: true },
        ]}
        value={split}
        onChange={setSplit}
      />
    </Card>
  );
}

function DatesExample() {
  const [range, setRange] = useState({ start: '2026-03-01', end: '' });
  return (
    <Card title="DateRangeField">
      <DateRangeField
        label="Dates"
        idPrefix="gallery"
        marking="Needed to publish"
        start={range.start}
        end={range.end}
        onChange={setRange}
        allowed={{ start: '2026-02-01', end: '2026-11-30' }}
        allowedHint="Must be inside the campaign's dates (01 Feb – 30 Nov 2026)"
      />
    </Card>
  );
}

function PickersExample() {
  const [owner, setOwner] = useState('U-1002');
  const [teams, setTeams] = useState(['T-1001', 'T-1003']);
  return (
    <Card title="PersonPicker and TeamPicker">
      <div className={styles.fields}>
        <TeamPicker
          label="Teams"
          name="galleryTeams"
          teams={TEAMS}
          value={teams}
          onChange={setTeams}
        />
        <PersonPicker
          label="Owner"
          name="galleryOwner"
          marking="Needed to publish"
          people={PEOPLE}
          currentUserId="U-1002"
          teamIds={teams}
          value={owner}
          onChange={setOwner}
        />
      </div>
    </Card>
  );
}

function AttributesExample() {
  const stored: AttributeValues = { 'SA-CLUSTER': ['SO-B2B'], 'SA-REGION': ['SO-EU'] };
  const [values, setValues] = useState<AttributeValues>(stored);
  const [asking, setAsking] = useState(false);
  const cleared = valuesToClear(CAMPAIGN_FORM, values, stored);
  return (
    <Card
      title="ShellAttributeFields and ShellAttributeValues"
      actions={
        <Button size="sm" onClick={() => setAsking(true)}>
          Try saving
        </Button>
      }
    >
      <div className={styles.columns}>
        <ShellAttributeFields
          form={CAMPAIGN_FORM}
          businessUnit={['SO-LUB']}
          values={values}
          previousValues={stored}
          onChange={setValues}
        />
        <ShellAttributeValues form={CAMPAIGN_FORM} businessUnit={['SO-LUB']} values={values} />
      </div>
      <ClearValuesDialog
        open={asking && cleared.length > 0}
        onOpenChange={setAsking}
        cleared={cleared}
        onConfirm={() => setAsking(false)}
      />
    </Card>
  );
}

function TreeExample() {
  const [expanded, setExpanded] = useState(new Set(['P-1001', 'C-1001']));
  function toggle(id: string) {
    const next = new Set(expanded);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    setExpanded(next);
  }
  return (
    <Card title="TreeTable">
      <TreeTable
        label="Plan tree"
        columns={[
          { key: 'name', header: 'Name' },
          { key: 'owner', header: 'Owner' },
          { key: 'budget', header: 'Budget', align: 'end' },
        ]}
        nodes={PLAN_TREE}
        expandedIds={expanded}
        onToggle={toggle}
        footer={{ name: 'Total · 1 campaign', budget: '$500,000.00' }}
      />
    </Card>
  );
}

function FormExamples() {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [panelOpen, setPanelOpen] = useState(false);
  const [copyName, setCopyName] = useState('FY2026 Lubricants Europe (copy)');
  return (
    <Card title="FormSection, FormFooter, FormDialog and FormPanel">
      <div className={styles.fields}>
        <FormSection title="Basics" description="Sections of a full-page form, in order.">
          <p>Fields go here.</p>
        </FormSection>
        <FormFooter>
          <Button variant="primary" onClick={() => setDialogOpen(true)}>
            Open form dialog
          </Button>
          <Button onClick={() => setPanelOpen(true)}>Open form panel</Button>
        </FormFooter>
      </div>
      <FormDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        title="Copy plan 'FY2026 Lubricants Europe'"
        submitLabel="Copy plan"
        onSubmit={() => setDialogOpen(false)}
      >
        <TextField
          label="Name of the copy"
          name="galleryCopyName"
          value={copyName}
          onChange={setCopyName}
        />
      </FormDialog>
      {panelOpen && (
        <FormPanel
          title="New team"
          isDirty={false}
          saving={false}
          problems={[]}
          failure={null}
          onSave={() => setPanelOpen(false)}
          onClose={() => setPanelOpen(false)}
        >
          <p>The fields of a short record.</p>
        </FormPanel>
      )}
    </Card>
  );
}
