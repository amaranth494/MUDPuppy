/**
 * Automation Engine - SP05 Automation Foundations
 * 
 * Handles aliases, triggers, and variable substitution for MUD command automation.
 * Includes loop detection, command queuing, and circuit breaker protections.
 */

import { Trigger, AutomationAliases, AutomationTriggers, AutomationVariables } from '../types';
import { SimpleVariableStore, VariableValue, executeAutomationAction } from './automation/evaluator';
import { TimerManager, SavedTimer } from './automation/timer';

// ============================================
// Types
// ============================================

// Command source context
export type CommandSource = 'user' | 'alias' | 'trigger';

// Command with metadata
export interface ProcessedCommand {
  command: string;
  source: CommandSource;
}

// Alias expansion result
export interface AliasExpansionResult {
  commands: string[];
  depth: number;
}

// Trigger fire result
export interface TriggerFireResult {
  trigger: Trigger;
  action: string;
  cooldownRemaining: number;
}

// Loop detection entry
interface LoopDetectionEntry {
  command: string;
  timestamp: number;
  cycleId: number;
}

// Circuit breaker state
export interface CircuitBreakerState {
  isTripped: boolean;
  tripReason: string | null;
  trippedAt: number | null;
}

// Automation configuration
export interface AutomationConfig {
  aliases: AutomationAliases;
  triggers: AutomationTriggers;
  variables: AutomationVariables;
  connectionId: string;
}

// ============================================
// Constants
// ============================================

// Max recursion depth for alias chains (SP05PH03T05)
const MAX_ALIAS_RECURSION_DEPTH = 3;

// Max commands per dispatch (SP05PH03T05)
const MAX_COMMANDS_PER_DISPATCH = 200;

// Max queue size for memory safety (SP05PH03T14)
const MAX_QUEUE_SIZE = 100;

// Max command history for loop detection (SP05PH03T10)
const MAX_COMMAND_HISTORY = 50;

// Loop detection threshold - same command repeated this many times triggers protection
const LOOP_DETECTION_THRESHOLD = 50;

// Loop detection time window in ms
const LOOP_DETECTION_WINDOW_MS = 2000;

// Max triggers per second (SP05 spec)
const MAX_TRIGGERS_PER_SECOND = 10;

// ============================================
// Automation Engine Class
// ============================================

export class AutomationEngine {
  // Automation data
  private aliases: AutomationAliases = { items: [] };
  private triggers: AutomationTriggers = { items: [] };
  private connectionId: string | null = null;
  
  // Unified variable store (PR01PH03T01)
  private variableStore: SimpleVariableStore;
  
  // Connection state
  private isConnected: boolean = false;
  
  // Trigger cooldown tracking
  private triggerLastFired: Map<string, number> = new Map();
  
  // Last trigger command for self-activation protection (SP05PH03T12)
  private lastTriggerCommand: string | null = null;
  private lastTriggerCycleId: number = 0;
  private currentCycleId: number = 0;
  
  // Command queue with backpressure (SP05PH03T11)
  private commandQueue: ProcessedCommand[] = [];
  private isProcessingQueue: boolean = false;
  private lastDispatchTime: number = 0;
  
  // Loop detection (SP05PH03T10)
  private commandHistory: LoopDetectionEntry[] = [];
  
  // Circuit breaker (SP05PH03T13)
  private circuitBreaker: CircuitBreakerState = {
    isTripped: false,
    tripReason: null,
    trippedAt: null,
  };
  
  // Rate limiting for triggers
  private triggerCountThisSecond: number = 0;
  private triggerCountResetTime: number = 0;
  
  // Callback for submitting commands to MUD
  private onSubmitCommand: ((command: string) => void) | null = null;
  
  // PR01PH08: Callback for outputting local messages to terminal
  private terminalCallback: ((message: string) => void) | null = null;
  
  // Callback for circuit breaker notifications
  private onCircuitBreakerTripped: ((reason: string) => void) | null = null;
  
