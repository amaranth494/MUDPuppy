import { useCallback, useEffect, useRef } from 'react';
import { useSession } from '../context/SessionContext';

/**
 * Input Interceptor Hook (SP04PH03)
 * 
 * Intercepts keyboard events to check for keybinding matches.
 * Returns the matched command if a keybinding exists, null otherwise.
 * 
 * Keybinding format: [Modifier+]*Key (e.g., "Ctrl+Shift+F1", "Alt+W", "F1")
 * Modifier order: Ctrl, Alt, Shift, Meta
 */

// Normalize modifier order
function normalizeModifiers(modifiers: Set<string>): string[] {
  const order = ['ctrl', 'alt', 'shift', 'meta'];
  return order.filter(m => modifiers.has(m));
}

// Convert keyboard event to canonical binding key
function eventToBindingKey(event: KeyboardEvent): string | null {
  const key = event.key.toLowerCase();
  
  // Ignore modifier-only keys
  if (key === 'control' || key === 'alt' || key === 'shift' || key === 'meta') {
    return null;
  }
  
  // Build modifier list
  const modifiers = new Set<string>();
  if (event.ctrlKey) modifiers.add('ctrl');
  if (event.altKey) modifiers.add('alt');
  if (event.shiftKey) modifiers.add('shift');
  if (event.metaKey) modifiers.add('meta');
  
  // Normalize key name
  let keyName = key;
  
  // Handle special keys
  if (key === ' ') {
    keyName = 'space';
  } else if (key === 'arrowup') {
    keyName = 'up';
  } else if (key === 'arrowdown') {
    keyName = 'down';
  } else if (key === 'arrowleft') {
    keyName = 'left';
  } else if (key === 'arrowright') {
    keyName = 'right';
  } else if (key === 'escape') {
    keyName = 'escape';
  } else if (key === 'enter') {
    keyName = 'enter';
  } else if (key === 'tab') {
    keyName = 'tab';
  } else if (key === 'backspace') {
    keyName = 'backspace';
  } else if (key === 'delete') {
    keyName = 'delete';
  } else if (key === 'home') {
    keyName = 'home';
  } else if (key === 'end') {
    keyName = 'end';
  } else if (key === 'pageup') {
    keyName = 'pageup';
  } else if (key === 'pagedown') {
    keyName = 'pagedown';
  } else if (key === 'insert') {
    keyName = 'insert';
  } else if (/^f\d+$/.test(key)) {
    // Function keys (f1-f12)
    keyName = key;
  } else if (/^[a-z0-9]$/.test(key)) {
    // Single alphanumeric characters
    keyName = key;
  } else {
    // Other special characters - use as-is but lowercase
    keyName = key;
  }
  
  // Build the canonical binding key
  const normalizedModifiers = normalizeModifiers(modifiers);
  if (normalizedModifiers.length > 0) {
    return [...normalizedModifiers, keyName].join('+');
  }
  return keyName;
}

// Find the most specific matching keybinding
// (longest modifier chain wins)
function findMatchingBinding(
  key: string,
  keybindings: Record<string, string>
): string | null {
  // First, try exact match
  if (keybindings[key]) {
    return keybindings[key];
  }
  
  // If no exact match, return null
  return null;
}

interface UseInputInterceptorOptions {
  // Callback when a keybinding is triggered
  onKeybindingDispatch?: (command: string) => void;
  // Whether the input is currently enabled
  enabled?: boolean;
}

/**
 * Input Interceptor Hook
 * 
 * Must be used within a SessionProvider.
 * Handles keybinding lookup and dispatch.
 */
export function useInputInterceptor(options: UseInputInterceptorOptions = {}) {
  const { onKeybindingDispatch, enabled = true } = options;
  const { profile, isInputLocked, connectionState } = useSession();
  
  // Track key state to prevent auto-repeat
  const keysPressed = useRef<Set<string>>(new Set());
  
  // Handle keyboard events
  const handleKeyDown = useCallback((event: KeyboardEvent) => {
    // Don't intercept if:
    // 1. Input is locked (modal active)
    // 2. Not connected
    // 3. No keybindings defined
    // 4. Disabled
    if (isInputLocked || connectionState !== 'connected' || !profile?.keybindings || !enabled) {
      return;
    }
    
    // Check if this is a key repeat (key already pressed)
    const eventKey = event.key.toLowerCase();
    if (keysPressed.current.has(eventKey)) {
      // Key is already pressed - this is auto-repeat, ignore
      event.preventDefault();
      return;
    }
    
    // Convert event to binding key
    const bindingKey = eventToBindingKey(event);
    if (!bindingKey) {
      return;
    }
    
    // Mark key as pressed
    keysPressed.current.add(eventKey);
    
    // Look up the binding
    const command = findMatchingBinding(bindingKey, profile.keybindings);
    if (command) {
      // Prevent default browser behavior
      event.preventDefault();
      
      // Dispatch the command
      if (onKeybindingDispatch) {
        onKeybindingDispatch(command);
      }
    }
  }, [profile, isInputLocked, connectionState, enabled, onKeybindingDispatch]);
  
  // Handle key up to track key release
  const handleKeyUp = useCallback((event: KeyboardEvent) => {
    const eventKey = event.key.toLowerCase();
    keysPressed.current.delete(eventKey);
  }, []);
  
  // Set up event listeners
  useEffect(() => {
    // Only add listener when connected and profile has keybindings
    if (connectionState !== 'connected' || !profile?.keybindings) {
      return;
    }
    
    window.addEventListener('keydown', handleKeyDown);
    window.addEventListener('keyup', handleKeyUp);
    
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
      window.removeEventListener('keyup', handleKeyUp);
    };
  }, [handleKeyDown, handleKeyUp, connectionState, profile]);
  
  return {
    // Current keybindings from profile
    keybindings: profile?.keybindings || {},
    // Whether intercept is active
    isActive: connectionState === 'connected' && !isInputLocked && enabled,
  };
}
