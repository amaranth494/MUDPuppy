/**
 * Automation Logic Engine - PR01PH02
 * 
 * Handles condition parsing, evaluation, and #IF/#ELSE/#ENDIF execution:
 * - Parse ${variable} references
 * - Parse comparison operators: > < >= <= ==
 * - Parse logical operators: AND, OR
 * - Evaluate conditions against variable values
 * - Execute commands based on condition results
 * 
 * Variable Resolution:
 * - Profile variables: Persisted with user profile (#SET commands)
 * - System variables: Read-only, managed by system
 * - Session variables: Temporary, cleared on disconnect
 */

import { ParsedToken, CommandToken, parser, validateSyntax } from './parser';
import { handleTimerCommand, handleCancelCommand, TimerManager, findTimerEnd, handleStartCommand, handleStopCommand, handleCheckCommand } from './timer';
// PR02PH06: Import ICM adapter for command classification
import { recognizeCommand } from '../icm-adapter';

// ============================================
// ANSI Color Helper Functions - PR02PH09
// ============================================

/**
 * Get ANSI color code from color name or hex value
 * Supports: black, red, green, yellow, blue, magenta, cyan, white
 * Also supports hex values like FFFF00
 */
function getAnsiColorCode(color: string): string {
  const colorMap: Record<string, string> = {
    black: '\x1b[30m',
    red: '\x1b[31m',
    green: '\x1b[32m',
    yellow: '\x1b[33m',
    blue: '\x1b[34m',
    magenta: '\x1b[35m',
    cyan: '\x1b[36m',
    white: '\x1b[37m',
    // Bright variants
    brightblack: '\x1b[90m',
    brightred: '\x1b[91m',
    brightgreen: '\x1b[92m',
    brightyellow: '\x1b[93m',
    brightblue: '\x1b[94m',
    brightmagenta: '\x1b[95m',
    brightcyan: '\x1b[96m',
    brightwhite: '\x1b[97m',
  };
  
  // Check named colors
  if (colorMap[color]) {
    return colorMap[color];
  }
  
  // Handle hex color (e.g., FFFF00 -> RGB)
  if (/^[0-9A-Fa-f]{6}$/.test(color)) {
    const r = parseInt(color.substring(0, 2), 16);
    const g = parseInt(color.substring(2, 4), 16);
    const b = parseInt(color.substring(4, 6), 16);
    // Use 256-color mode (38;5;nnn)
    const ansi256 = 16 + Math.floor(r / 51) * 36 + Math.floor(g / 51) * 6 + Math.floor(b / 51);
    return `\x1b[38;5;${ansi256}m`;
  }
  
  // Default to white if unknown
  return '\x1b[37m';
}

/**
 * Get ANSI background color code
 */
function getAnsiBackgroundCode(color: string): string {
  const bgColorMap: Record<string, string> = {
    black: '\x1b[40m',
    red: '\x1b[41m',
    green: '\x1b[42m',
    yellow: '\x1b[43m',
    blue: '\x1b[44m',
    magenta: '\x1b[45m',
    cyan: '\x1b[46m',
    white: '\x1b[47m',
    brightblack: '\x1b[100m',
    brightred: '\x1b[101m',
    brightgreen: '\x1b[102m',
    brightyellow: '\x1b[103m',
    brightblue: '\x1b[104m',
    brightmagenta: '\x1b[105m',
    brightcyan: '\x1b[106m',
    brightwhite: '\x1b[107m',
  };
  
  if (bgColorMap[color]) {
    return bgColorMap[color];
  }
  
  // Handle hex background color
  if (/^[0-9A-Fa-f]{6}$/.test(color)) {
    const r = parseInt(color.substring(0, 2), 16);
    const g = parseInt(color.substring(2, 4), 16);
    const b = parseInt(color.substring(4, 6), 16);
    const ansi256 = 16 + Math.floor(r / 51) * 36 + Math.floor(g / 51) * 6 + Math.floor(b / 51);
    return `\x1b[48;5;${ansi256}m`;
  }
  
  return '\x1b[47m';
}

// ============================================
// Types
// ============================================

// Variable value types
export type VariableValue = string | number | boolean;

// Variable resolver interface - separates profile, system, and session variables
// 
// Precedence: session > profile > system
// - Profile variables (${name}): Persisted with profile via #SET
// - Session variables (%1, %2): Temporary, from alias arguments/trigger captures
// - System variables (%TIME, %CHARACTER): Read-only, managed by system
export interface VariableResolver {
  // Profile variables (persisted with profile)
  getProfile(name: string): VariableValue | undefined;
  setProfile(name: string, value: VariableValue): void;
  
  // System variables (read-only, managed by system)
  getSystem(name: string): VariableValue | undefined;
  
  // Session variables (temporary, session-only)
  getSession(name: string): VariableValue | undefined;
  setSession(name: string, value: VariableValue): void;
  
  // Convenience: get from any source (profile -> session -> system priority)
  get(name: string): VariableValue | undefined;
}

// Legacy alias for backward compatibility
export type VariableStore = VariableResolver;

// Execution result
export interface ExecutionResult {
  success: boolean;
  commands: string[];
  errors: ExecutionError[];
}

export interface ExecutionError {
  message: string;
  line?: number;
  column?: number;
}

// Execution context
export interface ExecutionContext {
  variables: VariableStore;
  depth: number;
  maxDepth: number;
  timerManager?: TimerManager;
  executeCommands?: (commands: string[]) => void;
  aliasResolver?: (aliasName: string) => Promise<string[]>;
  // PR01PH08: For outputting timer status to local terminal
  outputMessage?: (message: string) => void;
}

