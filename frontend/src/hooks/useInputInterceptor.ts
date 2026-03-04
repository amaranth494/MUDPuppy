import { useCallback, useEffect, useRef } from 'react';
import { useSession } from '../context/SessionContext';
import { eventToCanonicalKey, findMatchingBinding } from '../services/keybindings';

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
    
    // Convert event to canonical binding key
    const bindingKey = eventToCanonicalKey(event);
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
