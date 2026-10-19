import { ClipboardList, Copy, ListChecks, Plus } from 'lucide-react';
import { useState } from 'react';
import { ApiError } from '@/api/errors';
import {
  ActionsMenu,
  Avatar,
  Button,
  Card,
  CompletenessChip,
  ConfirmDialog,
  DemoBanner,
  EmptyState,
  ErrorState,
  FormErrorSummary,
  InfoTip,
  LinkButton,
  LinkSummaryCard,
  Logo,
  PageHeader,
  PasswordField,
  SidePanel,
  Skeleton,
  StatusChip,
  SummaryCard,
  TextField,
  Tooltip,
  useToast,
} from '@/components';
import type { FieldProblem, MissingField, StatusTone } from '@/components';
import { useFormatters } from '@/lib/useFormatters';
import { usePageTitle } from '@/lib/usePageTitle';
import styles from './ComponentGalleryPage.module.css';
import { PlanModuleGallery } from './PlanModuleGallery';

/** The six status families with a status of each, as a word (design system §4). */
const STATUSES: { tone: StatusTone; word: string }[] = [
  { tone: 'neutral', word: 'Draft' },
  { tone: 'review', word: 'Submitted' },
  { tone: 'positive', word: 'Approved' },
  { tone: 'active', word: 'In progress' },
  { tone: 'done', word: 'Complete' },
  { tone: 'problem', word: 'Declined' },
];

/** The fields of the TextField card that the completeness chip says are missing. */
const MISSING_FIELDS: MissingField[] = [
  { id: 'campaignName', label: 'Campaign name' },
  { id: 'budgetOwner', label: 'Budget owner' },
];

/**
 * The problems the error summary lists. One list for the whole page: the summary takes the focus
 * whenever it is given a new list, so a list made anew on every render would keep taking it.
 */
const SUMMARY_ERRORS: FieldProblem[] = [
  { field: 'campaignName', message: 'Enter a name.' },
  { field: 'budgetOwner', message: 'Choose a budget owner.' },
];

const TOOLTIP_SIDES = [
  { side: 'top', label: 'Top' },
  { side: 'right', label: 'Right' },
  { side: 'bottom', label: 'Bottom' },
  { side: 'left', label: 'Left' },
] as const;

/** Does nothing: the example buttons are there to be looked at. */
function doNothing() {}

/**
 * Every shared building block of design system §5 in all its variants (FR-019, research R23), so
 * designers can review them in light and dark mode and the end-to-end tests can check them for
 * accessibility. Only in local and development environments (DevOnly in router.tsx).
 */