// AST Node types for conditions
export type ASTNode = 
  | VariableNode 
  | ValueNode 
  | ComparisonNode 
  | LogicalNode;

export interface VariableNode {
  type: 'variable';
  name: string;
}

export interface ValueNode {
  type: 'value';
  value: VariableValue;
}

export interface ComparisonNode {
  type: 'comparison';
  operator: '>' | '<' | '>=' | '<=' | '==' | '!=' | '=';
  left: ASTNode;
  right: ASTNode;
}

export interface LogicalNode {
  type: 'logical';
  operator: 'AND' | 'OR';
  left: ASTNode;
  right: ASTNode;
}

// ============================================
// Constants
// ============================================

const DEFAULT_MAX_DEPTH = 999; // High value - no practical limit

// Evaluation timeout in milliseconds (PR01PH05T03)
const EVALUATION_TIMEOUT_MS = 500;

// System variables - these are read-only and cannot be modified by #SET
export const SYSTEM_VARIABLES: Record<string, VariableValue> = {
  '%TIME': '',
  '%DATE': '',
  '%CHARACTER': '',
  '%PASSWORD': '',
  '%SERVER': '',
  '%SESSIONID': '',
  '%TICKCOUNT': 0,
  '%RANDOM': 0,
};

// Check if a variable name is a system variable
export function isSystemVariable(name: string): boolean {
  const upperName = name.toUpperCase();
  return upperName in SYSTEM_VARIABLES;
}

// Check if a variable is a system variable that should be set from connection
export function isConnectionSystemVariable(name: string): boolean {
  const upperName = name.toUpperCase();
  return upperName === '%CHARACTER' || upperName === '%PASSWORD' || upperName === '%SERVER';
}

// ============================================
// Condition Parser
// ============================================

/**
 * Parse a condition string into an AST
 * Examples:
 *   ${hp} < 30
 *   ${gold} >= 1000
 *   ${hp} < 50 AND ${mana} > 10
 *   ${target} == dragon
 */
export function parseCondition(condition: string): ASTNode | null {
  if (!condition || condition.trim() === '') {
    return null;
  }

  const trimmed = condition.trim();
  
  try {
    return parseLogicalExpression(trimmed);
  } catch (e) {
    console.error('[Evaluator] Failed to parse condition:', e);
    return null;
  }
}

/**
 * Parse a logical expression (AND/OR)
 */
function parseLogicalExpression(input: string): ASTNode {
  // Find the outermost AND/OR operator at the lowest precedence level
  // We need to find the rightmost AND/OR at the top level (not inside parentheses)
  
  let parenDepth = 0;
  let lastAndPos = -1;
  let lastOrPos = -1;
  
  for (let i = input.length - 1; i >= 0; i--) {
    const char = input[i];
    
    if (char === ')') {
      parenDepth++;
    } else if (char === '(') {
      parenDepth--;
    } else if (parenDepth === 0) {
      // Check for OR first (lower precedence than AND)
      if (i >= 1 && input.substring(i, i + 2).toUpperCase() === 'OR') {
        lastOrPos = i;
      }
      // Check for AND
      if (i >= 2 && input.substring(i - 1, i + 2).toUpperCase() === 'AND') {
        lastAndPos = i - 1;
      }
    }
  }
  
  // Handle OR at top level
  if (lastOrPos !== -1) {
    const left = input.substring(0, lastOrPos).trim();
    const right = input.substring(lastOrPos + 2).trim();
    return {
      type: 'logical',
      operator: 'OR',
      left: parseComparisonExpression(left),
      right: parseComparisonExpression(right)
    };
  }
  
  // Handle AND at top level
  if (lastAndPos !== -1) {
    const left = input.substring(0, lastAndPos).trim();
    const right = input.substring(lastAndPos + 3).trim();
    return {
      type: 'logical',
      operator: 'AND',
      left: parseComparisonExpression(left),
      right: parseComparisonExpression(right)
    };
  }
  
  // No logical operator, parse as comparison
  return parseComparisonExpression(input);
}

/**
 * Parse a comparison expression (> < >= <= ==)
 */
function parseComparisonExpression(input: string): ASTNode {
  const trimmed = input.trim();
  
  // Find comparison operator
  // Order matters: >=, <=, ==, != first, then >, <, =
  const operators = ['>=', '<=', '==', '!=', '>', '<', '='];
  
  for (const op of operators) {
    const pos = findOperatorAtTopLevel(trimmed, op);
    if (pos !== -1) {
      const left = trimmed.substring(0, pos).trim();
      const right = trimmed.substring(pos + op.length).trim();
      
      return {
        type: 'comparison',
        operator: op as ComparisonNode['operator'],
        left: parseValue(left),
        right: parseValue(right)
      };
    }
  }
  
  // No comparison operator, parse as value
  return parseValue(trimmed);
}

/**
 * Find operator position at top level (not inside quotes or nested)
 */
function findOperatorAtTopLevel(input: string, operator: string): number {
  let parenDepth = 0;
  let inQuote = false;
  let quoteChar = '';
  
  for (let i = 0; i < input.length; i++) {
    const char = input[i];
    
    // Handle quotes
    if ((char === '"' || char === "'") && !inQuote) {
      inQuote = true;
      quoteChar = char;
    } else if (char === quoteChar && inQuote) {
      inQuote = false;
      quoteChar = '';
      continue;
    }
    
    if (inQuote) continue;
    
    // Track parentheses
    if (char === '(') {
      parenDepth++;
    } else if (char === ')') {
      parenDepth--;
    }
    
    // Check for operator at top level
    if (parenDepth === 0 && input.substring(i, i + operator.length) === operator) {
      return i;
    }
  }
  
  return -1;
}

