// chatLines.ts — the conversation list's own rules, kept as plain functions
// with no React and no DOM in them so they can be read, and checked, on their
// own (code review WR-12 of Phase 5).
import { ChatLine } from '../types';

/**
 * The React key for one conversation line.
 *
 * Code review WR-12 of Phase 5: the list used `entry.id` alone, and the
 * server's system notices arrived with an empty id, so two notices gave two
 * children the same key and React reused and dropped nodes unpredictably.
 * The server now gives every line an id; the index fallback stays so an
 * empty id from any source can never collide again. Lines are only ever
 * appended, so an index is stable for the life of the list.
 */
export function chatLineKey(entry: ChatLine, index: number): string {
  return entry.id || `no-id-${index}`;
}

/**
 * Adds one pushed line to the list, unless a line with the same id is
 * already there.
 *
 * The same line can reach the panel twice: once in the reload that runs when
 * the panel attaches, and once as the live push, when the two overlap. A line
 * with no id can never be recognised as a repeat, so it is always added.
 */
export function appendChatLine(list: ChatLine[], entry: ChatLine): ChatLine[] {
  if (entry.id && list.some((existing) => existing.id === entry.id)) {
    return list;
  }
  return [...list, entry];
}

/**
 * Combines the reloaded conversation with whatever arrived live while the
 * reload was in flight.
 *
 * The reload used to REPLACE the list, so a push that landed before it
 * resolved was erased. Now the stored lines come first, in the server's
 * order, followed by every live line the reload does not already contain:
 * notices, lines that could not be saved, and anything newer than the reload.
 */
export function mergeLoadedConversation(loaded: ChatLine[], live: ChatLine[]): ChatLine[] {
  const loadedIds = new Set(loaded.map((line) => line.id).filter((id) => id !== ''));
  const merged: ChatLine[] = [];
  for (const line of loaded) {
    merged.push(line);
  }
  for (const line of live) {
    if (line.id && loadedIds.has(line.id)) continue;
    merged.push(line);
  }
  return merged;
}
