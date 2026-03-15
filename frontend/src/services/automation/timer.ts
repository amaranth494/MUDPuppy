/**
 * Timer System - PR01PH04
 * 
 * Handles timer creation, execution, and cancellation for automation:
 * - #TIMER name duration - create/update timer definition (starts in STOP state)
 * - #TIMER name duration REPEAT - create/update repeating timer
 * - #START timerName - start a stopped timer
 * - #STOP timerName - stop a running timer
 * - #CHECK timerName - check timer status
 * - #CANCEL timerName - cancel and remove a timer
 * 
 * Timers are connection-scoped and cleared on disconnect.
 * Timer definitions are persisted to profile.
 * Runtime countdowns are session-only (stopped on disconnect).
 */

import { ParsedToken, CommandToken } from './parser';

// ============================================
// Types
// ============================================

// Timer state
export type TimerState = 'stopped' | 'running';

// Saved timer definition (persisted to profile)
export interface SavedTimer {
  name: string;
  duration: number; // in milliseconds
  repeat: boolean;
  commands: string[];
}

// Active runtime timer (session-only)
export interface Timer {
  id: string;
  name: string;
  duration: number; // in milliseconds
  repeat: boolean;
  commands: string[];
  createdAt: number;
  fireCount: number;
  state: TimerState;
  startTime?: number; // when timer started (for #CHECK)
  // PR01PH08: Override behavior for #START single/nonstop
  singleShot?: boolean;  // if true, run once even if repeat is true
  forceRepeat?: boolean; // if true, keep repeating even if repeat is false
}

export interface TimerExecutionContext {
  executeCommands: (commands: string[]) => void;
  substituteVariables: (input: string) => string;
  outputMessage?: (message: string) => void; // For #CHECK output
  // PR01PH07T03: Execute commands through parser for #IF/#ELSE support
  executeThroughParser?: (commands: string[]) => Promise<void>;
}

export interface TimerManagerConfig {
  maxTimers?: number; // Optional max timers configuration
  onTimerFire?: (timer: Timer) => void;
  onTimerCancel?: (timerName: string) => void;
  onTimerSave?: (timer: SavedTimer) => Promise<void>; // Persist to profile
  onTimerDelete?: (timerName: string) => Promise<void>; // Delete from profile
  onError?: (error: string) => void;
}

// ============================================
// Timer Parser Functions
// ============================================

// Duration parsing patterns
const DURATION_REGEX = /^(\d+)(s|m|h)$/i;

/**
 * Parse timer duration string to milliseconds
 * Examples: "10s" -> 10000, "5m" -> 300000, "1h" -> 3600000
 */
export function parseDuration(durationStr: string): number | null {
  const match = durationStr.trim().match(DURATION_REGEX);
  if (!match) return null;
  
  const value = parseInt(match[1], 10);
  const unit = match[2].toLowerCase();
  
  switch (unit) {
    case 's': return value * 1000;
    case 'm': return value * 60 * 1000;
    case 'h': return value * 60 * 60 * 1000;
    default: return null;
  }
}

/**
 * Parse #TIMER command arguments
 * Format: "name duration" or "name duration REPEAT"
 */
export function parseTimerArgs(args: string): { name: string; duration: number; repeat: boolean } | null {
  const parts = args.trim().split(/\s+/);
  
  if (parts.length < 2) return null;
  
  const name = parts[0];
  const durationStr = parts[1];
  const repeat = parts.some(p => p.toUpperCase() === 'REPEAT');
  
  const duration = parseDuration(durationStr);
  if (duration === null) return null;
  
  return { name, duration, repeat };
}

/**
 * Extract commands from between #TIMER and #ENDTIMER tokens
 */
export function extractTimerCommands(tokens: ParsedToken[], startIndex: number, endIndex: number): string[] {
  const commands: string[] = [];
  
  for (let i = startIndex + 1; i < endIndex; i++) {
    const token = tokens[i];
    
    if (token.type === 'TEXT') {
      const trimmed = token.value.trim();
      if (trimmed) {
        commands.push(trimmed);
      }
    }
  }
  
  return commands;
}

/**
 * Find the matching #ENDTIMER for a #TIMER command
 */
export function findTimerEnd(tokens: ParsedToken[], startIndex: number): number | null {
  for (let i = startIndex + 1; i < tokens.length; i++) {
    const token = tokens[i];
    if (token.type === 'COMMAND' && token.command === 'ENDTIMER') {
      return i;
    }
  }
  return null;
}

// ============================================
// Timer Manager
// ============================================

