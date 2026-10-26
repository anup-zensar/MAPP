import { Pencil } from 'lucide-react';
import { useNavigate, useParams } from 'react-router';
import { ApiError } from '@/api/errors';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { LinkButton } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { LevelTag } from '@/ui/LevelTag/LevelTag';
import { LockedNote } from '@/ui/LockedNote/LockedNote';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { localDate } from '@/lib/format';
import { useFormatters } from '@/lib/useFormatters';
import { usePageTitle } from '@/lib/usePageTitle';
import {
  conditionText,
  isChoiceType,
  LEVEL_WORDS,
  MARKING_WORDS,
  TYPE_WORDS,
} from './attributeWords';
import { AttributePreview } from './AttributePreview';
import { OptionsCard } from './OptionsCard';
import { useShellAttribute } from './shellAttributesApi';
import type { ShellAttribute } from './shellAttributesApi';
import styles from './ShellAttributes.module.css';
import { useAttributeActions } from './useAttributeActions';
import { CanChange, SettingsViewOnlyNote } from '../shared/CanChange';

const NOT_FOUND = 'This Shell attribute no longer exists — someone may have deleted it.';

/**
 * One Shell attribute (feature 002, US2, US3, FR-020): its settings with where it is used, its
 * locks, its options, each with its own "Used by", and the preview of its level's form. Tab title
 * "<label> · Shell attributes".
 */
export function ShellAttributePage() {
  const { attributeId = '' } = useParams();
  const attribute = useShellAttribute(attributeId);
  usePageTitle(attribute.data ? `${attribute.data.label} · Shell attributes` : 'Shell attributes');

  if (attribute.data) {
    return <AttributeView attribute={attribute.data} />;
  }

  const notFound = attribute.error instanceof ApiError && attribute.error.kind === 'notFound';
  return (
    <>
      <PageHeader
        breadcrumb={[
          { label: 'Settings', to: '/settings' },
          { label: 'Shell attributes', to: '/settings/shell-attributes' },
          { label: notFound ? 'Not found' : 'Shell attribute' },
        ]}
        title="Shell attribute"
      />
      <Card>
        {notFound ? (
          <AttributeNotFound />
        ) : attribute.isError ? (
          <ErrorState error={attribute.error} onRetry={() => attribute.refetch()} />
        ) : (
          <Skeleton lines={6} height="18px" />
        )}
      </Card>
    </>
  );
}

/** The attribute is gone: says so, with the way back (plan "States on every page"). */
export function AttributeNotFound() {
  return (
    <div role="alert" className={styles.stack}>
      <p>{NOT_FOUND}</p>
      <div>
        <LinkButton to="/settings/shell-attributes">Back to Shell attributes</LinkButton>
      </div>
    </div>
  );
}

/** The attribute's page once it has arrived. */
function AttributeView({ attribute }: { attribute: ShellAttribute }) {
  const navigate = useNavigate();
  const listPath = `/settings/shell-attributes?level=${attribute.level}`;
  // After a delete, back to the list, where the message with Undo is shown.
  const actions = useAttributeActions(() => navigate(listPath));
  // One lock note per reason: Business unit's reason covers several fields.
  const lockReasons = [...new Set(attribute.locks.map((lock) => lock.reason))];

  return (
    <>
      <PageHeader
        breadcrumb={[
          { label: 'Settings', to: '/settings' },
          { label: 'Shell attributes', to: '/settings/shell-attributes' },
          { label: `${LEVEL_WORDS[attribute.level]} form`, to: listPath },
          { label: attribute.label },
        ]}
        title={attribute.label}
        actions={
          <CanChange>
            <LinkButton
              variant="primary"
              icon={<Pencil />}
              to={`/settings/shell-attributes/${attribute.id}/edit`}
            >
              Edit
            </LinkButton>
            <ActionsMenu
              label={attribute.label}
              items={[
                {
                  label: attribute.active ? 'Set inactive' : 'Reactivate',
                  onSelect: () => actions.setActive(attribute, !attribute.active),
                },
                {
                  label: 'Delete',
                  tone: 'danger',
                  separatorBefore: true,
                  onSelect: () => actions.askDelete(attribute),
                },
              ]}
            />
          </CanChange>
        }
      />
      <SettingsViewOnlyNote />
      <div className={styles.facts}>
        <LevelTag level={attribute.level} />
        <StatusChip tone={attribute.active ? 'positive' : 'neutral'}>
          {attribute.active ? 'Active' : 'Inactive'}
        </StatusChip>
        <span>{attribute.id}</span>
      </div>
      <div className={styles.stack}>
        {lockReasons.length > 0 && (
          <div className={styles.locks}>
            {lockReasons.map((reason) => (
              <LockedNote key={reason}>{reason}</LockedNote>
            ))}
          </div>
        )}
        <SettingsCard attribute={attribute} />
        {isChoiceType(attribute.type) ? (
          <OptionsCard attribute={attribute} />
        ) : (
          <Card title="Options">
            <p>{`${TYPE_WORDS[attribute.type]} attributes have no options.`}</p>
          </Card>
        )}
        <AttributePreview attribute={attribute} />
      </div>
      {actions.dialog}
    </>
  );
}

/** Level, type, marking, section, help text, dependency, condition, usage and who changed it. */
function SettingsCard({ attribute }: { attribute: ShellAttribute }) {
  const { date } = useFormatters();
  const day = (timestamp: string) => date(localDate(timestamp));

  return (
    <Card title="Settings">
      <dl className={styles.details}>
        <dt>Level</dt>
        <dd>{LEVEL_WORDS[attribute.level]}</dd>
        <dt>Type</dt>
        <dd>{TYPE_WORDS[attribute.type]}</dd>
        <dt>Marking</dt>
        <dd>{MARKING_WORDS[attribute.marking]}</dd>
        <dt>Section</dt>
        <dd>{attribute.section}</dd>
        <dt>Help text</dt>
        <dd>{attribute.helpText || '—'}</dd>
        <dt>Depends on</dt>
        <dd>{attribute.parentAttribute?.name ?? 'None'}</dd>
        <dt>Shown when the plan's business unit includes</dt>
        <dd>{conditionText(attribute) ?? 'Always shown'}</dd>
        <dt>Used by</dt>
        <dd>{attribute.usage.summary}</dd>
      </dl>
      <p className={styles.meta}>
        {`Created by ${attribute.createdBy.name} on ${day(attribute.createdAt)}, ` +
          `last changed by ${attribute.updatedBy.name} on ${day(attribute.updatedAt)}`}
      </p>
    </Card>
  );
}
