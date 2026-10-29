import { Link } from 'react-router';
import { Card } from '@/ui/Card/Card';
import { useAccess } from '@/session/access';
import type { Permission } from '@/session/access';
import type { HomeSummary } from './api';
import styles from './GettingStartedCard.module.css';

/** One step of the Plan module, in the order people do them (001 plan "Home"). */
interface Step {
  title: string;
  to: string;
  text: string;
  /** The permission the step needs (038 FR-051). */
  needs: Permission;
}

const STEPS: Step[] = [
  {
    title: 'Set up Settings',
    to: '/settings',
    needs: 'settings.manage',
    text: 'Add the models, activity types, Shell attributes and currencies your plans use.',
  },
  {
    title: 'Create a plan',
    to: '/plans',
    needs: 'plans.create',
    text: 'Give the plan its year, budget, owner and teams.',
  },
  {
    title: 'Add campaigns and initiatives',
    to: '/plans',
    needs: 'campaigns.edit',
    text: 'Split the plan into campaigns, and each campaign into initiatives with their own budgets.',
  },
  {
    title: 'Add activities and line items',
    to: '/plans',
    needs: 'activities.edit',
    text: 'Describe each piece of work and plan its costs as line items.',
  },
  {
    title: 'Submit for approval',
    to: '/approvals',
    needs: 'activities.submit',
    text: 'Send activities to their approvers and follow their status.',
  },
];

export interface GettingStartedCardProps {
  /** Why the card is shown: the person is in no team, or none of their teams has a plan. */
  reason: NonNullable<HomeSummary['gettingStarted']>;
}

const COUNT_WORDS = ['', 'one step', 'two steps', 'three steps', 'four steps', 'five steps'];

/**
 * Feature 001's five steps of the Plan module (001 FR-018), shown on Home only to new people: in
 * no team, or in teams without a plan (009 FR-020, D-170) — only the steps the person's roles allow
 * (038 FR-051). A person in no team also reads how to join one, or whom to ask.
 */
export function GettingStartedCard({ reason }: GettingStartedCardProps) {
  const access = useAccess();
  const steps = STEPS.filter((step) => access.has(step.needs));
  const admins = access.adminNames();

  return (
    <Card
      title="Getting started"
      subtitle={steps.length > 0 ? `The Plan module in ${COUNT_WORDS[steps.length]}` : undefined}
    >
      {reason === 'notInTeam' &&
        (access.has('people.manage') ? (
          <p className={styles.notice}>
            You're not in a team yet. Plans belong to teams — add yourself to one on the{' '}
            <Link to="/settings/teams">Teams page</Link>.
          </p>
        ) : (
          <p className={styles.notice}>
            You're not in a team yet. Plans belong to teams — ask an administrator
            {admins ? ` (${admins})` : ''} to add you to one.
          </p>
        ))}
      {/* role="list": some screen readers drop the list when its bullets are styled away. */}
      {steps.length > 0 && (
        <ol role="list" className={styles.steps}>
          {steps.map((step, index) => (
            <li key={step.title} className={styles.step}>
              {/* The list already numbers the steps for screen readers. */}
              <span className={styles.number} aria-hidden="true">
                {index + 1}
              </span>
              <div>
                <Link to={step.to} className={styles.link}>
                  {step.title}
                </Link>
                <p className={styles.text}>{step.text}</p>
              </div>
            </li>
          ))}
        </ol>
      )}
    </Card>
  );
}