  // Timer manager (PR01PH04)
  private timerManager: TimerManager;

  constructor() {
    this.variableStore = new SimpleVariableStore();
    this.timerManager = new TimerManager({
      maxTimers: 10,
      onTimerFire: (timer) => {
        console.log(`[Timer] '${timer.name}' fired (count: ${timer.fireCount})`);
      },
      onTimerCancel: (name) => {
        console.log(`[Timer] '${name}' cancelled`);
      },
      onError: (error) => {
        console.error(`[Timer] Error: ${error}`);
      }
    });
    
    // Set up the timer execution context (PR01PH04)
    // PR01PH07T03: Updated to also process through parser for #IF/#ELSE support
    this.timerManager.setExecutionContext({
      executeCommands: (commands: string[]) => {
        // Queue commands for execution
        for (const cmd of commands) {
          this.queueCommand({
            command: cmd,
            source: 'trigger', // Timers act like triggers
          });
        }
      },
      substituteVariables: (input: string) => {
        return this.substituteVariables(input);
      },
      // PR01PH07T03: New method to execute commands through parser
      executeThroughParser: async (commands: string[]): Promise<void> => {
        for (const cmd of commands) {
          const trimmedCmd = cmd.trim();
          if (trimmedCmd.startsWith('#')) {
            // Process # commands through parser
            try {
              const result = await executeAutomationAction(trimmedCmd, this.variableStore, this.timerManager, undefined, this.terminalCallback ?? undefined);
              if (!result.success) {
                console.warn('[Automation] Timer # command errors:', result.errors);
              }
              // Queue the resulting commands
              for (const execCmd of result.commands) {
                this.queueCommand({
                  command: execCmd,
                  source: 'trigger',
                });
              }
            } catch (error) {
              console.error('[Automation] Timer # command execution error:', error);
            }
          } else {
            // Regular command - queue directly
            this.queueCommand({
              command: trimmedCmd,
              source: 'trigger',
            });
          }
        }
      }
    });
  }

  /**
   * Initialize the automation engine with configuration
   */
  configure(config: AutomationConfig): void {
    this.aliases = config.aliases;
    this.triggers = config.triggers;
    this.connectionId = config.connectionId;
    
    // Load variables into the unified variable store (PR01PH03T01)
    if (config.variables && config.variables.items) {
      for (const variable of config.variables.items) {
        this.variableStore.setProfile(variable.name, variable.value);
      }
    }
  }
  
  /**
   * Set the callback for persisting variables to backend
   */
  setPersistVariablesCallback(callback: (variables: Record<string, VariableValue>) => Promise<void>): void {
    this.variableStore.setPersistCallback(callback);
  }
  
  /**
   * Set the callback for variable change notifications
   */
  setVariableChangeCallback(callback: (name: string, value: VariableValue) => void): void {
    this.variableStore.setVariableChangeCallback(callback);
  }
  
  /**
   * Get the variable store for external access
   */
  getVariableStore(): SimpleVariableStore {
    return this.variableStore;
  }

  /**
   * Get the timer manager for external access (PR01PH04)
   */
  getTimerManager(): TimerManager {
    return this.timerManager;
  }

  /**
   * Set the command submission callback
   */
  setSubmitCommandCallback(callback: (command: string) => void): void {
    this.onSubmitCommand = callback;
  }

  /**
   * PR01PH08: Set the terminal output callback for local messages
   */
  setTerminalCallback(callback: (message: string) => void): void {
    this.terminalCallback = callback;
  }

