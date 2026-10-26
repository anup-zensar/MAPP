import type { ReactNode } from 'react';
import { ActionsMenu } from '../ActionsMenu/ActionsMenu';
import type { ActionsMenuItem } from '../ActionsMenu/ActionsMenu';
import { Breadcrumb } from '../Breadcrumb/Breadcrumb';
import type { BreadcrumbItem } from '../Breadcrumb/Breadcrumb';
import styles from './PageHeader.module.css';

export interface PageHeaderProps {
  /**
   * Where the page sits ("Settings › Models"), above the title. A page under a breadcrumb shows
   * no eyebrow: the breadcrumb already names the section.
   */
  breadcrumb?: BreadcrumbItem[];
  /**
   * The section, as a small word above the title on a top-level page ("Settings"). It is not
   * repeated when it is the title itself, nor under a breadcrumb.
   */
  eyebrow?: string;
  /** Shown in the eyebrow's place: a record's level tag (<LevelTag level="plan" />). */
  tag?: ReactNode;
  /** The page's main heading (the only <h1> on the page). */
  title: string;
  /** An optional phrase after the title, in grey ("Welcome back," + "Priya."). */
  highlight?: string;
  /** Facts under the title: status and completeness chips, the ID, the owner, the dates. */
  meta?: ReactNode;
  /** One line that says what the page is for. */
  description?: string;
  /**
   * The page's buttons, on the right of the title: secondary buttons first, then at most one
   * primary button, which ends up rightmost.
   */
  actions?: ReactNode;
  /** Less-used and destructive actions: the "More" menu, left of the buttons (Delete last). */
  moreActions?: ActionsMenuItem[];
}

/**
 * The one header of every signed-in page (design system §2, §6.3): breadcrumb, eyebrow or level
 * tag, the title with its actions on the right ("More", secondary, primary), the facts and the
 * description. List, details and form pages all use it, so titles sit in the same place.
 */
export function PageHeader({
  breadcrumb,
  eyebrow,
  tag,
  title,
  highlight,
  meta,
  description,
  actions,
  moreActions,
}: PageHeaderProps) {
  const showEyebrow =
    eyebrow !== undefined && !breadcrumb && eyebrow.toLowerCase() !== title.toLowerCase();
  const hasMore = moreActions !== undefined && moreActions.length > 0;

  return (
    <header className={styles.header}>
      {breadcrumb && <Breadcrumb items={breadcrumb} />}
      {tag ? (
        <div className={styles.tag}>{tag}</div>
      ) : (
        showEyebrow && <p className={styles.eyebrow}>{eyebrow}</p>
      )}
      <div className={styles.titleRow}>
        <h1 className={styles.title}>
          {title}
          {/* The space stays outside the span, so screen readers keep it between the words. */}
          {highlight && ' '}
          {highlight && <span className={styles.highlight}>{highlight}</span>}
        </h1>
        {(actions || hasMore) && (
          <div className={styles.actions}>
            {hasMore && <ActionsMenu label={title} items={moreActions} trigger="more" />}
            {actions}
          </div>
        )}
      </div>
      {meta && <div className={styles.meta}>{meta}</div>}
      {description && <p className={styles.description}>{description}</p>}
    </header>
  );
}
