/**
 * Keybinding Service (SP04PH04)
 * 
 * Centralized keybinding management for MUDPuppy.
 * Handles canonicalization, lookup, and collision resolution.
 * 
 * Keybinding format: [Modifier+]*Key (e.g., "Ctrl+Shift+F1", "Alt+W", "F1")
 * Modifier order: Ctrl, Alt, Shift, Meta
 * 
 * All keybinding keys are normalized to lowercase for consistent matching.
 */

// Modifier order for canonicalization (most specific first)
const MODIFIER_ORDER = ['ctrl', 'alt', 'shift', 'meta'];

// Special key name mappings for normalization
const SPECIAL_KEY_MAP: Record<string, string> = {
  ' ': 'space',
  'arrowup': 'up',
  'arrowdown': 'down',
  'arrowleft': 'left',
  'arrowright': 'right',
  'escape': 'escape',
  'enter': 'enter',
  'tab': 'tab',
  'backspace': 'backspace',
  'delete': 'delete',
  'home': 'home',
  'end': 'end',
  'pageup': 'pageup',
  'pagedown': 'pagedown',
  'insert': 'insert',
};

/**
 * Normalize a modifier string to canonical form
 */
function normalizeModifier(modifier: string): string {
  const lower = modifier.toLowerCase();
  if (lower === 'control') return 'ctrl';
  if (lower === 'option') return 'alt';
  if (lower === 'command' || lower === 'cmd') return 'meta';
  return lower;
}

/**
 * Normalize a key name to canonical form
 */
function normalizeKeyName(key: string): string {
  const lower = key.toLowerCase();
  
  // Check special key mappings
  if (SPECIAL_KEY_MAP[lower]) {
    return SPECIAL_KEY_MAP[lower];
  }
  
  // Function keys (f1-f12)
  if (/^f\d+$/.test(lower)) {
    return lower;
  }
  
  // Single alphanumeric characters
  if (/^[a-z0-9]$/.test(lower)) {
    return lower;
  }
  
  // Other keys - return lowercase
  return lower;
}

/**
 * Parse a keybinding string into components
 * Returns { modifiers: string[], key: string } or null if invalid
 */
export function parseKeybinding(binding: string): { modifiers: string[], key: string } | null {
  if (!binding || typeof binding !== 'string') {
    return null;
  }
  
  const parts = binding.split('+').map(p => p.trim());
  
  if (parts.length === 0) {
    return null;
  }
  
  // Last part is the key, rest are modifiers
  const key = parts[parts.length - 1];
  const modifiers = parts.slice(0, -1);
  
  if (!key) {
    return null;
  }
  
  return {
    modifiers: modifiers.map(normalizeModifier),
    key: normalizeKeyName(key),
  };
}

/**
 * Convert parsed binding back to canonical string
 */
export function canonicalizeBinding(binding: { modifiers: string[], key: string }): string {
  // Sort modifiers in canonical order
  const sortedModifiers = [...binding.modifiers].sort((a, b) => {
    const aIndex = MODIFIER_ORDER.indexOf(a);
    const bIndex = MODIFIER_ORDER.indexOf(b);
    return aIndex - bIndex;
  });
  
  if (sortedModifiers.length > 0) {
    return [...sortedModifiers, binding.key].join('+');
  }
  return binding.key;
}

/**
 * Canonicalize a keybinding string to standard format
 * This should be used when saving keybindings and when matching
 */
export function canonicalizeKeybinding(binding: string): string | null {
  const parsed = parseKeybinding(binding);
  if (!parsed) {
    return null;
  }
  return canonicalizeBinding(parsed);
}

/**
 * Convert a keyboard event to canonical binding key
 */
export function eventToCanonicalKey(event: KeyboardEvent): string | null {
  const key = event.key.toLowerCase();
  
  // Ignore modifier-only keys
  if (key === 'control' || key === 'alt' || key === 'shift' || key === 'meta') {
    return null;
  }
  
  // Build modifier list
  const modifiers: string[] = [];
  if (event.ctrlKey) modifiers.push('ctrl');
  if (event.altKey) modifiers.push('alt');
  if (event.shiftKey) modifiers.push('shift');
  if (event.metaKey) modifiers.push('meta');
  
  // Normalize key name
  const keyName = normalizeKeyName(key);
  
  // Sort modifiers in canonical order
  const sortedModifiers = [...modifiers].sort((a, b) => {
    const aIndex = MODIFIER_ORDER.indexOf(a);
    const bIndex = MODIFIER_ORDER.indexOf(b);
    return aIndex - bIndex;
  });
  
  if (sortedModifiers.length > 0) {
    return [...sortedModifiers, keyName].join('+');
  }
  return keyName;
}

