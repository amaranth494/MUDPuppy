// PR02PH09: Logging utility - standardized logging for browser console
// All console.log calls should use this to ensure consistent formatting

/**
 * Get the caller's location (file:function) from the stack trace
 * Works in both development and production builds
 */
function getCallerLocation(): string {
  // Create error to capture stack trace
  const err = new Error();
  const stack = err.stack || '';
  
  // Parse stack - look at first few lines
  const lines = stack.split('\n');
  
  // Skip: Error line, getCallerLocation, formatTimestamp, logToConsole, etc.
  // Start from index 4 to get past our internal functions
  for (let i = 4; i < Math.min(10, lines.length); i++) {
    const line = lines[i];
    if (!line.trim()) continue;
    
    // Skip our logging utility lines
    if (line.includes('log.ts')) continue;
    
    // Match pattern: "at functionName (file:line:col)" or "at file:line:col"
    const match = line.match(/at\s+(?:(\w+)\s+)?\(?([^:]+):(\d+)/);
    if (match) {
      const funcName = match[1] || '';
      const fileName = match[2];
      const lineNum = match[3];
      
      // Extract just the filename
      const shortName = fileName.split('/').pop() || fileName.split('\\').pop() || fileName;
      
      // Skip bundled files
      if (shortName.includes('index-')) continue;
      
      if (funcName) {
        return `[${shortName}:${funcName}]`;
      }
      return `[${shortName}:${lineNum}]`;
    }
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
 * Format: [LOG] YYYY-MM-DD HH:MM:SS [file:function] message
 */
export function logToConsole(message: string, ...args: unknown[]): void {
  const timestamp = formatTimestamp();
  const location = getCallerLocation();
  console.log(`[LOG] ${timestamp} ${location} ${message}`, ...args);
}

/**
 * Log error to browser console with standardized formatting
 * Format: [LOG ERROR] YYYY-MM-DD HH:MM:SS [file:function] message
 */
export function logErrorToConsole(message: string, ...args: unknown[]): void {
  const timestamp = formatTimestamp();
  const location = getCallerLocation();
  console.error(`[LOG ERROR] ${timestamp} ${location} ${message}`, ...args);
}

/**
 * Log warning to browser console with standardized formatting
 * Format: [LOG WARN] YYYY-MM-DD HH:MM:SS [file:function] message
 */
export function logWarnToConsole(message: string, ...args: unknown[]): void {
  const timestamp = formatTimestamp();
  const location = getCallerLocation();
  console.warn(`[LOG WARN] ${timestamp} ${location} ${message}`, ...args);
}

/**
 * Log debug to browser console with standardized formatting
 * Format: [LOG DEBUG] YYYY-MM-DD HH:MM:SS [file:function] message
 */
export function logDebugToConsole(message: string, ...args: unknown[]): void {
  const timestamp = formatTimestamp();
  const location = getCallerLocation();
  console.debug(`[LOG DEBUG] ${timestamp} ${location} ${message}`, ...args);
}