export class TimerManager {
  private timers: Map<string, Timer> = new Map();
  private timerIds: Map<string, number> = new Map(); // name -> setTimeout ID
  private onTimerFire?: (timer: Timer) => void;
  private onTimerCancel?: (timerName: string) => void;
  private onTimerSave?: (timer: SavedTimer) => Promise<void>;
  private onTimerDelete?: (timerName: string) => Promise<void>;
  private onError?: (error: string) => void;
  
  // Context for executing timer commands
  private executionContext: TimerExecutionContext | null = null;
  
  constructor(config: TimerManagerConfig = {}) {
    this.onTimerFire = config.onTimerFire;
    this.onTimerCancel = config.onTimerCancel;
    this.onTimerSave = config.onTimerSave;
    this.onTimerDelete = config.onTimerDelete;
    this.onError = config.onError;
  }

  /**
   * Report an error
   */
  reportError(error: string): void {
    this.onError?.(error);
  }
  
  /**
   * Set the execution context for timer commands
   */
  setExecutionContext(context: TimerExecutionContext): void {
    this.executionContext = context;
  }
  
  /**
   * Get the current number of active timers
   */
  getActiveTimerCount(): number {
    return this.timers.size;
  }
  
  /**
   * Check if a timer with the given name exists
   */
  hasTimer(name: string): boolean {
    return this.timers.has(name);
  }
  
  /**
   * Get a timer by name
   */
  getTimer(name: string): Timer | undefined {
    return this.timers.get(name);
  }
  
  /**
   * Get all active timers
   */
  getAllTimers(): Timer[] {
    return Array.from(this.timers.values());
  }
  
  /**
   * Create or update a timer definition (persisted to profile)
   * Timer starts in STOPPED state by default
   */
  createOrUpdateTimer(name: string, duration: number, repeat: boolean, commands: string[]): { success: boolean; error?: string } {
    // Validate duration
    if (duration <= 0) {
      const error = `Invalid timer duration: ${duration}`;
      this.onError?.(error);
      return { success: false, error };
    }
    
    // Validate commands
    if (commands.length === 0) {
      const error = 'Timer has no commands to execute';
      this.onError?.(error);
      return { success: false, error };
    }
    
    const existingTimer = this.timers.get(name);
    const wasRunning = existingTimer?.state === 'running';
    
    // Stop existing timer if running
    if (existingTimer) {
      this.stopTimerInternal(name);
    }
    
    const timer: Timer = {
      id: `${name}-${Date.now()}`,
      name,
      duration,
      repeat,
      commands,
      createdAt: Date.now(),
      fireCount: 0,
      state: 'stopped' // Always start in STOPPED state
    };
    
    this.timers.set(name, timer);
    
    // Persist to profile
    this.persistTimer(timer);
    
    // If it was running before, restart it
    if (wasRunning) {
      this.startTimerByName(name);
    }
    
    return { success: true };
  }

  /**
   * Create a timer (legacy - for compatibility)
   */
  createTimer(name: string, duration: number, repeat: boolean, commands: string[]): { success: boolean; error?: string } {
    return this.createOrUpdateTimer(name, duration, repeat, commands);
  }

  /**
   * Start a timer by name (#START command)
   */
  startTimer(name: string): { success: boolean; error?: string } {
    return this.startTimerByName(name);
  }

  /**
   * Start a timer in single-shot mode (#START ... single)
   * Runs once even if timer is configured as repeating
   */
  startTimerSingle(name: string): { success: boolean; error?: string } {
    const timer = this.timers.get(name);
    
    if (!timer) {
      const error = `Timer '${name}' not found`;
      this.onError?.(error);
      return { success: false, error };
    }
    
    if (timer.state === 'running') {
      this.stopTimerInternal(name);
    }
    
    // Set single shot flag
    timer.singleShot = true;
    timer.forceRepeat = false;
    
    this.startTimerInternal(timer);
    
    return { success: true };
  }

  /**
   * Start a timer in nonstop mode (#START ... nonstop)
   * Keeps repeating even if timer is configured as non-repeating
   */
  startTimerNonstop(name: string): { success: boolean; error?: string } {
    const timer = this.timers.get(name);
    
    if (!timer) {
      const error = `Timer '${name}' not found`;
      this.onError?.(error);
      return { success: false, error };
    }
    
    if (timer.state === 'running') {
      this.stopTimerInternal(name);
    }
    
    // Set force repeat flag
    timer.forceRepeat = true;
    timer.singleShot = false;
    
    this.startTimerInternal(timer);
    
    return { success: true };
  }