  /**
   * Set the callbacks for persisting timers to backend
   * PR01PH04: Timers need to be saved to profile storage
   */
  setTimerCallbacks(
    onTimerSave: (timer: SavedTimer) => Promise<void>,
    onTimerDelete: (timerName: string) => Promise<void>
  ): void {
    this.timerManager = new TimerManager({
      maxTimers: 10,
      onTimerFire: (timer) => {
        console.log(`[Timer] '${timer.name}' fired (count: ${timer.fireCount})`);
      },
      onTimerCancel: (name) => {
        console.log(`[Timer] '${name}' cancelled`);
      },
      onTimerSave,
      onTimerDelete,
      onError: (error) => {
        console.error(`[Timer] Error: ${error}`);
      }
    });
    
    // Set up the timer execution context
    this.timerManager.setExecutionContext({
      executeCommands: (commands: string[]) => {
        for (const cmd of commands) {
          this.queueCommand({
            command: cmd,
            source: 'trigger',
          });
        }
      },
      substituteVariables: (input: string) => {
        return this.substituteVariables(input);
      },
      // PR01PH08: Output timer status messages locally, not to MUD
      outputMessage: (message: string) => {
        // Write directly to terminal - don't send to MUD
        if (this.terminalCallback) {
          this.terminalCallback(message + '\r\n');
        }
      },
      // PR01PH07T03: Execute through parser for #IF/#ELSE support
      executeThroughParser: async (commands: string[]) => {
        const actionText = commands.join('\n');
        const aliasResolver = async (aliasName: string): Promise<string[]> => {
          return await this.invokeExplicitAlias(aliasName);
        };
        const result = await executeAutomationAction(actionText, this.variableStore, this.timerManager, aliasResolver, this.terminalCallback ?? undefined);
        for (const cmd of result.commands) {
          this.queueCommand({
            command: cmd,
            source: 'trigger',
          });
        }
      }
    });
  }

  /**
   * Load timers from profile storage
   * Called after setTimerCallbacks when connecting
   */
  loadTimers(savedTimers: SavedTimer[]): void {
    if (this.timerManager) {
      this.timerManager.loadTimers(savedTimers);
    }
  }

  /**
   * Set the circuit breaker notification callback
   */
  setCircuitBreakerCallback(callback: (reason: string) => void): void {
    this.onCircuitBreakerTripped = callback;
  }

  /**
   * Mark the engine as connected
   */
  connect(): void {
    this.isConnected = true;
    this.resetCycleId();
    // Clear session variables on new connect (PR01PH03T04)
    this.variableStore.clearSession();
  }

  /**
   * Disconnect and clear state (SP05PH03T06)
   */
  disconnect(): void {
    this.isConnected = false;
    this.clearTriggers();
    this.clearCommandQueue();
    this.resetCircuitBreaker();
    // Clear session variables (PR01PH03T04)
    this.variableStore.clearSession();
    // Clear all timers (PR01PH04)
    // Stop all running timers but keep definitions for reconnect
    this.timerManager.stopAllTimers();
  }

  /**
   * Clear all trigger timers and state
   */
  private clearTriggers(): void {
    this.triggerLastFired.clear();
    this.lastTriggerCommand = null;
    this.lastTriggerCycleId = 0;
  }

  /**
   * Clear the command queue
   */
  private clearCommandQueue(): void {
    this.commandQueue = [];
    this.isProcessingQueue = false;
  }

  /**
   * Reset the circuit breaker
   */
  private resetCircuitBreaker(): void {
    this.circuitBreaker = {
      isTripped: false,
      tripReason: null,
      trippedAt: null,
    };
  }

  /**
   * Reset the cycle ID for trigger self-activation protection
   */
  resetCycleId(): void {
    this.currentCycleId = Date.now();
    this.lastTriggerCycleId = 0;
  }

  // ============================================
  // Command Processing Pipeline
  // ============================================

