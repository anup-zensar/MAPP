import type { ComponentProps, ReactNode } from 'react';
import { Link } from 'react-router';
import { cx } from '@/lib/cx';
import styles from './Button.module.css';

/**
 * primary: the page's one main action (solid blue). secondary: everything else (outlined).
 * danger: an action that removes something (red text and outline). ghost: text only, for quiet
 * actions such as an icon button in a header.
 */
export type ButtonVariant = 'primary' | 'secondary' | 'danger' | 'ghost';
export type ButtonSize = 'md' | 'sm';

/** How a Button or LinkButton looks. */
interface ButtonLook {
  variant?: ButtonVariant;
  size?: ButtonSize;
  /** A lucide icon shown before the label. Alone, it needs an aria-label to name the button. */
  icon?: ReactNode;
}

export interface ButtonProps extends ComponentProps<'button'>, ButtonLook {}

export interface LinkButtonProps extends ComponentProps<typeof Link>, ButtonLook {}

/**
 * NorthStar's button (design system §5): label it with a verb ("Create campaign"), never "OK".
 * Never disable it without saying why next to it (constitution III).
 */
export function Button({
  variant = 'secondary',
  size = 'md',
  icon,
  className,
  children,
  type = 'button',
  ...buttonProps
}: ButtonProps) {
  return (
    <button
      type={type}
      className={buttonClassName(variant, size, children == null, className)}
      {...buttonProps}
    >
      {icon}
      {children}
    </button>
  );
}

/** A link that looks like a Button, for actions that open another page ("Go to Home"). */
export function LinkButton({
  variant = 'secondary',
  size = 'md',
  icon,
  className,
  children,
  ...linkProps
}: LinkButtonProps) {
  return (
    <Link className={buttonClassName(variant, size, children == null, className)} {...linkProps}>
      {icon}
      {children}
    </Link>
  );
}

function buttonClassName(
  variant: ButtonVariant,
  size: ButtonSize,
  iconOnly: boolean,
  className: string | undefined,
): string {
  return cx(styles.button, styles[variant], styles[size], iconOnly && styles.iconOnly, className);
}