  /**
   * Internal start by name
   */
  private startTimerByName(name: string): { success: boolean; error?: string } {
    const timer = this.timers.get(name);
    
    if (!timer) {
      const error = `Timer '${name}' not found`;
      this.onError?.(error);
      return { success: false, error };
    }
    
    if (timer.state === 'running') {
      // Restart if already running - cancel and restart
      this.stopTimerInternal(name);
    }
    
    // Start the timer
    this.startTimerInternal(timer);
    
    return { success: true };
  }

  /**
   * Stop a timer by name (#STOP command)
   */
  stopTimer(name: string): { success: boolean; error?: string } {
    const timer = this.timers.get(name);
    
    if (!timer) {
      const error = `Timer '${name}' not found`;
      this.onError?.(error);
      return { success: false, error };
    }
    
    this.stopTimerInternal(name);
    
    return { success: true };
  }

  /**
   * Start all timers using saved settings (#START all command)
   */
  startAllTimers(): { success: boolean; output?: string } {
    const startedTimers: string[] = [];
    const skippedTimers: string[] = [];
    
    for (const [name, timer] of this.timers) {
      if (timer.state === 'running') {
        skippedTimers.push(name);
      } else {
        // Clear any override flags and use saved settings
        timer.singleShot = undefined;
        timer.forceRepeat = undefined;
        this.startTimerInternal(timer);
        startedTimers.push(name);
      }
    }
    
    if (startedTimers.length === 0 && skippedTimers.length === 0) {
      return { success: true, output: 'No timers defined.' };
    }
    
    let output = '';
    if (startedTimers.length > 0) {
      output += `Started: ${startedTimers.join(', ')}`;
    }
    if (skippedTimers.length > 0) {
      if (output) output += '; ';
      output += `Already running: ${skippedTimers.join(', ')}`;
    }
    
    return { success: true, output };
  }

  /**
   * List all timers with status (#CHECK all command)
   */
  listAllTimers(): { success: boolean; output?: string } {
    if (this.timers.size === 0) {
      return { success: true, output: 'No timers defined.' };
    }
    
    const lines: string[] = [];
    for (const [name, timer] of this.timers) {
      const durationStr = this.formatDuration(timer.duration);
      let status: string = timer.state === 'running' ? 'running' : 'stopped';
      
      if (timer.state === 'running') {
        const elapsed = Date.now() - (timer.startTime || Date.now());
        const remaining = Math.max(0, timer.duration - elapsed);
        const remainingStr = this.formatDuration(remaining);
        status = `running (${remainingStr} remaining)`;
      }
      
      // Add override indicator
      if (timer.singleShot) {
        status += ' [single]';
      } else if (timer.forceRepeat) {
        status += ' [nonstop]';
      } else if (timer.repeat) {
        status += ' [repeat]';
      }
      
      lines.push(`${name}: ${durationStr} - ${status}`);
    }
    
    return { success: true, output: `Timers:,${lines.join(',')}` };
  }

  /**
   * Format duration in ms to human-readable string
   */
  private formatDuration(ms: number): string {
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    if (ms < 3600000) return `${(ms / 60000).toFixed(1)}m`;
    return `${(ms / 3600000).toFixed(1)}h`;
  }

  /**
   * Check timer status (#CHECK command)
   */
  checkTimer(name: string): { success: boolean; error?: string; output?: string } {
    const timer = this.timers.get(name);
    
    if (!timer) {
      const error = `Timer '${name}' not found`;
      this.onError?.(error);
      return { success: false, error };
    }
    
    if (timer.state === 'running') {
      // Calculate remaining time
      const elapsed = Date.now() - (timer.startTime || Date.now());
      const remaining = Math.max(0, timer.duration - elapsed);
      const seconds = Math.ceil(remaining / 1000);
      
      // Build status string with override info
      let status = `${seconds} seconds`;
      if (timer.singleShot) {
        status += ' (single)';
      } else if (timer.forceRepeat) {
        status += ' (nonstop)';
      }
      
      return { success: true, output: status };
    } else {
      return { success: true, output: 'Timer is stopped.' };
    }
  }

  /**
   * Delete a timer (#CANCEL command)
   */
  deleteTimer(name: string): { success: boolean; error?: string } {
    const timer = this.timers.get(name);
    
    if (!timer) {
      const error = `Timer '${name}' not found`;
      this.onError?.(error);
      return { success: false, error };
    }
    
    // Stop if running
    this.stopTimerInternal(name);
    
    // Remove from map
    this.timers.delete(name);
    
    // Delete from profile
    this.deleteTimerFromProfile(name);
    
    return { success: true };
  }

  /**
   * Cancel a timer (legacy alias)
   */
  cancelTimer(name: string): { success: boolean; error?: string } {
    return this.deleteTimer(name);
  }

