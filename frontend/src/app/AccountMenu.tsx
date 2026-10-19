import { LogOut, UserRound } from 'lucide-react';
import { DropdownMenu } from 'radix-ui';
import { Link } from 'react-router';
import { Avatar, Tooltip } from '@/components';
import { cx } from '@/lib/cx';
import styles from './AccountMenu.module.css';
import { MY_PROFILE } from './navigation';
import { useCurrentUser, useSession } from './useSession';

export interface AccountMenuProps {
  /**
   * How the button looks where it is: "full" (avatar and first name) in the sidebar, "rail"
   * (avatar only) in the collapsed sidebar, "drawer" (avatar and the full name, without a
   * tooltip, which would swallow the first Escape) in the menu drawer.
   */
  variant: 'full' | 'rail' | 'drawer';
}

/**
 * The account button and its menu (FR-012): who is signed in — name, e-mail and job title —
 * then My profile and Sign out.
 */
export function AccountMenu({ variant }: AccountMenuProps) {
  const person = useCurrentUser();
  const { signOut } = useSession();
  // The full name in the drawer, the first name elsewhere.
  const shownName = variant === 'drawer' ? person.name : person.firstName;

  // The button's name comes from its text, "Priya — account menu", so it always contains the words
  // written on it (WCAG 2.5.3, label in name). In the icon rail the name is there but hidden.
  const trigger = (
    <DropdownMenu.Trigger asChild>
      <button type="button" className={cx(styles.button, styles[variant])}>
        <Avatar initials={person.initials} />
        <span className={variant === 'rail' ? 'visually-hidden' : styles.name}>
          {shownName}
        </span>{' '}
        <span className="visually-hidden">— account menu</span>
      </button>
    </DropdownMenu.Trigger>
  );

  return (
    // Not modal, like every NorthStar menu: the rest of the page stays usable while it is open.
    <DropdownMenu.Root modal={false}>
      {variant === 'drawer' ? (
        trigger
      ) : (
        // The first name may be cut short on the button; the tooltip has everything in full.
        <Tooltip content={[person.name, person.jobTitle].filter(Boolean).join(' — ')}>
          {trigger}
        </Tooltip>
      )}
      <DropdownMenu.Portal>
        <DropdownMenu.Content className={styles.menu} side="top" align="start" sideOffset={8}>
          {/* A label, not an item: it tells who is signed in and cannot be chosen. */}
          <DropdownMenu.Label className={styles.header}>
            <Avatar initials={person.initials} size="lg" />
            <span className={styles.details}>
              <span className={styles.fullName}>{person.name}</span>
              <span className={styles.detail}>{person.email}</span>
              <span className={styles.detail}>{person.jobTitle}</span>
            </span>
          </DropdownMenu.Label>
          <DropdownMenu.Separator className={styles.separator} />
          <DropdownMenu.Item asChild className={styles.item}>
            <Link to={MY_PROFILE.path}>
              <UserRound />
              {MY_PROFILE.label}
            </Link>
          </DropdownMenu.Item>
          <DropdownMenu.Item className={styles.item} onSelect={() => signOut()}>
            <LogOut />
            Sign out
          </DropdownMenu.Item>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
}
