// PR02PH09: Logging utility - standardized logging for browser console
// All console.log calls should use this to ensure consistent formatting

/**
 * Get caller's file, line, and function name from stack trace
 * Returns format: [filename:line] or [filename:line funcName]
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
    const functionName = caller.getFunctionName(); // Get the function name
    
    // Extract relative path from src/ directory
    let relativePath = fileName;
    const srcIndex = fileName.indexOf('src/');
    if (srcIndex !== -1) {
      relativePath = fileName.substring(srcIndex + 4); // +4 to skip 'src/'
    } else {
      // Fallback: just use filename if src/ not found
      relativePath = fileName.split('/').pop() || fileName.split('\\').pop() || fileName;
    }
    
    // Include function name if available
    if (functionName) {
      return `[${relativePath}:${lineNumber} ${functionName}]`;
    }
    return `[${relativePath}:${lineNumber}]`;
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
