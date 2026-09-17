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
  const features = `width=${width},height=${height}`;
  let win = window.open('', `mudpuppy-${key}`, features);
  if (!win) return null; // blocked — caller shows the red notice and stays docked

  // Code review WR-13 of Phase 5: `mudpuppy-${key}` is a NAMED window, so
  // when one is already open under that name the browser hands it back
  // instead of opening a new one. After a refresh of the play screen that
  // window is an orphan: it still shows its last contents, but every button
  // in it belonged to the page that is gone and does nothing. It used to be
  // reused as it stood -- the stylesheets appended a second time, the live
  // view painted underneath the frozen copy. It is always emptied first now,
  // so what the owner sees is only ever the live view.
  if (!resetPopoutDocument(win)) {
    // The window could not be emptied (it is no longer a blank same-origin
    // page). Never paint into it: close it and open a fresh one under a name
    // nothing else has used.
    closePopout(win);
    win = window.open('', `mudpuppy-${key}-${Date.now()}`, features);
    if (!win || !resetPopoutDocument(win)) {
      closePopout(win);
      return null;
    }
  }

  win.document.title = title;
  win.document.body.style.margin = '0';
  win.document.body.style.background = 'var(--color-bg)';
  // Copy every stylesheet link and inline <style> tag from this document so the
  // popout renders with the exact same tokens, typography and colour buckets —
  // no separate CSS file, no drift.
  const head = win.document.head;
  document.querySelectorAll('link[rel="stylesheet"], style').forEach((node) => {
    head.appendChild(node.cloneNode(true));
  });
  return win;
}

/**
 * Empties a pop-out window's head and body so nothing from an earlier use
 * survives. Returns false when the document cannot be reached or changed.
 */
function resetPopoutDocument(win: Window): boolean {
  try {
    const doc = win.document;
    if (!doc || !doc.head || !doc.body) return false;
    doc.head.replaceChildren();
    doc.body.replaceChildren();
    doc.body.removeAttribute('style');
    doc.body.removeAttribute('class');
    doc.documentElement.removeAttribute('class');
    return true;
  } catch {
    return false;
  }
}

/**
 * Closes every window in the list. Meant for the play screen's own
 * `pagehide` / `beforeunload`: a refresh, a closed tab or a navigation away
 * never runs a React cleanup, so without this the pop-outs were left behind
 * (code review WR-13 of Phase 5; D-29: a pop-out "lives only while the play
 * screen tab is open").
 */
export function closePopouts(wins: Array<Window | null>): void {
  wins.forEach((win) => closePopout(win));
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