/**
 * Parse a value (variable or literal)
 */
function parseValue(input: string): ASTNode {
  const trimmed = input.trim();
  
  // Check for variable reference ${name}
  if (trimmed.startsWith('${') && trimmed.endsWith('}')) {
    const name = trimmed.slice(2, -1);
    return {
      type: 'variable',
      name
    };
  }
  
  // Check for quoted string
  if ((trimmed.startsWith('"') && trimmed.endsWith('"')) ||
      (trimmed.startsWith("'") && trimmed.endsWith("'"))) {
    return {
      type: 'value',
      value: trimmed.slice(1, -1)
    };
  }
  
  // Check for boolean
  if (trimmed.toLowerCase() === 'true') {
    return {
      type: 'value',
      value: true
    };
  }
  if (trimmed.toLowerCase() === 'false') {
    return {
      type: 'value',
      value: false
    };
  }
  
  // Check for number
  const num = parseFloat(trimmed);
  if (!isNaN(num)) {
    return {
      type: 'value',
      value: num
    };
  }
  
  // Default to string value
  return {
    type: 'value',
    value: trimmed
  };
}

// ============================================
// Evaluation Engine
// ============================================

/**
 * Smart type inference for #SET command:
 * - If string is numeric, convert to number
 * - Otherwise keep as string
 * This ensures variables are stored with the correct type for operations
 */
function inferType(value: VariableValue): VariableValue {
  if (typeof value === 'string') {
    // Try to convert to number
    const num = parseFloat(value);
    if (!isNaN(num) && isFinite(num)) {
      return num;
    }
    
    // Check for boolean strings
    if (value === 'true') return true;
    if (value === 'false') return false;
    
    return value;
  }
  
  return value;
}

/**
 * Evaluate an AST node against variables
 */
export function evaluate(node: ASTNode, variables: VariableStore): VariableValue {
  switch (node.type) {
    case 'variable': {
      const value = variables.get(node.name);
      return value !== undefined ? value : '';
    }
    
    case 'value':
      return node.value;
    
    case 'comparison': {
      const left = evaluate(node.left, variables);
      const right = evaluate(node.right, variables);
      const result = compareValues(left, right, node.operator);
      return result;
    }
    
    case 'logical': {
      const left = evaluate(node.left, variables);
      const leftTruthy = isTruthy(left);
      
      if (node.operator === 'AND') {
        if (!leftTruthy) return false;
        const right = evaluate(node.right, variables);
        return isTruthy(right);
      } else { // OR
        if (leftTruthy) return true;
        const right = evaluate(node.right, variables);
        return isTruthy(right);
      }
    }
    
    default:
      return false;
  }
}

/**
 * Compare two values with the given operator
 */
function compareValues(left: VariableValue, right: VariableValue, operator: string): boolean {
  // Coerce to numbers for comparison if both are numeric
  const leftNum = typeof left === 'number' ? left : (typeof left === 'string' ? parseFloat(left) : (left ? 1 : 0));
  const rightNum = typeof right === 'number' ? right : (typeof right === 'string' ? parseFloat(right) : (right ? 1 : 0));
  
  // If both can be treated as numbers, do numeric comparison
  const leftIsNumeric = typeof left === 'number' || (!isNaN(parseFloat(String(left))) && isFinite(Number(left)));
  const rightIsNumeric = typeof right === 'number' || (!isNaN(parseFloat(String(right))) && isFinite(Number(right)));
  
  if (leftIsNumeric && rightIsNumeric) {
    switch (operator) {
      case '>': return leftNum > rightNum;
      case '<': return leftNum < rightNum;
      case '>=': return leftNum >= rightNum;
      case '<=': return leftNum <= rightNum;
      case '==': return leftNum === rightNum;
      case '!=': return leftNum !== rightNum;
      case '=': return leftNum === rightNum;
    }
  }
  
  // String comparison
  const leftStr = String(left).toLowerCase();
  const rightStr = String(right).toLowerCase();
  
  switch (operator) {
    case '>': return leftStr > rightStr;
    case '<': return leftStr < rightStr;
    case '>=': return leftStr >= rightStr;
    case '<=': return leftStr <= rightStr;
    case '==': return leftStr === rightStr;
    case '!=': return leftStr !== rightStr;
    case '=': return leftStr === rightStr;
  }
  
  return false;
}

/**
 * Convert a value to boolean for truthiness
 */
function isTruthy(value: VariableValue): boolean {
  if (typeof value === 'boolean') return value;
  if (typeof value === 'number') return value !== 0;
  if (typeof value === 'string') return value.length > 0;
  return false;
}

// ============================================
// Execution Engine
// ============================================

/**
 * Default variable resolver implementation
 * Uses three separate maps for profile, system, and session variables
 * 
 * Variable Resolution Precedence (get() method):
 *   session > profile > system
 * 
 * Profile variables (${name}): Persisted with the user profile via #SET
 * Session variables (%1, %2): Temporary, populated from alias arguments or trigger pattern captures
 * System variables (%TIME, %CHARACTER): Read-only, managed by the system
 * 
 * #SET command:
 * - Backend write fails → value does not persist, automation continues, failure is logged
 */
export class SimpleVariableStore implements VariableResolver {
  private profileVariables: Map<string, VariableValue> = new Map();
  private systemVariables: Map<string, VariableValue> = new Map();
  private sessionVariables: Map<string, VariableValue> = new Map();
  
  // Callback for persisting profile variables to backend
  private onPersistProfile: ((variables: Record<string, VariableValue>) => Promise<void>) | null = null;
  
  // Callback for notifying UI of variable changes
  private onVariableChange: ((name: string, value: VariableValue) => void) | null = null;
  