  /**
   * Stop timer internally
   */
  private stopTimerInternal(name: string): void {
    const timerId = this.timerIds.get(name);
    if (timerId !== undefined) {
      window.clearTimeout(timerId);
      this.timerIds.delete(name);
    }
    
    const timer = this.timers.get(name);
    if (timer) {
      timer.state = 'stopped';
      timer.startTime = undefined;
    }
  }

  /**
   * Start timer internally
   * PR01PH07T03: Updated to use executeThroughParser for #IF/#ELSE support
   */
  private startTimerInternal(timer: Timer): void {
    // Clear any existing timeout
    const existingId = this.timerIds.get(timer.name);
    if (existingId !== undefined) {
      window.clearTimeout(existingId);
    }
    
    timer.state = 'running';
    timer.startTime = Date.now();
    
    const executeAndSchedule = async () => {
      if (!this.timers.has(timer.name) || timer.state !== 'running') {
        return; // Timer was stopped or cancelled
      }
      
      timer.fireCount++;
      this.onTimerFire?.(timer);
      
      // Execute commands through the automation pipeline
      // PR01PH07T03: Use executeThroughParser if available for #IF/#ELSE support
      if (this.executionContext) {
        const substitutedCommands = timer.commands.map(cmd => 
          this.executionContext!.substituteVariables(cmd)
        );
        
        // Use parser execution path if available
        if (this.executionContext.executeThroughParser) {
          await this.executionContext.executeThroughParser(substitutedCommands);
        } else {
          // Fallback to direct execution
          this.executionContext.executeCommands(substitutedCommands);
        }
      }
      
      // Schedule next fire for repeating timers
      // PR01PH08: Check for override flags (single/nonstop)
      const shouldRepeat = timer.forceRepeat || (timer.repeat && !timer.singleShot);
      if (shouldRepeat && this.timers.has(timer.name) && timer.state === 'running') {
        const timerId = window.setTimeout(executeAndSchedule, timer.duration);
        this.timerIds.set(timer.name, timerId);
      } else {
        // One-time timer complete - stop it and clear override flags
        timer.state = 'stopped';
        timer.singleShot = undefined;
        timer.forceRepeat = undefined;
        this.timerIds.delete(timer.name);
      }
    };
    
    const timerId = window.setTimeout(executeAndSchedule, timer.duration);
    this.timerIds.set(timer.name, timerId);
  }

  /**
   * Persist timer to profile
   */
  private async persistTimer(timer: Timer): Promise<void> {
    if (this.onTimerSave) {
      const savedTimer: SavedTimer = {
        name: timer.name,
        duration: timer.duration,
        repeat: timer.repeat,
        commands: timer.commands
      };
      try {
        await this.onTimerSave(savedTimer);
      } catch (error) {
        console.error('[TimerManager] Failed to persist timer:', error);
      }
    }
  }

  /**
   * Delete timer from profile
   */
  private async deleteTimerFromProfile(name: string): Promise<void> {
    if (this.onTimerDelete) {
      try {
        await this.onTimerDelete(name);
      } catch (error) {
        console.error('[TimerManager] Failed to delete timer from profile:', error);
      }
    }
  }

  /**
   * Load timers from profile (all start in STOP state)
   */
  loadTimers(savedTimers: SavedTimer[]): void {
    for (const saved of savedTimers) {
      const timer: Timer = {
        id: `${saved.name}-${Date.now()}`,
        name: saved.name,
        duration: saved.duration,
        repeat: saved.repeat,
        commands: saved.commands,
        createdAt: Date.now(),
        fireCount: 0,
        state: 'stopped' // Always start in STOPPED state
      };
      this.timers.set(saved.name, timer);
    }
  }

  /**
   * Cancel all timers
   */
  cancelAllTimers(): void {
    for (const [name, timerId] of this.timerIds) {
      window.clearTimeout(timerId);
      this.onTimerCancel?.(name);
    }
    this.timerIds.clear();
    this.timers.clear();
  }

  /**
   * Stop all timers (but keep definitions)
   */
  stopAllTimers(): { success: boolean; output?: string } {
    const runningTimers: string[] = [];
    
    for (const name of this.timerIds.keys()) {
      this.stopTimerInternal(name);
      runningTimers.push(name);
    }
    
    if (runningTimers.length === 0) {
      return { success: true, output: 'No timers running.' };
    }
    
    return { success: true, output: `Stopped: ${runningTimers.join(', ')}` };
  }

  /**
   * Check if there are any active timers
   */
  hasActiveTimers(): boolean {
    return this.timers.size > 0;
  }
}