  /**
   * Process user input through the automation pipeline
   * Called from submitCommand with source='user'
   * 
   * Integration notes (PR01PH07):
   * - # commands are processed via executeAutomationAction (parser integrated)
   * - Aliases are expanded via evaluateAlias - parser integration needed
   */
  async processUserInput(input: string): Promise<ProcessedCommand[]> {
    // Check circuit breaker
    if (this.circuitBreaker.isTripped) {
      console.warn('[Automation] Circuit breaker tripped, ignoring input');
      return [];
    }

    // Step 1: Command separation (SP05PH03T09)
    const commands = this.parseCommandString(input);
    
    if (commands.length === 0) {
      return [];
    }

    // Check for too many commands
    if (commands.length > MAX_COMMANDS_PER_DISPATCH) {
      this.tripCircuitBreaker('Too many commands in single input');
      return [];
    }

    // Step 2-4: Process each command through the pipeline
    const processedCommands: ProcessedCommand[] = [];
    
    for (const cmd of commands) {
      // Check if this is a # command (IF, SET, TIMER, CANCEL)
      if (cmd.trim().startsWith('#')) {
        // Process # commands using the evaluator (PR01PH02, PR01PH04)
        try {
          const result = await executeAutomationAction(cmd, this.variableStore, this.timerManager, undefined, this.terminalCallback ?? undefined);
          if (!result.success) {
            console.warn('[Automation] # command errors:', result.errors);
          }
          // Commands are executed via timer callback or added here
          for (const execCmd of result.commands) {
            processedCommands.push({
              command: execCmd,
              source: 'user',
            });
          }
        } catch (error) {
          console.error('[Automation] # command execution error:', error);
        }
        continue;
      }
      
      // Variable substitution (SP05PH03T03) - for user input
      let processed = this.substituteVariables(cmd);
      
      // Alias evaluation (SP05PH03T02) - only for user source
      // Returns array to handle nested semicolons in replacement
      // PR01PH07T02: Pass through parser for #IF/#ELSE support
      const expansions = this.evaluateAlias(processed, 0);
      
      // Check for alias expansion loop
      if (expansions.depth > MAX_ALIAS_RECURSION_DEPTH) {
        console.warn('[Automation] Alias expansion depth exceeded');
        continue;
      }

      // Process each expanded command
      for (const expandedCmd of expansions.commands) {
        // PR01PH07T02: Check if expanded command contains # commands - parse through automation action
        const trimmedCmd = expandedCmd.trim();
        if (trimmedCmd.startsWith('#')) {
          // Process # commands using the evaluator (PR01PH02, PR01PH04)
          try {
            const result = await executeAutomationAction(trimmedCmd, this.variableStore, this.timerManager, undefined, this.terminalCallback ?? undefined);
            if (!result.success) {
              console.warn('[Automation] Alias # command errors:', result.errors);
            }
            // Add the resulting commands
            for (const execCmd of result.commands) {
              processedCommands.push({
                command: execCmd,
                source: 'alias',
              });
            }
          } catch (error) {
            console.error('[Automation] Alias # command execution error:', error);
          }
          continue;
        }
        
        // Record in command history for loop detection
        this.recordCommand(expandedCmd);

        processedCommands.push({
          command: expandedCmd,
          source: 'alias',
        });
      }
    }

    // Check for automation loop
    if (this.detectAutomationLoop()) {
      this.tripCircuitBreaker('Automation loop detected');
      return [];
    }

    // Queue all processed commands for sending to MUD
    for (const cmd of processedCommands) {
      this.queueCommand(cmd);
    }

    return processedCommands;
  }

  /**
   * Process a trigger action
   * Trigger actions bypass alias engine but support:
   * - Command separation (semicolons)
   * - Variable substitution
   * - Explicit @alias invocation
   * - PR01PH07T01: Parser integration via executeAutomationAction in fireTrigger
   * - PR01PH07T02: Alias replacement also goes through parser
   * Note: Trigger actions do NOT implicitly pass through alias evaluation
   */
  async processTriggerAction(actionText: string): Promise<string[]> {
    // SP05: Trigger actions support command separation like user input
    const commands = this.parseCommandString(actionText);
    
    const processedCommands: string[] = [];
    
    for (const cmd of commands) {
      // Variable substitution
      let processed = this.substituteVariables(cmd);
      
      // Check for explicit alias invocation (@alias)
      // Note: Trigger actions do NOT implicitly pass through alias evaluation
      // PR01PH07T02: But explicit alias invocations do go through parser
      if (processed.startsWith('@')) {
        const aliasCommands = await this.invokeExplicitAlias(processed.substring(1));
        for (const aliasCmd of aliasCommands) {
          processedCommands.push(aliasCmd);
        }
        continue;
      }
      
      processedCommands.push(processed);
    }
    
    return processedCommands;
  }

