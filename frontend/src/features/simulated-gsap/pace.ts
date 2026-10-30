// The simulated GSAP answers in milliseconds; waiting a moment shows the request going to an outside system.
export const gsapPace = { ms: import.meta.env.MODE === 'test' ? 0 : 1500 };

export function waitForGsap(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, gsapPace.ms));
}