  /**
   * Set the persistence callback for profile variables
   */
  setPersistCallback(callback: (variables: Record<string, VariableValue>) => Promise<void>): void {
    this.onPersistProfile = callback;
  }
  
  /**
   * Set the variable change callback for UI updates
   */
  setVariableChangeCallback(callback: (name: string, value: VariableValue) => void): void {
    this.onVariableChange = callback;
  }
  
  // Profile variables (persisted with profile)
  getProfile(name: string): VariableValue | undefined {
    return this.profileVariables.get(name);
  }
  
  async setProfile(name: string, value: VariableValue): Promise<void> {
    // Check for system variable protection (PR01PH03T03)
    if (isSystemVariable(name)) {
      throw new Error(`Cannot modify system variable: ${name}`);
    }
    
    this.profileVariables.set(name, value);
    
    // Notify UI of change
    if (this.onVariableChange) {
      this.onVariableChange(name, value);
    }
    
    // Persist to backend if callback is set
    if (this.onPersistProfile) {
      try {
        await this.onPersistProfile(this.toObject());
      } catch (error) {
        console.error('[SimpleVariableStore] Failed to persist variable:', error);
        // Don't throw - variable is still set in memory
      }
    }
  }
  
  // System variables (read-only)
  getSystem(name: string): VariableValue | undefined {
    return this.systemVariables.get(name);
  }
  
  // PR01PH08: Set system variable (used for connection-based system vars like %CHARACTER, %PASSWORD, %SERVER)
  setSystem(name: string, value: VariableValue): void {
    this.systemVariables.set(name, value);
  }
  
  // Session variables (temporary)
  getSession(name: string): VariableValue | undefined {
    // Handle %1, %2 syntax for session variables (PR01PH03T04)
    if (/^%\d+$/.test(name)) {
      return this.sessionVariables.get(name);
    }
    return this.sessionVariables.get(name);
  }
  
  setSession(name: string, value: VariableValue): void {
    // Handle %1, %2 syntax for session variables (PR01PH03T04)
    this.sessionVariables.set(name, value);
  }
  
  // Convenience: get from any source (session > profile > system priority)
  get(name: string): VariableValue | undefined {
    // Priority: session > profile > system
    // Check session first (includes %1, %2 syntax)
    if (this.sessionVariables.has(name)) {
      return this.sessionVariables.get(name);
    }
    
    // Check profile
    if (this.profileVariables.has(name)) {
      return this.profileVariables.get(name);
    }
    
    // Check system
    return this.systemVariables.get(name);
  }
  
  // Legacy setter - maps to setProfile for backward compatibility
  async set(name: string, value: VariableValue): Promise<void> {
    await this.setProfile(name, value);
  }
  
  // Get all variables as object (profile only for persistence)
  toObject(): Record<string, VariableValue> {
    const obj: Record<string, VariableValue> = {};
    this.profileVariables.forEach((value, key) => {
      obj[key] = value;
    });
    return obj;
  }
  
  // Load from object (profile only)
  fromObject(obj: Record<string, VariableValue>): void {
    this.profileVariables.clear();
    Object.entries(obj).forEach(([key, value]) => {
      this.profileVariables.set(key, value);
    });
  }
  
  // Clear session variables (called on disconnect) - PR01PH03T04
  clearSession(): void {
    this.sessionVariables.clear();
  }
  
  // Initialize/update system variables - PR01PH03T01
  updateSystemVariable(name: string, value: VariableValue): void {
    this.systemVariables.set(name, value);
  }
  
  // Set multiple system variables at once
  setSystemVariables(variables: Record<string, VariableValue>): void {
    Object.entries(variables).forEach(([key, value]) => {
      this.systemVariables.set(key, value);
    });
  }
}

/**
 * Execute parsed tokens with #IF/#ELSE/#ENDIF support
 */
export async function executeTokens(
  tokens: ParsedToken[],
  variables: VariableStore,
  maxDepth: number = DEFAULT_MAX_DEPTH,
  timerManager?: TimerManager,
  aliasResolver?: (aliasName: string) => Promise<string[]>,
  outputMessage?: (message: string) => void
): Promise<ExecutionResult> {
  const errors: ExecutionError[] = [];
  const commands: string[] = [];
  
  try {
    const context: ExecutionContext = {
      variables,
      depth: 0,
      maxDepth,
      timerManager,
      executeCommands: undefined,
      aliasResolver,
      outputMessage
    };

    const result = await executeTokenList(tokens, context, errors);
    commands.push(...result);
    
    return {
      success: errors.length === 0,
      commands,
      errors
    };
  } catch (e) {
    const error = e as Error;
    errors.push({
      message: `Execution error: ${error.message}`
    });
    
    return {
      success: false,
      commands,
      errors
    };
  }
}

/**
 * Execute a list of tokens, handling #IF/#ELSE/#ENDIF blocks
 */