  /**
   * Process server output through trigger engine (SP05PH03T04)
   */
  processServerOutput(outputLines: string[]): void {
    if (!this.isConnected || this.circuitBreaker.isTripped) {
      return;
    }

    // Rate limiting (SP05 spec - max 10 triggers per second)
    this.enforceTriggerRateLimit();

    // Check trigger cooldown and process each line
    for (const line of outputLines) {
      // Skip if this output was generated by our own trigger (SP05PH03T12)
      if (this.shouldSkipTriggerOutput(line)) {
        continue;
      }

      const matchingTrigger = this.findMatchingTrigger(line);
      if (matchingTrigger) {
        console.log('[Trigger] ' + matchingTrigger.match);
        this.fireTrigger(matchingTrigger);
      }
    }
  }

  // ============================================
  // Command Separation (SP05PH03T09)
  // ============================================

  /**
   * Parse command string by semicolon separator
   * "look;inv;draw sword" -> ["look", "inv", "draw sword"]
   */
  private parseCommandString(input: string): string[] {
    // Split by semicolon
    const parts = input.split(';');
    
    // Trim whitespace and filter empty commands
    const commands: string[] = [];
    for (const part of parts) {
      const trimmed = part.trim();
      if (trimmed.length > 0) {
        commands.push(trimmed);
      }
    }

    return commands;
  }

  // ============================================
  // Variable Substitution (SP05PH03T03)
  // ============================================

  /**
   * Substitute ${variable_name} with variable values
   * Uses unified variable store (PR01PH03T01)
   */
  private substituteVariables(input: string): string {
    // Match ${variable_name} pattern
    const variablePattern = /\$\{([^}]+)\}/g;
    