export function ComponentGalleryPage() {
  usePageTitle('Shared building blocks');
  const { show } = useToast();
  // Amounts as the signed-in person shows them (feature 003, FR-023).
  const { money, moneyCompact } = useFormatters();
  const [panelOpen, setPanelOpen] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);

  function showUndoMessage() {
    show({
      message: "Campaign 'Helix launch' moved to the recycle bin",
      action: {
        label: 'Undo',
        altText: 'Restore it from the recycle bin',
        onClick: () => show({ message: "Campaign 'Helix launch' restored" }),
      },
    });
  }

  return (
    <>
      <PageHeader
        eyebrow="Developer"
        title="Shared building blocks"
        description="Every shared component of design system §5 in all its variants."
      />
      <div className={styles.cards}>
        <Card title="Button and LinkButton">
          <div className={styles.row}>
            <Button variant="primary">Primary</Button>
            <Button>Secondary</Button>
            <Button variant="danger">Danger</Button>
            <Button variant="ghost">Ghost</Button>
          </div>
          <div className={styles.row}>
            <Button variant="primary" size="sm">
              Small primary
            </Button>
            <Button size="sm">Small</Button>
            <Button variant="primary" icon={<Plus />}>
              With an icon
            </Button>
            {/* An icon alone needs a name and a tooltip (design system §9). */}
            <Tooltip content="Copy plan">
              <Button variant="ghost" icon={<Copy />} aria-label="Copy plan" />
            </Tooltip>
            <LinkButton to="/">Go to Home</LinkButton>
          </div>
        </Card>

        <Card title="PageHeader">
          <PageHeader
            eyebrow="Plans"
            title="Plan with"
            highlight="confidence."
            description="Every plan, campaign and activity in one place."
            actions={
              <Button variant="primary" icon={<Plus />}>
                New plan
              </Button>
            }
          />
        </Card>

        <Card title="SummaryCard">
          <div className={styles.summaries}>
            <SummaryCard label="Plans" value={6} tip="The plans of this year that you can see." />
            <SummaryCard
              label="Planned cost"
              value={moneyCompact({ amount: '1250000', currency: 'USD' })}
              fullValue={money({ amount: '1250000', currency: 'USD' })}
              tip="The planned cost of every activity in these plans."
            />
            <SummaryCard
              label="Waiting for approval"
              value={3}
              tone="review"
              tip="Activities sent for approval that are not approved or declined yet."
            />
            <SummaryCard
              label="Over budget"
              value={money({ amount: '-12500', currency: 'USD' })}
              tone="problem"
              tip="How much the planned cost is above the budget."
            />
          </div>
        </Card>

        <Card title="LinkSummaryCard" subtitle="The link variant: the whole card opens its list">
          <div className={styles.summaries}>
            <LinkSummaryCard
              icon={<ClipboardList />}
              tint="var(--blue)"
              label="My teams' plans"
              tip="Plans that have at least one of your teams."
              value="3"
              detail={<span>2 published · 1 draft</span>}
              to="/plans?scope=myTeams"
              linkLabel="open the Plans list"
            />
            <LinkSummaryCard
              icon={<ListChecks />}
              tint="var(--level-campaign)"
              label="Planned cost"
              tip="The planned cost of every activity in these plans."
              value={moneyCompact({ amount: '1250000', currency: 'USD' })}
              fullValue={money({ amount: '1250000', currency: 'USD' })}
              to="#main"
              linkLabel="go to the top of this page"
            />
          </div>
        </Card>

        <Card
          title="Card"
          subtitle="With a subtitle and actions on the right"
          actions={<Button size="sm">Edit</Button>}
        >
          <p>Content sits on white, with rounded corners, a faint border and a soft shadow.</p>
        </Card>

        <Card title="StatusChip">
          <div className={styles.row}>
            {STATUSES.map(({ tone, word }) => (
              <StatusChip key={tone} tone={tone}>
                {word}
              </StatusChip>
            ))}
          </div>
        </Card>

        <Card title="CompletenessChip">
          <div className={styles.row}>
            <CompletenessChip missing={[]} />
            <CompletenessChip missing={MISSING_FIELDS} onSelectField={focusField} />
          </div>
        </Card>

        <Card title="InfoTip">
          <p className={styles.label}>
            Unassigned planned cost
            <InfoTip
              label="Unassigned planned cost"
              text="The part of an activity's planned cost that no line item holds yet."
            />
          </p>
        </Card>

        <Card title="Tooltip">
          <div className={styles.row}>
            {TOOLTIP_SIDES.map(({ side, label }) => (
              <Tooltip key={side} side={side} content={`A tooltip on the ${side}`}>
                <Button>{label}</Button>
              </Tooltip>
            ))}
          </div>
        </Card>

        <Card title="ActionsMenu">
          <ActionsMenu
            label="Helix launch"
            items={[
              { label: 'Edit', onSelect: () => setPanelOpen(true) },
              {
                label: 'Delete',
                tone: 'danger',
                separatorBefore: true,
                onSelect: () => setConfirmOpen(true),
              },
            ]}
          />
        </Card>

        <Card title="Avatar">
          <div className={styles.row}>
            <span className={styles.person}>
              <Avatar initials="PR" size="sm" />
              Priya Raman
            </span>
            <span className={styles.person}>
              <Avatar initials="DO" size="md" />
              Daniel Okoro
            </span>
            <span className={styles.person}>
              <Avatar initials="ML" size="lg" />
              Mei Lin
            </span>
          </div>
        </Card>

        <Card title="Logo">
          <div className={styles.row}>
            <Logo />
            <Logo size="lg" />
            <Logo withWordmark />
            <Logo size="lg" withWordmark />
          </div>
        </Card>

        <Card title="SidePanel">
          <Button onClick={() => setPanelOpen(true)}>Open side panel</Button>
        </Card>

        <Card title="ConfirmDialog">
          <Button onClick={() => setConfirmOpen(true)}>Open confirm dialog</Button>
        </Card>

        <Card title="Toast">
          <div className={styles.row}>
            <Button onClick={() => show({ message: "Campaign 'Helix launch' created" })}>
              Show message
            </Button>
            <Button onClick={showUndoMessage}>Show message with Undo</Button>
          </div>
        </Card>

        <Card title="EmptyState">
          <EmptyState
            icon={<ClipboardList />}
            title="No plans yet — create your first plan"
            text="A plan holds the campaigns, initiatives and activities of one year."
            action={
              <Button variant="primary" icon={<Plus />}>
                New plan
              </Button>
            }
          />
        </Card>

        <Card title="Skeleton">
          <div className={styles.stack}>
            <Skeleton lines={3} />
            <Skeleton width="240px" height="40px" />
          </div>
        </Card>

        <Card title="TextField and PasswordField">
          <FieldExamples />
        </Card>

        <Card title="FormErrorSummary">
          <ErrorSummaryExample />
        </Card>

        <Card title="ErrorState">
          <div className={styles.columns}>
            <ErrorState error={new ApiError('unreachable')} onRetry={doNothing} />
            <ErrorState error={new ApiError('other')} onRetry={doNothing} />
          </div>
        </Card>

        <Card title="DemoBanner">
          <DemoBanner />
        </Card>

        <PlanModuleGallery />
      </div>

      <SidePanel
        open={panelOpen}
        onOpenChange={setPanelOpen}
        title="Edit campaign"
        description="Short records are created and edited in a panel like this one."
        footer={
          <>
            <Button variant="primary" onClick={() => setPanelOpen(false)}>
              Save
            </Button>
            <Button onClick={() => setPanelOpen(false)}>Cancel</Button>
          </>
        }
      >
        <PanelField />
      </SidePanel>

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title="Delete campaign 'Helix launch'?"
        confirmLabel="Delete campaign"
        tone="danger"
        onConfirm={() => setConfirmOpen(false)}
      >
        It goes to the recycle bin together with its 3 initiatives. You can restore it.
      </ConfirmDialog>
    </>
  );
}

