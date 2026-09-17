// popout.ts — D-29: the owner can put the AI-player view or the AI-chatter
// view into its own separate browser window. This is the only `window.open`
// in the codebase and the only place a second document is created; every
// other window-like surface in the app is a route the router already knows
// about. The child window has no URL of its own to sign into and no router
// to match — it is blank HTML that the existing React tree paints into, in
// the same JavaScript process, subscribed to the exact same wsManager/
// useSession() instance the docked panel already uses (05-UI-SPEC.md §7).

/**
 * Opens a same-origin, blank child window for the given view and clones this
 * document's styling into it so it renders with the exact same tokens,
 * typography and colour buckets — no separate CSS file, no drift.
 *
 * Must be called synchronously inside the button's onClick — not after an
 * await or inside a promise callback — because that is what makes the
 * browser's pop-up blocker treat the window as a direct result of the
 * click rather than an unrequested pop-up.
 *
 * Returns null when the browser blocked it, so the caller can show the
 * locked pop-up-blocked notice and leave the view docked.
 */
export function openPopout(key: 'ai-player' | 'ai-chatter', title: string, width: number, height: number): Window | null {
  const win = window.open('', `mudpuppy-${key}`, `width=${width},height=${height}`);
  if (!win) return null; // blocked — caller shows the red notice and stays docked
  win.document.title = title;
  win.document.body.style.margin = '0';
  win.document.body.style.background = 'var(--color-bg)';
  // Copy every stylesheet link and inline <style> tag from this document so the
  // popout renders with the exact same tokens, typography and colour buckets —
  // no separate CSS file, no drift.
  document.querySelectorAll('link[rel="stylesheet"], style').forEach((node) => {
    win.document.head.appendChild(node.cloneNode(true));
  });
  return win;
}

/**
 * Closes a popped-out window, tolerating one the owner already closed by
 * hand (its own native close button) before this was called.
 */
export function closePopout(win: Window | null): void {
  if (win && !win.closed) {
    win.close();
  }
}