async function executeTokenList(
  tokens: ParsedToken[],
  context: ExecutionContext,
  errors: ExecutionError[]
): Promise<string[]> {
  const commands: string[] = [];
  let i = 0;
  
  while (i < tokens.length) {
    const token = tokens[i];
    
    if (token.type === 'COMMAND') {
      switch (token.command) {
        case 'IF':
          // Get condition from args
          const conditionStr = token.args || '';
          const ast = parseCondition(conditionStr);
          
          if (!ast) {
            errors.push({
              message: `Failed to parse condition: ${conditionStr}`,
              line: token.line,
              column: token.column
            });
            i++;
            continue;
          }
          
          // Evaluate condition
          const result = evaluate(ast, context.variables);
          const conditionTrue = isTruthy(result);
          
          // Find matching #ELSE/#ENDIF
          const blockEnd = findBlockEnd(tokens, i);
          
          if (blockEnd === -1) {
            errors.push({
              message: 'Missing #ENDIF',
              line: token.line,
              column: token.column
            });
            i++;
            continue;
          }
          
          // Execute appropriate branch
          
          if (conditionTrue) {
            // Execute IF branch (tokens between IF and ELSE/ENDIF)
            const ifEnd = findElseOrEndif(tokens, i);
            const ifTokens = tokens.slice(i + 1, ifEnd);
            commands.push(...await executeTokenList(ifTokens, context, errors));
          } else {
            // Check for ELSE branch
            const elsePos = findElseOrEndif(tokens, i);
            const nextToken = tokens[elsePos];
            
            if (nextToken && nextToken.type === 'COMMAND' && (nextToken as CommandToken).command === 'ELSE') {
              // Execute ELSE branch (tokens between ELSE and ENDIF)
              const elseTokens = tokens.slice(elsePos + 1, blockEnd);
              commands.push(...await executeTokenList(elseTokens, context, errors));
            }
          }
          
          i = blockEnd + 1;
          continue;
          
        case 'SET':
          // Handle #SET variable value (async for persistence)
          await handleSetCommand(token, context.variables, errors);
          i++;
          continue;
        
        case 'ADD':
          // Handle #ADD variable value - add to numeric variable
          await handleAddCommand(token, context.variables, errors);
          i++;
          continue;
        
        case 'SUB':
          // Handle #SUB variable value - subtract from numeric variable
          await handleSubCommand(token, context.variables, errors);
          i++;
          continue;
          
        case 'TIMER':
          // Handle #TIMER command - create a timer
          if (context.timerManager) {
            const result = handleTimerCommand(token, tokens, context.timerManager, i);
            if (!result.success && result.error) {
              errors.push({
                message: result.error,
                line: token.line,
                column: token.column
              });
            }
            // Skip to #ENDTIMER
            const endIndex = findTimerEnd(tokens, i);
            if (endIndex !== null) {
              i = endIndex + 1;
              continue;
            }
          }
          i++;
          continue;
          
        case 'CANCEL':
          // Handle #CANCEL command
          if (context.timerManager) {
            const result = handleCancelCommand(token, context.timerManager);
            if (!result.success && result.error) {
              errors.push({
                message: result.error,
                line: token.line,
                column: token.column
              });
            }
          }
          i++;
          continue;
          
        case 'START':
          // Handle #START command - start a stopped timer
          // PR01PH08: Output to local terminal for 'all' variant
          if (context.timerManager) {
            const result = handleStartCommand(token, context.timerManager);
            if (!result.success && result.error) {
              errors.push({
                message: result.error,
                line: token.line,
                column: token.column
              });
            } else if (result.output) {
              // Remove brackets and add bright cyan color for system messages
              const brightCyan = '\x1b[96m';
              const reset = '\x1b[0m';
              const formatted = result.output.includes(',') 
                ? result.output.split(',').join('\r\n')
                : result.output;
              context.outputMessage?.(`\r\n${brightCyan}${formatted}${reset}\r\n`);
            }
          }
          i++;
          continue;
        
        case 'STOP':
          // Handle #STOP command - stop a running timer
          // PR01PH08: Output to local terminal for 'all' variant
          if (context.timerManager) {
            const result = handleStopCommand(token, context.timerManager);
            if (!result.success && result.error) {
              errors.push({
                message: result.error,
                line: token.line,
                column: token.column
              });
            } else if (result.output) {
              // Remove brackets and add bright cyan color for system messages
              const brightCyan = '\x1b[96m';
              const reset = '\x1b[0m';
              const formatted = result.output.includes(',') 
                ? result.output.split(',').join('\r\n')
                : result.output;
              context.outputMessage?.(`\r\n${brightCyan}${formatted}${reset}\r\n`);
            }
          }
          i++;
          continue;
         
        case 'CHECK':
          // Handle #CHECK command - output timer status
          // PR01PH08: Output to local terminal, NOT as MUD command
          if (context.timerManager) {
            const result = handleCheckCommand(token, context.timerManager);
            if (!result.success && result.error) {
              errors.push({
                message: result.error,
                line: token.line,
                column: token.column
              });
            } else if (result.output) {
              // Output directly to terminal via outputMessage callback - DO NOT add to commands!
              // This prevents the message from being sent to the MUD
              // Remove brackets and add bright cyan color for system messages
              const brightCyan = '\x1b[96m';
              const reset = '\x1b[0m';
              const formatted = result.output.includes(',') 
                ? result.output.split(',').join('\r\n')
                : result.output;
              context.outputMessage?.(`\r\n${brightCyan}${formatted}${reset}\r\n`);
            }
          }
          i++;
          continue;
          
        case 'ECHO':
          // Handle #ECHO command - output message to terminal
          // PR02PH09: Echo outputs directly to local terminal
          // PR02PH09: Extended to support color options: #ECHO (color:red,bold) message

          if (token.args) {
            // Parse optional color specification: (color:xxx,yyy)
            const colorMatch = token.args.match(/^\s*\(([^)]+)\)\s*(.*)$/s);
            
            let text = token.args;
            let prefix = '';
            let suffix = '';
            
            if (colorMatch) {
              const options = colorMatch[1].toLowerCase();
              text = colorMatch[2];
              
              // Parse color options (comma-separated)
              const optionParts = options.split(',').map(o => o.trim());
              
              for (const opt of optionParts) {
                if (opt.startsWith('color:')) {
                  const colorValue = opt.substring(6);
                  prefix += getAnsiColorCode(colorValue);
                } else if (opt === 'bold') {
                  prefix += '\x1b[1m';
                } else if (opt === 'underline') {
                  prefix += '\x1b[4m';
                } else if (opt.startsWith('background:')) {
                  const bgColor = opt.substring(11);
                  prefix += getAnsiBackgroundCode(bgColor);
                }
              }
              suffix = '\x1b[0m';
            } else {
              // Default: bright green
              prefix = '\x1b[92m';
              suffix = '\x1b[0m';
            }
            
            context.outputMessage?.(`\r\n${prefix}${text}${suffix}\r\n`);
          }
          i++;
          continue;
          
        case 'LOG':
           // Handle #LOG command - output to log
           // PR02PH09: Log outputs to terminal with different formatting
           // PR02PH09: Extended to support output targets: #LOG (to:console) message
           if (token.args) {
             // Parse optional target specification: (to:xxx)
             const targetMatch = token.args.match(/^\s*\(([^)]+)\)\s*(.*)$/s);
             
             let text = token.args;
             let target = 'terminal'; // default target // default target
             
             if (targetMatch) {
               const options = targetMatch[1].toLowerCase();
               text = targetMatch[2];
               
               // Parse target options
               if (options.startsWith('to:')) {
                 target = options.substring(3);
               }
             }
             
             // Helper to get timestamp
             const now = new Date();
             const timestamp = now.toISOString().replace('T', ' ').substring(0, 19);
             
             if (target === 'console') {
                // Send to web console (F12)
                console.log(`[LOG] ${timestamp} ${text}`);
              } else if (target === 'local') {
                // Send to terminal only
                const brightYellow = '\x1b[93m';
                const reset = '\x1b[0m';
                context.outputMessage?.(`\r\n${brightYellow}[LOG] ${timestamp} ${text}${reset}\r\n`);
              } else if (target.includes('console') && target.includes('local')) {
                // Multi-target: send to both console and terminal
                console.log(`[LOG] ${timestamp} ${text}`);
                const brightYellow = '\x1b[93m';
                const reset = '\x1b[0m';
                context.outputMessage?.(`\r\n${brightYellow}[LOG] ${timestamp} ${text}${reset}\r\n`);
              } else {
                // Default: output to terminal with bright yellow formatting
                const brightYellow = '\x1b[93m';
                const reset = '\x1b[0m';
                context.outputMessage?.(`\r\n${brightYellow}[LOG] ${timestamp} ${text}${reset}\r\n`);
              }
           }
           i++;
           continue;
          
        case 'HELP':
          // Handle #HELP command - show help
          // PR02PH09: Show help for available commands
          {
            const helpText = 'Available commands: #IF/#ELSE/#ENDIF, #SET, #ADD, #SUB, #TIMER, #START/#STOP/#CHECK/#CANCEL, #ECHO, #LOG, #HELP';
            const brightCyan = '\x1b[96m';
            const reset = '\x1b[0m';
            context.outputMessage?.(`\r\n${brightCyan}${helpText}${reset}\r\n`);
          }
          i++;
          continue;
          
        case 'ELSE':
        case 'ENDIF':
        case 'ENDTIMER':
          // These are handled elsewhere or are not direct execution commands
          // For now, skip them in inline execution
          i++;
          continue;
          
        default:
          i++;
          continue;
      }
    }
    
    // Handle TEXT and VARIABLE tokens - collect commands
    if (token.type === 'TEXT' || token.type === 'VARIABLE') {
      const text = token.value;
      if (text.trim()) {
        // Check for @alias invocation - resolve if aliasResolver is provided
        const trimmedText = text.trim();
        // PR02PH06: Use ICM classification instead of redundant prefix check
        const textClassification = recognizeCommand(trimmedText);
        if (textClassification.operator === '@' && context.aliasResolver) {
          const aliasName = trimmedText.substring(1).trim();
          if (aliasName) {
            try {
              const aliasCommands = await context.aliasResolver(aliasName);
              commands.push(...aliasCommands);
            } catch (error) {
              errors.push({
                message: `Failed to resolve alias @${aliasName}: ${error}`,
                line: token.line,
                column: token.column
              });
            }
          }
        } else {
          // PR02PH06: Use ICM classification - send to MUD only if not internal command
          const cmdClassification = recognizeCommand(trimmedText);
          if (!cmdClassification.isInternal) {
            // Only send to MUD if it's not an internal command
            // Lines starting with # are automation commands that should be processed, not sent
            commands.push(trimmedText);
          }
        }
      }
      i++;
      continue;
    }
    
    i++;
  }
  
  return commands;
}

