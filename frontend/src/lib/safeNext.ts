/**
 * Where to go after signing in (FR-005, research R15). This is the only place that decides it.
 *
 * `raw` is the "next" address the sign-in page received. It is used only when it is a page of
 * NorthStar itself and not the sign-in page; anything else — another site, a javascript: address,
 * nothing at all — goes to Home. That way a link to the sign-in page can never send people to
 * another site after they sign in.
 */
export function safeNext(raw: string | null | undefined): string {
  if (!raw) {
    return '/';
  }

  // The browser's own address parser decides, so tricks such as "//evil.com", "/\evil.com" or a
  // tab inside the address are read exactly as the browser would read them.
  let url: URL;
  try {
    url = new URL(raw, window.location.origin);
  } catch {
    return '/';
  }

  if (url.origin !== window.location.origin || isSignInPage(url.pathname)) {
    return '/';
  }
  return url.pathname + url.search + url.hash;
}

/** The router opens the sign-in page also for "/sign-in/" and "/SIGN-IN". */
function isSignInPage(pathname: string): boolean {
  return pathname.toLowerCase().replace(/\/+$/, '') === '/sign-in';
}
