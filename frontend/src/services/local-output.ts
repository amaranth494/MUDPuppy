/**
 * Local Output Service - PR02PH09
 * 
 * Unified interface for all local text output to the terminal.
 * This ensures consistent styling for all local information
 * (ICM errors, validation messages, command echoes, timer status).
 * 
 * All local output goes through this service rather than raw terminal writes.
 */

import type { Terminal } from '@xterm/xterm';

/**
 * Local output message types with default colors
 */
export type LocalOutputType = 
  | 'info'      // General info - cyan
  | 'success'   // Success messages - green  
  | 'warning'   // Warnings - yellow
  | 'error'     // Errors - red
  | 'command'   // Command echo - bright cyan
  | 'debug'     // Debug info - magenta
  | 'system';   // System messages - white

/**
 * Options for local output
 */
export interface LocalOutputOptions {
  /** Message type (determines color) */
  type?: LocalOutputType;
  /** Whether to include timestamp */
  timestamp?: boolean;
  /** Whether to add newline before message */
  newlineBefore?: boolean;
  /** Whether to add newline after message */
  newlineAfter?: boolean;
}

/**
 * ANSI color codes for local output types
 */
const COLORS: Record<LocalOutputType, string> = {
  info: '\x1b[36m',      // cyan
  success: '\x1b[32m',   // green
  warning: '\x1b[33m',   // yellow
  error: '\x1b[31m',     // red
  command: '\x1b[96m',   // bright cyan
  debug: '\x1b[35m',     // magenta
  system: '\x1b[37m',    // white
};

const RESET = '\x1b[0m';

/**
 * Write local output to terminal with consistent styling
 * 
 * @param terminal - The terminal instance to write to
 * @param message - The message to display
 * @param options - Output options
 */
export function writeLocal(
  terminal: Terminal | undefined,
  message: string,
  options: LocalOutputOptions = {}
): void {
  if (!terminal) return;
  
  const {
    type = 'info',
    timestamp = false,
    newlineBefore = true,
    newlineAfter = true,
  } = options;
  
  let output = '';
  
  // Add newline before if requested
  if (newlineBefore) {
    output += '\r\n';
  }
  
  // Add timestamp if requested
  if (timestamp) {
    const now = new Date();
    const timeStr = now.toLocaleTimeString('en-US', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
    output += `\x1b[90m[${timeStr}]${RESET} `;
  }
  
  // Add colored message
  const color = COLORS[type];
  output += `${color}${message}${RESET}`;
  
  // Add newline after if requested
  if (newlineAfter) {
    output += '\r\n';
  }
  
  terminal.write(output);
}

/**
 * Write an error message to terminal
 */
export function writeError(
  terminal: Terminal | undefined,
  message: string,
  options: Partial<LocalOutputOptions> = {}
): void {
  writeLocal(terminal, message, { ...options, type: 'error' });
}

/**
 * Write a warning message to terminal
 */
export function writeWarning(
  terminal: Terminal | undefined,
  message: string,
  options: Partial<LocalOutputOptions> = {}
): void {
  writeLocal(terminal, message, { ...options, type: 'warning' });
}

/**
 * Write a success message to terminal
 */
export function writeSuccess(
  terminal: Terminal | undefined,
  message: string,
  options: Partial<LocalOutputOptions> = {}
): void {
  writeLocal(terminal, message, { ...options, type: 'success' });
}

/**
 * Write a debug message to terminal
 */
export function writeDebug(
  terminal: Terminal | undefined,
  message: string,
  options: Partial<LocalOutputOptions> = {}
): void {
  writeLocal(terminal, message, { ...options, type: 'debug' });
}

/**
 * Write a command echo to terminal
 */
export function writeCommand(
  terminal: Terminal | undefined,
  command: string,
  options: Partial<LocalOutputOptions> = {}
): void {
  writeLocal(terminal, command, { ...options, type: 'command' });
}

/**
 * Write a system message to terminal
 */
export function writeSystem(
  terminal: Terminal | undefined,
  message: string,
  options: Partial<LocalOutputOptions> = {}
): void {
  writeLocal(terminal, message, { ...options, type: 'system' });
}

/**
 * Write an ICM error message to terminal
 */
export function writeICMError(
  terminal: Terminal | undefined,
  message: string,
  options: Partial<LocalOutputOptions> = {}
): void {
  writeLocal(terminal, `[ICM] ${message}`, { ...options, type: 'error' });
}

/**
 * Write an ICM info message to terminal
 */
export function writeICMInfo(
  terminal: Terminal | undefined,
  message: string,
  options: Partial<LocalOutputOptions> = {}
): void {
  writeLocal(terminal, `[ICM] ${message}`, { ...options, type: 'info' });
}