/** The text fields: with a marking and an error, with a hint, and a password. */
function FieldExamples() {
  const [campaignName, setCampaignName] = useState('');
  const [budgetOwner, setBudgetOwner] = useState('Priya Raman');
  const [password, setPassword] = useState('');

  return (
    <div className={styles.fields}>
      <TextField
        label="Campaign name"
        name="campaignName"
        marking="Required"
        value={campaignName}
        onChange={setCampaignName}
        error="Enter a name."
      />
      <TextField
        label="Budget owner"
        name="budgetOwner"
        marking="Needed to publish"
        hint="The person who answers for the budget."
        value={budgetOwner}
        onChange={setBudgetOwner}
      />
      <PasswordField label="Password" name="password" value={password} onChange={setPassword} />
    </div>
  );
}

/**
 * The error summary appears when asked for: it takes the focus when it appears, which on page
 * load would move the focus, and the scroll position, away from the top of the page.
 */
function ErrorSummaryExample() {
  const [shown, setShown] = useState(false);

  return (
    <div className={styles.stack}>
      <div>
        <Button onClick={() => setShown(true)}>Show the error summary</Button>
      </div>
      {shown && <FormErrorSummary errors={SUMMARY_ERRORS} />}
    </div>
  );
}

/** The one field of the example side panel. */
function PanelField() {
  const [name, setName] = useState('Helix launch');
  return (
    <TextField label="Name" name="panelName" marking="Required" value={name} onChange={setName} />
  );
}

/** Moves the keyboard focus to a field of the TextField card, as a page does for its fields. */
function focusField(id: string) {
  document.getElementById(id)?.focus();
}