/**
 * Find the best matching keybinding for a given canonical key
 * Implements "most specific match wins" collision resolution
 * 
 * Rules:
 * 1. Prefer bindings with more modifiers (more specific)
 * 2. If tied, prefer exact match
 * 3. If still tied, lexical order (deterministic)
 */
export function findMatchingBinding(
  canonicalKey: string,
  keybindings: Record<string, string>
): string | null {
  // First, try exact match
  if (keybindings[canonicalKey]) {
    return keybindings[canonicalKey];
  }
  
  // Parse the incoming key to get its modifier count
  const incomingParsed = parseKeybinding(canonicalKey);
  if (!incomingParsed) return null;
  
  const incomingModifierCount = incomingParsed.modifiers.length;
  
  // Find all bindings that could match (same key, different modifiers)
  // We need to check if the canonicalKey could match any stored bindings
  // For example: "ctrl+f1" should match stored "f1" (less specific)
  
  let bestMatch: { binding: string; command: string; specificity: number } | null = null;
  
  for (const [bindingKey, command] of Object.entries(keybindings)) {
    // Canonicalize the stored binding
    const canonicalStored = canonicalizeKeybinding(bindingKey);
    if (!canonicalStored) continue;
    
    // Check if this binding matches
    // Exact match already handled above
    if (canonicalStored === canonicalKey) continue;
    
    // Check if it's a partial match (same key, different modifiers)
    const storedParsed = parseKeybinding(canonicalStored);
    if (!storedParsed) continue;
    
    // The key must match
    if (storedParsed.key !== incomingParsed.key) continue;
    
    // Calculate specificity: more modifiers = more specific
    // We prefer LESS specific when the user presses a modifier combo
    // Example: User presses Ctrl+F1, we prefer Ctrl+F1 over F1
    const storedModifierCount = storedParsed.modifiers.length;
    
    // Calculate "match quality"
    // If user presses Ctrl+F1 and we have Ctrl+F1, that's perfect (both have same modifiers)
    // If user presses Ctrl+F1 and we have F1, that's a fallback
    
    // We want: prefer exact key match, then prefer more modifiers matching
    const modifierMatch = storedModifierCount === incomingModifierCount ? 2 : 
                         (incomingModifierCount > storedModifierCount && 
                          storedParsed.modifiers.every(m => incomingParsed.modifiers.includes(m))) ? 1 : 0;
    
    if (modifierMatch === 0) continue;
    
    // This is a candidate - compare specificity
    // Higher specificity score wins
    const specificity = storedModifierCount * 10 + modifierMatch;
    
    if (!bestMatch || specificity > bestMatch.specificity) {
      bestMatch = { binding: canonicalStored, command, specificity };
    } else if (specificity === bestMatch.specificity) {
      // Tie-break with lexical order
      if (canonicalStored < bestMatch.binding) {
        bestMatch = { binding: canonicalStored, command, specificity };
      }
    }
  }
  
  return bestMatch?.command ?? null;
}

/**
 * Normalize all keybindings in a profile to canonical format
 * Call this when loading profile or before saving
 */
export function normalizeKeybindings(keybindings: Record<string, string>): Record<string, string> {
  const normalized: Record<string, string> = {};
  
  for (const [key, command] of Object.entries(keybindings)) {
    const canonical = canonicalizeKeybinding(key);
    if (canonical) {
      normalized[canonical] = command;
    }
  }
  
  return normalized;
}

/**
 * Validate a keybinding format
 * Returns true if valid, false otherwise
 */
export function isValidKeybindingFormat(binding: string): boolean {
  const parsed = parseKeybinding(binding);
  return parsed !== null && parsed.key.length > 0;
}

/**
 * Validate a command string
 * Returns true if valid, false otherwise
 */
export function isValidCommand(command: string): boolean {
  // Must be non-empty string
  if (!command || typeof command !== 'string') {
    return false;
  }
  
  // Max 500 characters
  if (command.length > 500) {
    return false;
  }
  
  return true;
}