// ============================================
// Timer Command Handlers
// ============================================

/**
 * Handle #TIMER command execution
 * Creates or updates a timer definition
 */
export function handleTimerCommand(
  token: CommandToken,
  tokens: ParsedToken[],
  timerManager: TimerManager,
  currentIndex: number
): { success: boolean; error?: string } {
  // Parse timer arguments
  const args = token.args || '';
  const parsed = parseTimerArgs(args);
  
  if (!parsed) {
    const error = `Invalid #TIMER syntax. Expected: #TIMER name duration [REPEAT]`;
    timerManager.reportError(error);
    return { success: false, error };
  }
  
  // Find matching #ENDTIMER
  const endIndex = findTimerEnd(tokens, currentIndex);
  if (endIndex === null) {
    const error = `#TIMER without matching #ENDTIMER`;
    timerManager.reportError(error);
    return { success: false, error };
  }
  
  // Extract commands between #TIMER and #ENDTIMER
  const commands = extractTimerCommands(tokens, currentIndex, endIndex);
  
  // Create or update the timer (starts in STOPPED state)
  return timerManager.createOrUpdateTimer(parsed.name, parsed.duration, parsed.repeat, commands);
}

/**
 * Handle #START command - starts a stopped timer
 * Supports modifiers:
 * - #START timerName - use timer settings (repeat checkbox)
 * - #START timerName single - run once, ignore repeat setting
 * - #START timerName nonstop - keep repeating, ignore repeat setting
 * - #START all - start all timers using saved settings
 */
export function handleStartCommand(
  token: CommandToken,
  timerManager: TimerManager
): { success: boolean; error?: string; output?: string } {
  const args = token.args?.trim();
  
  if (!args) {
    const error = `#START requires a timer name`;
    timerManager.reportError(error);
    return { success: false, error };
  }
  
  // Handle #START all
  if (args.toLowerCase() === 'all') {
    return timerManager.startAllTimers();
  }
  
  // Parse the arguments - could be "timerName", "timerName single", or "timerName nonstop"
  const parts = args.split(/\s+/);
  const timerName = parts[0];
  const modifier = parts[1]?.toLowerCase();
  
  // Call appropriate method based on modifier
  if (modifier === 'single') {
    const result = timerManager.startTimerSingle(timerName);
    return { success: result.success, error: result.error };
  } else if (modifier === 'nonstop') {
    const result = timerManager.startTimerNonstop(timerName);
    return { success: result.success, error: result.error };
  } else if (modifier) {
    const error = `Unknown modifier '${modifier}' for #START. Use 'single' or 'nonstop'`;
    timerManager.reportError(error);
    return { success: false, error };
  }
  
  // Default: use timer settings
  const result = timerManager.startTimer(timerName);
  return { success: result.success, error: result.error };
}

/**
 * Handle #STOP command - stops a running timer
 * Supports:
 * - #STOP timerName - stop a specific timer
 * - #STOP all - stop all running timers
 */
export function handleStopCommand(
  token: CommandToken,
  timerManager: TimerManager
): { success: boolean; error?: string; output?: string } {
  const timerName = token.args?.trim();
  
  if (!timerName) {
    const error = `#STOP requires a timer name`;
    timerManager.reportError(error);
    return { success: false, error };
  }
  
  // Handle #STOP all
  if (timerName.toLowerCase() === 'all') {
    return timerManager.stopAllTimers();
  }
  
  const result = timerManager.stopTimer(timerName);
  return { success: result.success, error: result.error };
}

/**
 * Handle #CHECK command - outputs timer status
 * Supports:
 * - #CHECK timerName - check specific timer
 * - #CHECK all - list all timers
 */
export function handleCheckCommand(
  token: CommandToken,
  timerManager: TimerManager
): { success: boolean; error?: string; output?: string } {
  const timerName = token.args?.trim();
  
  if (!timerName) {
    const error = `#CHECK requires a timer name`;
    timerManager.reportError(error);
    return { success: false, error };
  }
  
  // Handle #CHECK all
  if (timerName.toLowerCase() === 'all') {
    return timerManager.listAllTimers();
  }
  
  return timerManager.checkTimer(timerName);
}

/**
 * Handle #CANCEL command execution
 */
export function handleCancelCommand(
  token: CommandToken,
  timerManager: TimerManager
): { success: boolean; error?: string } {
  const timerName = token.args?.trim();
  
  if (!timerName) {
    const error = `#CANCEL requires a timer name`;
    timerManager.reportError(error);
    return { success: false, error };
  }
  
  return timerManager.deleteTimer(timerName);
}
