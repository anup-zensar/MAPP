import { Link } from 'react-router';
import { useCurrentUser } from '@/app/useSession';
import { Card, PageHeader } from '@/components';
import { usePageTitle } from '@/lib/usePageTitle';
import styles from './HomePage.module.css';

/** One step of the Plan module, in the order people do them (plan "Home"). */
interface Step {
  title: string;
  to: string;
  text: string;
}

const STEPS: Step[] = [
  {
    title: 'Set up Settings',
    to: '/settings',
    text: 'Add the models, activity types, Shell attributes and currencies your plans use.',
  },
  {
    title: 'Create a plan',
    to: '/plans',
    text: 'Give the plan its year, budget, owner and teams.',
  },
  {
    title: 'Add campaigns and initiatives',
    to: '/plans',
    text: 'Split the plan into campaigns, and each campaign into initiatives with their own budgets.',
  },
  {
    title: 'Add activities and line items',
    to: '/plans',
    text: 'Describe each piece of work and plan its costs as line items.',
  },
  {
    title: 'Submit for approval',
    to: '/approvals',
    text: 'Send activities to their approvers and follow their status.',
  },
];

/**
 * Home (US3, FR-018): greets the person by first name and teaches the order of work of the Plan
 * module in five steps, each linking to its page. Feature 009 replaces this page with the
 * dashboard; nothing else refers to it but the route "/".
 */
export function HomePage() {
  const person = useCurrentUser();
  usePageTitle('Home');

  return (
    <>
      <PageHeader
        eyebrow="Home"
        title="Welcome back,"
        highlight={`${person.firstName}.`}
        description="Here's how planning works in NorthStar — follow the steps in order."
      />
      <Card title="Getting started" subtitle="The Plan module in five steps">
        {/* role="list": some screen readers drop the list when its bullets are styled away. */}
        <ol role="list" className={styles.steps}>
          {STEPS.map((step, index) => (
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
      </Card>
    </>
  );
}
