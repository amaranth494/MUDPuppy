// PR02PH09: Logging utility - standardized logging for browser console
// All console.log calls should use this to ensure consistent formatting

/**
 * Get caller's file and line from stack trace
 * Returns format: [filename:line]
 */
function getCallerLocation(): string {
  const originalPrepareStackTrace = Error.prepareStackTrace;
  Error.prepareStackTrace = (_, stack) => stack;
  
  const err = new Error();
  const stack = err.stack as unknown as NodeJS.CallSite[];
  
  Error.prepareStackTrace = originalPrepareStackTrace;
  
  // Skip: getCallerLocation, logToConsole/logErrorToConsole, and the actual caller
  // stack[0] = getCallerLocation
  // stack[1] = logToConsole/logErrorToConsole
  // stack[2] = actual caller
  if (stack && stack.length > 2) {
    const caller = stack[2];
    const fileName = caller.getFileName() || 'unknown';
    const lineNumber = caller.getLineNumber() || 0;
    
    // Extract just the filename from full path
    const shortName = fileName.split('/').pop() || fileName.split('\\').pop() || fileName;
    
    return `[${shortName}:${lineNumber}]`;
  }
  
  return '';
}

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
 * Format: [LOG] YYYY-MM-DD HH:MM:SS [file.ts:line] message
 */
export function logToConsole(message: string, ...args: unknown[]): void {
  const timestamp = formatTimestamp();
  const location = getCallerLocation();
  console.log(`[LOG] ${timestamp} ${location} ${message}`, ...args);
}

/**
 * Log error to browser console with standardized formatting
 * Format: [LOG ERROR] YYYY-MM-DD HH:MM:SS [file.ts:line] message
 */
export function logErrorToConsole(message: string, ...args: unknown[]): void {
  const timestamp = formatTimestamp();
  const location = getCallerLocation();
  console.error(`[LOG ERROR] ${timestamp} ${location} ${message}`, ...args);
}

/**
 * Log warning to browser console with standardized formatting
 * Format: [LOG WARN] YYYY-MM-DD HH:MM:SS [file.ts:line] message
 */
export function logWarnToConsole(message: string, ...args: unknown[]): void {
  const timestamp = formatTimestamp();
  const location = getCallerLocation();
  console.warn(`[LOG WARN] ${timestamp} ${location} ${message}`, ...args);
}

/**
 * Log debug to browser console with standardized formatting
 * Format: [LOG DEBUG] YYYY-MM-DD HH:MM:SS [file.ts:line] message
 */
export function logDebugToConsole(message: string, ...args: unknown[]): void {
  const timestamp = formatTimestamp();
  const location = getCallerLocation();
  console.debug(`[LOG DEBUG] ${timestamp} ${location} ${message}`, ...args);
}