/**
 * Find the end of an #IF block (#ENDIF that closes this IF)
 */
function findBlockEnd(tokens: ParsedToken[], startIndex: number): number {
  // Start from the next token after startIndex (skip the starting #IF)
  let depth = 0;
  
  for (let i = startIndex + 1; i < tokens.length; i++) {
    const token = tokens[i];
    
    if (token.type === 'COMMAND') {
      if (token.command === 'IF') {
        // Nested #IF - increase depth
        depth++;
      } else if (token.command === 'ENDIF') {
        if (depth === 0) {
          // This ENDIF matches our IF
          return i;
        }
        depth--;
      }
    }
  }
  
  return -1;
}

/**
 * Find the next #ELSE or #ENDIF at the current nesting level
 */
function findElseOrEndif(tokens: ParsedToken[], startIndex: number): number {
  // Start from the next token after startIndex (skip the starting #IF)
  // Depth starts at 0 because we're looking for the ELSE/ENDIF that matches
  // the IF at position startIndex
  let depth = 0;
  
  for (let i = startIndex + 1; i < tokens.length; i++) {
    const token = tokens[i];
    
    if (token.type === 'COMMAND') {
      if (token.command === 'IF') {
        // Nested #IF - increase depth
        depth++;
      } else if (token.command === 'ENDIF') {
        if (depth === 0) {
          // This ENDIF matches our starting IF
          return i;
        }
        depth--;
      } else if (token.command === 'ELSE' && depth === 0) {
        // This ELSE matches our starting IF
        return i;
      }
    }
  }
  
  return -1;
}

