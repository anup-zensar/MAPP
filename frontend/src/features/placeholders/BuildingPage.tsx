import { Construction } from 'lucide-react';
import { useState } from 'react';
import type { ReactNode } from 'react';
import { useNavigate } from 'react-router';
import { Breadcrumb, Button, Card, EmptyState, PageHeader, SidePanel } from '@/components';
import type { BreadcrumbItem } from '@/components';
import { usePageTitle } from '@/lib/usePageTitle';
import styles from './BuildingPage.module.css';

/** What a stub says until its feature builds it. */
export const BEING_BUILT = 'This page is being built.';

export interface BuildingPageProps {
  /** The small word above the title, as the feature's plan has it ("Settings", "Plans"). */
  eyebrow: string;
  /** The page's title from the feature's plan; also the browser tab's, unless `tabTitle` is given. */
  title: string;
  description?: string;
  breadcrumb?: BreadcrumbItem[];
  /** The browser tab's title when it differs from `title`; null leaves the tab to a panel on top. */
  tabTitle?: string | null;
  /** Actions in the page header (another feature's button placed here). */
  actions?: ReactNode;
  /** Sections other features show on this page, placed where the plans put them. */
  children?: ReactNode;
}

/**
 * The stub of a page whose feature (002–008) is being built (Plan-module shared foundation): the
 * page's title and description, and "This page is being built." The feature replaces the stub's
 * file with the real page; the route stays.
 */
export function BuildingPage({
  eyebrow,
  title,
  description,
  breadcrumb,
  tabTitle,
  actions,
  children,
}: BuildingPageProps) {
  usePageTitle(tabTitle === undefined ? title : tabTitle);
  return (
    <>
      {breadcrumb && <Breadcrumb items={breadcrumb} />}
      <PageHeader eyebrow={eyebrow} title={title} description={description} actions={actions} />
      <div className={styles.stack}>
        <Card>
          <EmptyState icon={<Construction />} title={BEING_BUILT} />
        </Card>
        {children}
      </div>
    </>
  );
}

export interface BuildingPanelProps {
  title: string;
  description?: string;
  /** Where closing the panel leads: the page under it. */
  closeTo: string;
  /** The footer button that closes it: "Back to the plan". */
  closeLabel: string;
  /** Sections other features show in this panel. */
  children?: ReactNode;
}

/**
 * The stub of a side panel with its own address (feature 005's campaign and initiative panels
 * over the plan page): its title, "This page is being built." and a button back to the page.
 */
export function BuildingPanel({
  title,
  description,
  closeTo,
  closeLabel,
  children,
}: BuildingPanelProps) {
  usePageTitle(title);
  const navigate = useNavigate();
  const [open, setOpen] = useState(true);

  function close() {
    setOpen(false);
    void navigate(closeTo);
  }

  return (
    <SidePanel
      open={open}
      onOpenChange={(next) => !next && close()}
      title={title}
      description={description}
      footer={<Button onClick={close}>{closeLabel}</Button>}
    >
      <div className={styles.stack}>
        <EmptyState icon={<Construction />} title={BEING_BUILT} />
        {children}
      </div>
    </SidePanel>
  );
}

export interface BuildingSectionProps {
  /** The section's title, as the plans name it: "Line items", "Approval". */
  title: string;
}

/**
 * The stub of a section one feature shows inside another feature's page (the line items and the
 * approval of an activity, the activities of an initiative): its title and "This section is being
 * built.".
 */
export function BuildingSection({ title }: BuildingSectionProps) {
  return (
    <Card title={title}>
      <p className={styles.text}>This section is being built.</p>
    </Card>
  );
}
