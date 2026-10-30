import { ClipboardList, Copy, Plus } from 'lucide-react';
import { useState } from 'react';
import { ApiError } from '@/api/errors';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { Avatar } from '@/ui/Avatar/Avatar';
import { Button, LinkButton } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { CompletenessChip } from '@/ui/CompletenessChip/CompletenessChip';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { DemoBanner } from '@/ui/DemoBanner/DemoBanner';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { InfoTip } from '@/ui/InfoTip/InfoTip';
import { Logo } from '@/ui/Logo/Logo';
import { LogoTile } from '@/ui/Logo/LogoTile';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { SidePanel } from '@/ui/SidePanel/SidePanel';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { useToast } from '@/ui/Toast/useToast';
import { Tooltip } from '@/ui/Tooltip/Tooltip';
import type { MissingField } from '@/ui/CompletenessChip/CompletenessChip';
import type { StatusTone } from '@/ui/StatusChip/StatusChip';
import { usePageTitle } from '@/ui/usePageTitle';
import styles from './ComponentGalleryPage.module.css';
import { ErrorSummaryExample } from './ErrorSummaryExample';
import { FieldExamples } from './FieldExamples';
import { PanelField } from './PanelField';
import { SummaryExamples } from './SummaryExamples';
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
  const [panelOpen, setPanelOpen] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);

  function showUndoMessage() {
    show({
      message: "Campaign 'Helix launch' deleted",
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
            <Button variant="tertiary">Ghost</Button>
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
              <Button variant="tertiary" icon={<Copy />} aria-label="Copy plan" />
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

        <SummaryExamples />

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
            <LogoTile />
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

/** Moves the keyboard focus to a field of the TextField card, as a page does for its fields. */
function focusField(id: string) {
  document.getElementById(id)?.focus();
}
