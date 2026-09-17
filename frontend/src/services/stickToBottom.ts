// stickToBottom.ts — the one rule both of the AI Assist panel's scrolling
// lists follow (owner-reported fix OW-01): a list that is showing its newest
// line follows new lines down; a list the owner has scrolled up, to read
// history, is left exactly where he put it. Plain functions, no React.

/**
 * How close to the bottom, in pixels, still counts as "at the bottom". A
 * little slack absorbs sub-pixel rounding and a last line only just out of
 * view, so the list does not come unstuck by accident.
 */
export const STICK_TO_BOTTOM_SLACK_PX = 24;

/** The three numbers the rule needs; every scrolling element has them. */
export interface ScrollMetrics {
  scrollTop: number;
  scrollHeight: number;
  clientHeight: number;
}

/** Whether the list is showing its newest line. */
export function isScrolledToBottom(el: ScrollMetrics, slack: number = STICK_TO_BOTTOM_SLACK_PX): boolean {
  return el.scrollHeight - el.scrollTop - el.clientHeight <= slack;
}

/**
 * Scrolls the list to its newest line, but only when `pinned` says the owner
 * had not scrolled away from it. Returns whether it scrolled.
 */
export function followIfPinned(el: ScrollMetrics | null, pinned: boolean): boolean {
  if (!el || !pinned) return false;
  el.scrollTop = el.scrollHeight;
  return true;
}
