import { ChevronDown, LogOut, UserRound } from 'lucide-react';
import { DropdownMenu } from 'radix-ui';
import { Link } from 'react-router';
import { Avatar } from '@/ui/Avatar/Avatar';
import { Tooltip } from '@/ui/Tooltip/Tooltip';
import styles from './AccountMenu.module.css';
import { MY_PROFILE } from './navigation';
import { useCurrentUser, useSession } from '@/session/useSession';

/**
 * The account button at the right of the top bar and its menu (FR-012): who is signed in —
 * name, e-mail and job title — then My profile and Sign out. The button shows the avatar and the
 * first name (the avatar alone on phones, where the name stays in the button's name).
 */
export function AccountMenu() {
  const person = useCurrentUser();
  const { signOut } = useSession();

  return (
    // Not modal, like every MAPP menu: the rest of the page stays usable while it is open.
    <DropdownMenu.Root modal={false}>
      {/* The first name may be cut short on the button; the tooltip has everything in full. */}
      <Tooltip content={[person.name, person.jobTitle].filter(Boolean).join(' — ')}>
        <DropdownMenu.Trigger asChild>
          {/* The button's name comes from its text, "Priya — account menu", so it always contains
              the words written on it (WCAG 2.5.3, label in name). */}
          <button type="button" className={styles.button}>
            <Avatar initials={person.initials} />
            <span className={styles.name}>{person.firstName}</span>{' '}
            <span className="visually-hidden">— account menu</span>
            <ChevronDown className={styles.chevron} aria-hidden="true" />
          </button>
        </DropdownMenu.Trigger>
      </Tooltip>
      <DropdownMenu.Portal>
        <DropdownMenu.Content className={styles.menu} side="bottom" align="end" sideOffset={8}>
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
              <UserRound aria-hidden="true" />
              {MY_PROFILE.label}
            </Link>
          </DropdownMenu.Item>
          <DropdownMenu.Item className={styles.item} onSelect={() => signOut()}>
            <LogOut aria-hidden="true" />
            Sign out
          </DropdownMenu.Item>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
}
