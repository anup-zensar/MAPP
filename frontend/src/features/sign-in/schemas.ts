import { z } from 'zod';
import type { SessionEnd } from '@/session/useSession';

/** E-mail and password: both are needed before the API is asked anything (FR-004). */
export const signInSchema = z.object({
  email: z.string().refine((email) => email.trim() !== '', 'Enter your e-mail'),
  password: z.string().min(1, 'Enter the password'),
});

export type SignInValues = z.infer<typeof signInSchema>;

const noticeStateSchema = z.object({ notice: z.enum(['signedOut', 'sessionLost']) });

/** How the session ended, from the router state endSession sends along; null for anything else. */
export function noticeOf(routerState: unknown): SessionEnd | null {
  const parsed = noticeStateSchema.safeParse(routerState);
  return parsed.success ? parsed.data.notice : null;
}
