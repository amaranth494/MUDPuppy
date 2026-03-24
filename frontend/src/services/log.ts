// PR02PH09: Logging utility - standardized logging for browser console
// All console.log calls should use this to ensure consistent formatting

/**
 * Parse function name from raw stack trace string
 * Chrome format: "at functionName (file:line:col)" or "at file:line:col"
 */
function parseFunctionNameFromStack(stackString: string, callerIndex: number): string | null {
  const lines = stackString.split('\n');
  // Skip first line (Error), find the caller line
  const callerLine = lines[callerIndex + 1]; // +1 because lines[0] is "Error"
  
  if (!callerLine) return null;
  
  // Match "at functionName (file:line:col)" or "at functionName@file:line:col"
  const match = callerLine.match(/at\s+([^\s(]+)/);
  if (match && match[1]) {
    const name = match[1];
    // Filter out internal/framework names
    if (name === 'Object' || name === 'Function' || name === '<anonymous>') {
      return null;
    }
    return name;
  }
  return null;
}

/**
 * Get caller's file, line, and function name from stack trace
 * Returns format: [filename:line] or [filename:line funcName]
 */
function getCallerLocation(): string {
  const err = new Error();
  const stackString = err.stack || '';
  
  // First try CallSite API
  const originalPrepareStackTrace = Error.prepareStackTrace;
  Error.prepareStackTrace = (_, stack) => stack;
  
  const stackTrace = err.stack as unknown as NodeJS.CallSite[];
  
  Error.prepareStackTrace = originalPrepareStackTrace;
  
  // Skip: getCallerLocation, logToConsole/logErrorToConsole, and the actual caller
  // stack[0] = getCallerLocation
  // stack[1] = logToConsole/logErrorToConsole
  // stack[2] = actual caller
  if (stackTrace && stackTrace.length > 2) {
    const caller = stackTrace[2];
    const fileName = caller.getFileName() || 'unknown';
    const lineNumber = caller.getLineNumber() || 0;
    let functionName = caller.getFunctionName();
    
    // Fallback: Try to get function name from method name
    if (!functionName && caller.getMethodName) {
      functionName = caller.getMethodName();
    }
    
    // Final fallback: Parse from raw stack string
    if (!functionName) {
      functionName = parseFunctionNameFromStack(stackString, 2);
    }
    
    // Extract relative path from src/ directory
    let relativePath = fileName;
    const srcIndex = fileName.indexOf('src/');
    if (srcIndex !== -1) {
      relativePath = fileName.substring(srcIndex + 4);
    } else {
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
