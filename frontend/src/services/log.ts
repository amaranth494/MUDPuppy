// PR02PH09: Logging utility - standardized logging for browser console
// All console.log calls should use this to ensure consistent formatting

/**
 * Format a timestamp for log entries
 * Format: YYYY-MM-DD HH:MM:SS
 */
function formatTimestamp(): string {
  const now = new Date();
  return now.toISOString().replace('T', ' ').substring(0, 19);
}

/**
 * Log to browser console with standardized #LOG formatting
 * Use this instead of console.log for all F12 console output
 */
export function logToConsole(message: string, ...args: unknown[]): void {
  const timestamp = formatTimestamp();
  console.log(`[LOG] ${timestamp} ${message}`, ...args);
}

/**
 * Log error to browser console with standardized formatting
 */
export function logErrorToConsole(message: string, ...args: unknown[]): void {
  const timestamp = formatTimestamp();
  console.error(`[LOG ERROR] ${timestamp} ${message}`, ...args);
}

/**
 * Log warning to browser console with standardized formatting
 */
export function logWarnToConsole(message: string, ...args: unknown[]): void {
  const timestamp = formatTimestamp();
  console.warn(`[LOG WARN] ${timestamp} ${message}`, ...args);
}

/**
 * Log debug to browser console with standardized formatting
 */
export function logDebugToConsole(message: string, ...args: unknown[]): void {
  const timestamp = formatTimestamp();
  console.debug(`[LOG DEBUG] ${timestamp} ${message}`, ...args);
}
