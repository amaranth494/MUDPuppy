// PR02PH09: Logging utility - standardized logging for browser console
// All console.log calls should use this to ensure consistent formatting

/**
 * Parse caller info from raw stack trace string
 * Chrome format: "at functionName (file:line:col)" or "at file:line:col"
 */
function parseCallerFromStack(stackString: string): { fileName: string; lineNumber: number; functionName: string | null } {
  const lines = stackString.split('\n').filter(line => line.trim());
  
  // Find the first "at" line that's NOT from log.ts (our logging utility)
  for (let i = 1; i < lines.length; i++) {
    const line = lines[i];
    
    // Skip lines from our logging utility
    if (line.includes('log.ts')) continue;
    
    // Match "at functionName (file:line:col)" or "at functionName@file:line:col"
    const match = line.match(/at\s+(?:([^\s(]+)\s+)?\(?([^:]+):(\d+)/);
    if (match) {
      const functionName = match[1] || null;
      const fileName = match[2] || 'unknown';
      const lineNumber = parseInt(match[3], 10) || 0;
      
      // Filter out internal names
      if (functionName && (functionName === 'Object' || functionName === 'Function' || functionName.includes('<anonymous>'))) {
        continue;
      }
      
      return { fileName, lineNumber, functionName };
    }
  }
  
  return { fileName: 'unknown', lineNumber: 0, functionName: null };
}

/**
 * Get caller's file, line, and function name from stack trace
 * Returns format: [filename:line] or [filename:line funcName]
 */
function getCallerLocation(): string {
  const err = new Error();
  const stackString = err.stack || '';
  
  // Use raw stack parsing (works reliably in both dev and production)
  const caller = parseCallerFromStack(stackString);
  
  // Extract relative path from src/ directory
  let relativePath = caller.fileName;
  const srcIndex = caller.fileName.indexOf('src/');
  if (srcIndex !== -1) {
    relativePath = caller.fileName.substring(srcIndex + 4);
  } else {
    relativePath = caller.fileName.split('/').pop() || caller.fileName.split('\\').pop() || caller.fileName;
  }
  
  // Include function name if available
  if (caller.functionName) {
    return `[${relativePath}:${caller.lineNumber} ${caller.functionName}]`;
  }
  return `[${relativePath}:${caller.lineNumber}]`;
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