    return input.replace(variablePattern, (match, varName) => {
      const value = this.variableStore.get(varName);
      if (value !== undefined) {
        return String(value);
      }
      // Return original if variable not found (preserves ${name} for clarity)
      return match;
    });
  }

  // ============================================
  // Alias Processing (SP05PH03T02)
  // ============================================

  /**
   * Evaluate alias for a command
   * Returns the expanded command and whether expansion occurred
   */
  private evaluateAlias(input: string, depth: number): AliasExpansionResult {
    // Get enabled aliases
    const enabledAliases = this.aliases.items.filter(a => a.enabled);
    
    for (const alias of enabledAliases) {
      // Use exact match - pattern must match the entire input exactly
      let matchResult: { matched: boolean; args: string[] } | null = null;
      
      if (input === alias.pattern) {
        // Exact match - no arguments captured
        matchResult = { matched: true, args: [] };
      } else if (input.startsWith(alias.pattern + ' ')) {
        // Pattern matches followed by space (captures arguments)
        const remaining = input.substring(alias.pattern.length + 1).trim();
        // Split remaining into words for %1, %2, %3... placeholders
        const args = remaining ? remaining.split(/\s+/) : [];
        matchResult = { matched: true, args: args };
      }
      
      if (matchResult && matchResult.matched) {
        // Build replacement with %1, %2, %3... substitution
        let replacement = alias.replacement;
        
        // Replace %1, %2, %3... with captured arguments
        // %1 = first arg, %2 = second arg, etc.
        // If argument doesn't exist, replace with empty string
        replacement = replacement.replace(/%(\d+)/g, (_match, numStr) => {
          const index = parseInt(numStr, 10);
          return matchResult.args[index - 1] || '';
        });

        // Substitute ${variable} patterns in the replacement
        replacement = this.substituteVariables(replacement);

        // PR01PH08: Normalize newlines to semicolons for consistent command processing
        // This fixes issues with multi-line alias replacements causing MUD echo problems
        replacement = replacement.replace(/\n/g, ';');

        // Handle nested semicolons in replacement - split and evaluate each
        // This implements depth-first expansion where each part of the replacement
        // is evaluated for more aliases before returning
        const replacementCommands = this.parseCommandString(replacement);
        const allExpandedCommands: string[] = [];
        let maxDepth = depth + 1;
        
        for (const repCmd of replacementCommands) {
          // Substitute variables in each command part
          const varSubbedCmd = this.substituteVariables(repCmd);
          
          // Recursively evaluate each part (depth-first)
          const nestedResult = this.evaluateAlias(varSubbedCmd, depth + 1);
          allExpandedCommands.push(...nestedResult.commands);
          maxDepth = Math.max(maxDepth, nestedResult.depth);
        }
        
        return {
          commands: allExpandedCommands,
          depth: maxDepth,
        };
      }
    }

    // No matching alias found - return input as single command
    return {
      commands: [input],
      depth: depth,
    };
  }

  /**
   * Explicit alias invocation from trigger action (@alias)
   * PR01PH07T02: Also processes # commands in alias replacement through parser
   */
  private async invokeExplicitAlias(text: string): Promise<string[]> {
    // Parse alias name and args
    const spaceIndex = text.indexOf(' ');
    let aliasName: string;
    let args: string;
    
    if (spaceIndex > 0) {
      aliasName = text.substring(0, spaceIndex);
      args = text.substring(spaceIndex + 1);
    } else {
      aliasName = text;
      args = '';
    }

    // Find the alias
    const alias = this.aliases.items.find(a => 
      a.enabled && a.pattern === aliasName
    );
    
    if (!alias) {
      return [];
    }

    // Build replacement with %1, %2, %3... substitution
    let replacement = alias.replacement;
    const argsArray = args ? args.split(/\s+/) : [];
    
    // Replace %1, %2, %3... with captured arguments
    replacement = replacement.replace(/%(\d+)/g, (_match, numStr) => {
      const index = parseInt(numStr, 10);
      return argsArray[index - 1] || '';
    });

    // Substitute ${variable} patterns in the replacement
    replacement = this.substituteVariables(replacement);
    
    // PR01PH08: Normalize newlines to semicolons for consistent command processing
    // This fixes issues with multi-line alias replacements causing MUD echo problems
    replacement = replacement.replace(/\n/g, ';');
    
    // PR01PH07T02: Check if replacement contains # commands - process through parser
    const trimmed = replacement.trim();
    if (trimmed.startsWith('#')) {
      try {
        const result = await executeAutomationAction(trimmed, this.variableStore, this.timerManager, undefined, this.terminalCallback ?? undefined);
        if (!result.success) {
          console.warn('[Automation] Explicit alias # command errors:', result.errors);
        }
        // Return all resulting commands
        return result.commands;
      } catch (error) {
        console.error('[Automation] Explicit alias # command execution error:', error);
        return [];
      }
    }
    
    // Handle semicolons in replacement - split into individual commands
    return this.parseCommandString(replacement);
  }

  // ============================================
  // Trigger Processing (SP05PH03T04)
  // ============================================

  /**
   * Find a trigger that matches the output line
   */
  private findMatchingTrigger(line: string): Trigger | null {
    const now = Date.now();
    
    for (const trigger of this.triggers.items) {
      if (!trigger.enabled) continue;
      
      // Check cooldown
      const lastFired = this.triggerLastFired.get(trigger.id) || 0;
      if (now - lastFired < trigger.cooldown_ms) {
        continue;
      }

      // Check match type
      if (trigger.type === 'contains') {
        if (line.includes(trigger.match)) {
          return trigger;
        }
      }
    }
    
    return null;
  }

  /**
   * Fire a trigger - queue its action for execution
   */
  private fireTrigger(trigger: Trigger): void {
    const now = Date.now();
    
    // Update last fired time
    this.triggerLastFired.set(trigger.id, now);

    // Set last trigger command for self-activation protection
    this.lastTriggerCommand = trigger.action;
    this.lastTriggerCycleId = this.currentCycleId;

    // Create alias resolver callback to resolve @alias invocations
    const aliasResolver = async (aliasName: string): Promise<string[]> => {
      return await this.invokeExplicitAlias(aliasName);
    };

    // Always use executeAutomationAction for all trigger actions to ensure timeout protection (PR01PH05T03)
    executeAutomationAction(trigger.action, this.variableStore, this.timerManager, aliasResolver, this.terminalCallback ?? undefined)
      .then(result => {
        if (!result.success) {
          console.warn('[Automation] Trigger action errors:', result.errors);
        }
        // Commands are executed via timer callback or added here
        for (const cmd of result.commands) {
          console.log('[Trigger] Action: ' + cmd);
          this.queueCommand({
            command: cmd,
            source: 'trigger',
          });
        }
      })
      .catch(error => {
        console.error('[Automation] Trigger action execution error:', error);
      });
  }

  /**
   * Check if this output line should be skipped for trigger matching
   * (SP05PH03T12 - trigger self-activation protection)
   */
  private shouldSkipTriggerOutput(line: string): boolean {
    // If we have a last trigger command and the output contains it
    // AND we're still in the same cycle, skip this output
    if (this.lastTriggerCommand && 
        this.lastTriggerCycleId === this.currentCycleId &&
        line.includes(this.lastTriggerCommand)) {
      // Clear the last trigger command after one cycle
      this.lastTriggerCommand = null;
      this.lastTriggerCycleId = 0;
      return true;
    }
    
    // Clear if we're in a new cycle
    if (this.lastTriggerCycleId !== this.currentCycleId) {
      this.lastTriggerCommand = null;
      this.lastTriggerCycleId = 0;
    }
    
    return false;
  }

  /**
   * Enforce trigger rate limiting
   */
  private enforceTriggerRateLimit(): void {
    const now = Date.now();
    
    // Reset counter every second
    if (now > this.triggerCountResetTime) {
      this.triggerCountThisSecond = 0;
      this.triggerCountResetTime = now + 1000;
    }
    
    // Check if we've hit the limit
    if (this.triggerCountThisSecond >= MAX_TRIGGERS_PER_SECOND) {
      console.warn('[Automation] Trigger rate limit reached');
      return;
    }
    
    this.triggerCountThisSecond++;
  }

  // ============================================
  // Command Queue with Backpressure (SP05PH03T11)
  // ============================================

  /**
   * Queue a command for processing
   * Returns true if queued, false if queue is full
   */
  private queueCommand(cmd: ProcessedCommand): boolean {
    // Check queue size for memory safety
    if (this.commandQueue.length >= MAX_QUEUE_SIZE) {
      console.warn('[Automation] Command queue full');
      return false;
    }
    
    this.commandQueue.push(cmd);
    
    // Start processing if not already
    if (!this.isProcessingQueue) {
      this.processCommandQueue();
    }
    
    return true;
  }

  /**
   * Process the command queue with backpressure
   */
  private async processCommandQueue(): Promise<void> {
    if (this.isProcessingQueue || this.commandQueue.length === 0) {
      return;
    }
    
    this.isProcessingQueue = true;
    
    while (this.commandQueue.length > 0) {
      // Check circuit breaker before processing
      if (this.circuitBreaker.isTripped) {
        break;
      }
      
      const cmd = this.commandQueue.shift();
      if (!cmd) continue;
      
      // Submit the command
      if (this.onSubmitCommand) {
        this.onSubmitCommand(cmd.command);
      }
      
      // Track dispatch time for backpressure
      this.lastDispatchTime = Date.now();
      
      // Small delay to prevent flooding (backpressure)
      // This allows the server to respond and controls command rate
      await this.sleep(50);
    }
    
    this.isProcessingQueue = false;
  }

  /**
   * Sleep utility
   */
  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  // ============================================
  // Loop Detection (SP05PH03T10)
  // ============================================

  /**
   * Record a command in history for loop detection
   */
  private recordCommand(command: string): void {
    const entry: LoopDetectionEntry = {
      command: command,
      timestamp: Date.now(),
      cycleId: this.currentCycleId,
    };
    
    this.commandHistory.push(entry);
    
    // Trim history to max size for memory safety
    if (this.commandHistory.length > MAX_COMMAND_HISTORY) {
      this.commandHistory = this.commandHistory.slice(-MAX_COMMAND_HISTORY);
    }
  }

  /**
   * Detect automation loop - rapid repetition of same command
   */
  private detectAutomationLoop(): boolean {
    const now = Date.now();
    const windowStart = now - LOOP_DETECTION_WINDOW_MS;
    
    // Filter to recent commands in current cycle
    const recentCommands = this.commandHistory.filter(
      e => e.timestamp > windowStart && e.cycleId === this.currentCycleId
    );
    
    if (recentCommands.length < LOOP_DETECTION_THRESHOLD) {
      return false;
    }
    
    // Count occurrences of each command
    const commandCounts = new Map<string, number>();
    for (const entry of recentCommands) {
      const count = commandCounts.get(entry.command) || 0;
      commandCounts.set(entry.command, count + 1);
    }
    
    // Check if any command repeats too many times
    for (const [_, count] of commandCounts) {
      if (count >= LOOP_DETECTION_THRESHOLD) {
        return true;
      }
    }
    
    return false;
  }

  // ============================================
  // Circuit Breaker (SP05PH03T13)
  // ============================================

  /**
   * Trip the circuit breaker
   */
  private tripCircuitBreaker(reason: string): void {
    this.circuitBreaker = {
      isTripped: true,
      tripReason: reason,
      trippedAt: Date.now(),
    };
    
    // Clear the command queue
    this.clearCommandQueue();
    
    // Notify via callback
    if (this.onCircuitBreakerTripped) {
      this.onCircuitBreakerTripped(reason);
    }
    
    console.warn('[Automation] Circuit breaker tripped:', reason);
  }

  /**
   * Get current circuit breaker state
   */
  getCircuitBreakerState(): CircuitBreakerState {
    return { ...this.circuitBreaker };
  }

  /**
   * Reset the circuit breaker (user手动 Resume)
   */
  resume(): void {
    this.resetCircuitBreaker();
    this.commandHistory = [];
  }

  // ============================================
  // Utility Methods
  // ============================================

  /**
   * Get the connection ID
   */
  getConnectionId(): string | null {
    return this.connectionId;
  }

  /**
   * Get last dispatch time (for debugging backpressure)
   */
  getLastDispatchTime(): number {
    return this.lastDispatchTime;
  }

  /**
   * Get the current queue size
   */
  getQueueSize(): number {
    return this.commandQueue.length;
  }

  /**
   * Check if engine is connected
   */
  isEngineConnected(): boolean {
    return this.isConnected;
  }
}

// ============================================
// Singleton Instance
// ============================================

let automationEngineInstance: AutomationEngine | null = null;

/**
 * Get the automation engine singleton
 */
export function getAutomationEngine(): AutomationEngine {
  if (!automationEngineInstance) {
    automationEngineInstance = new AutomationEngine();
  }
  return automationEngineInstance;
}

/**
 * Reset the automation engine (for testing or reconnection)
 */
export function resetAutomationEngine(): void {
  if (automationEngineInstance) {
    automationEngineInstance.disconnect();
  }
  automationEngineInstance = new AutomationEngine();
}