/**
 * Handle #SET command
 */
async function handleSetCommand(
  token: CommandToken,
  variables: VariableStore,
  errors: ExecutionError[]
): Promise<void> {
  const args = token.args || '';
  const match = args.match(/^(\S+)\s+(.*)$/);
  
  if (!match) {
    errors.push({
      message: '#SET requires variable name and value',
      line: token.line,
      column: token.column
    });
    return;
  }
  
  const [, varName, varValue] = match;
  
  // Parse the value to determine type
  const parsedValue = parseValue(varValue.trim());
  let value = evaluate(parsedValue, variables);
  
  // Smart type inference: determine the best type for storage
  value = inferType(value);
  
  // Check for system variable protection (PR01PH03T03)
  if (isSystemVariable(varName)) {
    errors.push({
      message: `Cannot modify system variable: ${varName}`,
      line: token.line,
      column: token.column
    });
    return;
  }
  
  // Handle session variables (%1, %2 syntax) - PR01PH03T04
  if (/^%\d+$/.test(varName)) {
    variables.setSession(varName, value);
    return;
  }
  
  // Profile variable - set and persist
  try {
    await variables.setProfile(varName, value);
  } catch (error) {
    const err = error as Error;
    errors.push({
      message: err.message || 'Failed to set variable',
      line: token.line,
      column: token.column
    });
  }
}

/**
 * Handle #ADD command - add to a numeric variable
 * #ADD varname value
 */
async function handleAddCommand(
  token: CommandToken,
  variables: VariableStore,
  errors: ExecutionError[]
): Promise<void> {
  const args = token.args || '';
  const match = args.match(/^(\S+)\s+(.*)$/);
  
  if (!match) {
    errors.push({
      message: '#ADD requires variable name and numeric value',
      line: token.line,
      column: token.column
    });
    return;
  }
  
  const [, varName, varValue] = match;
  
  // Check for system variable protection
  if (isSystemVariable(varName)) {
    errors.push({
      message: `Cannot modify system variable: ${varName}`,
      line: token.line,
      column: token.column
    });
    return;
  }
  
  // Get current value
  let currentValue: number | undefined = variables.get(varName) as number | undefined;
  
  // If variable doesn't exist or isn't a number, treat as 0
  if (currentValue === undefined || currentValue === null) {
    currentValue = 0;
  } else if (typeof currentValue === 'string') {
    // Try to convert string to number
    const num = parseFloat(currentValue);
    if (isNaN(num)) {
      currentValue = 0;
    } else {
      currentValue = num;
    }
  } else if (typeof currentValue === 'boolean') {
    currentValue = currentValue ? 1 : 0;
  }
  
  // Ensure current value is numeric
  if (typeof currentValue !== 'number') {
    currentValue = 0;
  }
  
  // Parse the value to add
  const parsedValue = parseValue(varValue.trim());
  let addValue: number | undefined = evaluate(parsedValue, variables) as number | undefined;
  
  // Ensure add value is numeric - try to convert if needed
  if (typeof addValue === 'string') {
    const num = parseFloat(addValue);
    if (isNaN(num)) {
      // Lenient: treat non-numeric strings as 0
      addValue = 0;
    } else {
      addValue = num;
    }
  } else if (typeof addValue === 'boolean') {
    addValue = addValue ? 1 : 0;
  } else if (typeof addValue === 'undefined') {
    addValue = 0;
  }
  
  if (addValue === undefined) {
    addValue = 0;
  }
  
  const newValue = currentValue + addValue;
  
  // Handle session variables (%1, %2 syntax)
  if (/^%\d+$/.test(varName)) {
    variables.setSession(varName, newValue);
    return;
  }
  
  // Profile variable - set and persist
  try {
    await variables.setProfile(varName, newValue);
  } catch (error) {
    const err = error as Error;
    errors.push({
      message: err.message || 'Failed to add to variable',
      line: token.line,
      column: token.column
    });
  }
}

/**
 * Handle #SUB command - subtract from a numeric variable
 * #SUB varname value
 */
async function handleSubCommand(
  token: CommandToken,
  variables: VariableStore,
  errors: ExecutionError[]
): Promise<void> {
  const args = token.args || '';
  const match = args.match(/^(\S+)\s+(.*)$/);
  
  if (!match) {
    errors.push({
      message: '#SUB requires variable name and numeric value',
      line: token.line,
      column: token.column
    });
    return;
  }
  
  const [, varName, varValue] = match;
  
  // Check for system variable protection
  if (isSystemVariable(varName)) {
    errors.push({
      message: `Cannot modify system variable: ${varName}`,
      line: token.line,
      column: token.column
    });
    return;
  }
  
  // Get current value
  let currentValue: number | undefined = variables.get(varName) as number | undefined;
  
  // If variable doesn't exist or isn't a number, treat as 0
  if (currentValue === undefined || currentValue === null) {
    currentValue = 0;
  } else if (typeof currentValue === 'string') {
    // Try to convert string to number
    const num = parseFloat(currentValue);
    if (isNaN(num)) {
      currentValue = 0;
    } else {
      currentValue = num;
    }
  } else if (typeof currentValue === 'boolean') {
    currentValue = currentValue ? 1 : 0;
  }
  
  // Ensure current value is numeric
  if (typeof currentValue !== 'number') {
    currentValue = 0;
  }
  
  // Parse the value to subtract
  const parsedValue = parseValue(varValue.trim());
  let subValue: number | undefined = evaluate(parsedValue, variables) as number | undefined;
  
  // Ensure sub value is numeric - try to convert if needed
  if (typeof subValue === 'string') {
    const num = parseFloat(subValue);
    if (isNaN(num)) {
      // Lenient: treat non-numeric strings as 0
      subValue = 0;
    } else {
      subValue = num;
    }
  } else if (typeof subValue === 'boolean') {
    subValue = subValue ? 1 : 0;
  } else if (typeof subValue === 'undefined') {
    subValue = 0;
  }
  
  if (subValue === undefined) {
    subValue = 0;
  }
  
  const newValue = currentValue - subValue;
  
  // Handle session variables (%1, %2 syntax)
  if (/^%\d+$/.test(varName)) {
    variables.setSession(varName, newValue);
    return;
  }
  
  // Profile variable - set and persist
  try {
    await variables.setProfile(varName, newValue);
  } catch (error) {
    const err = error as Error;
    errors.push({
      message: err.message || 'Failed to subtract from variable',
      line: token.line,
      column: token.column
    });
  }
}

/**
 * Substitute variables in a string
 */
export function substituteVariables(input: string, variables: VariableStore): string {
  const variablePattern = /\$\{([^}]+)\}/g;
  
  return input.replace(variablePattern, (match, varName) => {
    const value = variables.get(varName);
    if (value !== undefined) {
      return String(value);
    }
    return match;
  });
}

// ============================================
// Convenience Functions
// ============================================

/**
 * Parse and execute an automation action string
 * 
 * This is a convenience function that parses and executes
 * automation action text with #IF/#ELSE/#ENDIF support.
 * Includes 500ms timeout for condition evaluation to prevent runaway scripts (PR01PH05T03).
 * Supports @alias invocation if aliasResolver callback is provided.
 * 
 * @param actionText - The action text to execute
 * @param variables - Variable store for variable substitution
 * @param timerManager - Optional timer manager for timer commands
 * @param aliasResolver - Optional callback to resolve @alias invocations
 * @param outputMessage - Optional callback for local output (e.g., timer status)
 */
export async function executeAutomationAction(
  actionText: string,
  variables: VariableStore,
  timerManager?: TimerManager,
  aliasResolver?: (aliasName: string) => Promise<string[]>,
  outputMessage?: (message: string) => void
): Promise<ExecutionResult> {
  // Parse the action text
  const parseResult = parser.parse(actionText);
  
  if (!parseResult.success) {
    return {
      success: false,
      commands: [],
      errors: parseResult.errors.map((e) => ({
        message: e.message,
        line: e.line,
        column: e.column
      }))
    };
  }
  
  // Validate syntax
  const validationErrors = validateSyntax(parseResult.tokens);
  if (validationErrors.length > 0) {
    return {
      success: false,
      commands: [],
      errors: validationErrors.map((e) => ({
        message: e.message,
        line: e.line,
        column: e.column
      }))
    };
  }
  
  // Execute with timeout protection (PR01PH05T03)
  return await executeWithTimeout(
    parseResult.tokens, 
    variables, 
    DEFAULT_MAX_DEPTH, 
    timerManager,
    aliasResolver,
    outputMessage
  );
}

/**
 * Execute tokens with timeout protection
 * Wraps condition evaluation in a 500ms timeout to prevent runaway scripts
 */
async function executeWithTimeout(
  tokens: ParsedToken[],
  variables: VariableStore,
  maxDepth: number,
  timerManager?: TimerManager,
  aliasResolver?: (aliasName: string) => Promise<string[]>,
  outputMessage?: (message: string) => void
): Promise<ExecutionResult> {
  let timeoutId: ReturnType<typeof setTimeout> | null = null;
  let isTimedOut = false;
  
  // Create a promise that rejects after timeout
  const timeoutPromise = new Promise<ExecutionResult>((_, reject) => {
    timeoutId = setTimeout(() => {
      isTimedOut = true;
      console.error('[Evaluator] Evaluation timeout exceeded (500ms)');
      reject(new Error('Evaluation timeout exceeded (500ms)'));
    }, EVALUATION_TIMEOUT_MS);
  });
  
  try {
    // Execute the tokens
    const executionPromise = executeTokens(tokens, variables, maxDepth, timerManager, aliasResolver, outputMessage);
    
    // Race between execution and timeout
    const result = await Promise.race([executionPromise, timeoutPromise]);
    
    // Clear timeout if execution completed first
    if (timeoutId) {
      clearTimeout(timeoutId);
    }
    
    // Check if we timed out during execution
    if (isTimedOut) {
      return {
        success: false,
        commands: [],
        errors: [{
          message: `Evaluation timeout exceeded (${EVALUATION_TIMEOUT_MS}ms) - possible infinite loop detected`
        }]
      };
    }
    
    return result;
  } catch (error) {
    // Clear timeout on error
    if (timeoutId) {
      clearTimeout(timeoutId);
    }
    
    // If it's the timeout error, return error result
    if (isTimedOut || (error instanceof Error && error.message.includes('timeout'))) {
      return {
        success: false,
        commands: [],
        errors: [{
          message: `Evaluation timeout exceeded (${EVALUATION_TIMEOUT_MS}ms) - possible infinite loop detected`
        }]
      };
    }
    
    // Re-throw other errors
    throw error;
  }
}
