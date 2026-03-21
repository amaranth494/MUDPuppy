/**
 * ICM (Internal Command Module) Frontend Adapter
 * 
 * Provides a TypeScript interface for command validation, normalization,
 * and submission through the ICM pipeline.
 */

const API_BASE = '/api/v1';

// Types matching backend ICM types

/** Operator family types */
export type OperatorFamily = '#' | '@' | '$' | '%';

/** Execution context */
export type ExecutionContext = 'preview' | 'submission' | 'automation' | 'operational';

/** ICM request options */
export interface ICMOptions {
  skipSafety?: boolean;
  dryRun?: boolean;
  trace?: boolean;
}

/** ICM request */
export interface ICMRequest {
  raw: string;
  context: ExecutionContext;
  sessionId: string;
  userId?: string;
  variables?: Record<string, string>;
  options?: ICMOptions;
}

/** ICM error context */
export interface ICMErrorContext {
  phase: string;
  position?: number;
  input?: string;
}

/** ICM error */
export interface ICMError {
  code: string;
  message: string;
  userMessage: string;
  details?: Record<string, unknown>;
  context?: ICMErrorContext;
}

/** State effect from command execution */
export interface StateEffect {
  type: string;
  key: string;
  value: unknown;
}

/** Result metadata */
export interface ResultMetadata {
  executionTime: number;
  handler: string;
}

/** Command result */
export interface CommandResult {
  output?: string;
  effects?: StateEffect[];
  metadata: ResultMetadata;
}

/** Processing trace phase */
export interface PhaseTrace {
  phase: string;
  duration: number;
  input: string;
  output: string;
  error?: string;
}

/** Processing trace */
export interface ProcessingTrace {
  phases: PhaseTrace[];
  warnings?: string[];
}

/** ICM response */
export interface ICMResponse {
  isInternal: boolean;
  operator?: OperatorFamily;
  command?: string;
  normalized?: string;
  resolved?: string;
  result?: CommandResult;
  error?: ICMError;
  processingTime: number;
  trace?: ProcessingTrace;
}

/** Classification result */
export interface ClassificationResult {
  isInternal: boolean;
  operator?: OperatorFamily;
  command?: string;
  shouldPassThrough: boolean;
  passThroughValue: string;
}

/** Normalized command */
export interface NormalizedCommand {
  canonical: string;
  operator?: OperatorFamily;
  command: string;
  args: string[];
  originalInput: string;
  transformations: string[];
  isInternal: boolean;
  requiresExecution: boolean;
}

// Frontend-only ICM implementation (for preview/validation)

/** Reserved operators */
const RESERVED_OPERATORS: Record<string, OperatorFamily> = {
  '#': '#',
  '@': '@',
  '$': '$',
  '%': '%',
};

/** Known structured commands */
const KNOWN_STRUCTURED_COMMANDS = new Set([
  'IF', 'ELSE', 'ENDIF',
  'SET', 'ADD', 'SUB',
  'TIMER', 'START', 'STOP', 'CHECK', 'CANCEL',
  'LOG', 'ECHO', 'HELP', 'GAG', 'DELAY', 'EXPAND',
]);

/** Known system variables */
const KNOWN_SYSTEM_VARIABLES = new Set([
  'TIME', 'DATE', 'CHARACTER', 'SERVER', 'SESSIONID', 'HOST', 'PORT',
]);

/** Escape sequences */
const ESCAPE_SEQUENCES: Record<string, string> = {
  '\\#': '#',
  '\\@': '@',
  '\\$': '$',
  '\\%': '%',
  '\\\\': '\\',
  '\\n': '\n',
  '\\t': '\t',
  '\\r': '\r',
};

/**
 * Frontend-only recognizer for preview validation
 */
export function recognizeCommand(input: string): ClassificationResult {
  const trimmed = input.trim();
  
  if (!trimmed) {
    return {
      isInternal: false,
      shouldPassThrough: false,
      passThroughValue: '',
    };
  }

  const firstChar = trimmed[0] as string;
  const op = RESERVED_OPERATORS[firstChar];

  if (!op) {
    // Unknown operator - pass through to MUD
    return {
      isInternal: false,
      shouldPassThrough: true,
      passThroughValue: trimmed,
    };
  }

  // It's a reserved operator - classify further
  return classifyReservedOperator(trimmed, op);
}

function classifyReservedOperator(input: string, op: OperatorFamily): ClassificationResult {
  // Extract command part
  const rest = input.slice(1).trim();
  const parts = rest.split(/\s+/);
  const command = parts[0] || '';

  // Check if known
  const isKnown = isKnownCommand(op, command);

  if (!isKnown) {
    // Unknown command in reserved family - error, not pass through
    return {
      isInternal: true,
      operator: op,
      command: input,
      shouldPassThrough: false,
      passThroughValue: '',
    };
  }

  return {
    isInternal: true,
    operator: op,
    command: input,
    shouldPassThrough: false,
    passThroughValue: '',
  };
}

function isKnownCommand(op: OperatorFamily, command: string): boolean {
  switch (op) {
    case '#':
      return KNOWN_STRUCTURED_COMMANDS.has(command.toUpperCase());
    case '@':
      // Aliases checked at runtime
      return true;
    case '$':
      // Variables don't need to be known
      return true;
    case '%':
      // Check numeric or known system variable
      if (/^\d+$/.test(command)) {
        return true;
      }
      return KNOWN_SYSTEM_VARIABLES.has(command.toUpperCase());
    default:
      return false;
  }
}

/**
 * Normalize a command (frontend-only)
 */
export function normalizeCommand(input: string): NormalizedCommand {
  const trimmed = input.trim();
  
  // Classify first
  const classification = recognizeCommand(trimmed);
  
  if (!classification.isInternal) {
    return {
      canonical: trimmed,
      command: trimmed,
      args: [],
      originalInput: input,
      transformations: [],
      isInternal: false,
      requiresExecution: false,
    };
  }

  // Parse based on operator
  const operator = classification.operator!;
  const rest = trimmed.slice(1).trim();
  const parts = parseArgs(rest);
  
  let command: string;
  let args: string[];

  switch (operator) {
    case '#':
      command = parts[0].toUpperCase();
      args = parts.slice(1);
      break;
    case '@':
    case '$':
    case '%':
      command = parts[0];
      args = parts.slice(1);
      break;
    default:
      command = trimmed;
      args = [];
  }

  const canonical = buildCanonical(operator, command, args);

  return {
    canonical,
    operator,
    command,
    args,
    originalInput: input,
    transformations: ['normalize'],
    isInternal: true,
    requiresExecution: true,
  };
}

function parseArgs(input: string): string[] {
  const args: string[] = [];
  let current = '';
  let inQuotes = false;
  let quoteChar = '';

  for (let i = 0; i < input.length; i++) {
    const ch = input[i];

    if (!inQuotes && (ch === '"' || ch === "'")) {
      inQuotes = true;
      quoteChar = ch;
    } else if (inQuotes && ch === quoteChar) {
      inQuotes = false;
      quoteChar = '';
    } else if (!inQuotes && /\s/.test(ch)) {
      if (current) {
        args.push(current);
        current = '';
      }
    } else {
      current += ch;
    }
  }

  if (current) {
    args.push(current);
  }

  return args;
}

function buildCanonical(op: OperatorFamily, cmd: string, args: string[]): string {
  const parts = [op, cmd, ...args];
  return parts.join(' ');
}

/**
 * Validate a command (frontend-only preview)
 */
export function validateCommand(input: string): ICMError | null {
  const classification = recognizeCommand(input);
  
  if (!classification.isInternal) {
    // Not an internal command - no validation needed
    return null;
  }

  // Check for unknown commands
  const rest = input.trim().slice(1).trim();
  const parts = parseArgs(rest);
  const command = parts[0]?.toUpperCase() || '';

  // Validate based on operator
  switch (classification.operator) {
    case '#':
      if (!KNOWN_STRUCTURED_COMMANDS.has(command)) {
        return {
          code: 'E2004',
          message: `Unknown directive: ${command}`,
          userMessage: `Unknown command: ${command}`,
        };
      }
      // Validate specific commands
      if (command === 'SET' && parts.length < 3) {
        return {
          code: 'E2001',
          message: 'SET requires name and value',
          userMessage: 'Missing required argument: name and value',
        };
      }
      if (command === 'TIMER' && parts.length < 4) {
        return {
          code: 'E2001',
          message: 'TIMER requires name, interval, and command',
          userMessage: 'Missing required arguments',
        };
      }
      break;

    case '$':
      if (!/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(parts[0])) {
        return {
          code: 'E2002',
          message: 'Invalid variable name',
          userMessage: 'Invalid variable name',
        };
      }
      break;

    case '%':
      // Check for malformed
      if (parts[0] === '') {
        return null; // Pass through
      }
      break;
  }

  return null;
}

/**
 * Resolve escape sequences
 */
export function resolveEscapes(input: string): string {
  let result = input;
  for (const [escape, literal] of Object.entries(ESCAPE_SEQUENCES)) {
    result = result.replace(new RegExp(escape.replace(/[\\]/g, '\\\\'), 'g'), literal);
  }
  return result;
}

/**
 * Check if input should pass through to MUD
 */
export function shouldPassThrough(input: string): boolean {
  const classification = recognizeCommand(input);
  return classification.shouldPassThrough;
}

/**
 * Get the pass-through value
 */
export function getPassThroughValue(input: string): string {
  const classification = recognizeCommand(input);
  if (classification.shouldPassThrough) {
    return classification.passThroughValue;
  }
  return '';
}

// Backend API integration

const ICM_API_BASE = `${API_BASE}/icm`;

/**
 * Validate command via backend API
 */
export async function validateCommandRemote(
  input: string,
  sessionId: string,
  context: ExecutionContext = 'preview'
): Promise<ICMResponse> {
  const request: ICMRequest = {
    raw: input,
    context,
    sessionId,
  };

  const response = await fetch(`${ICM_API_BASE}/validate`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify(request),
  });

  if (!response.ok) {
    throw new Error(`ICM validate failed: ${response.statusText}`);
  }

  return response.json();
}

/**
 * Normalize command via backend API
 */
export async function normalizeCommandRemote(
  input: string,
  sessionId: string
): Promise<ICMResponse> {
  const request: ICMRequest = {
    raw: input,
    context: 'preview',
    sessionId,
  };

  const response = await fetch(`${ICM_API_BASE}/normalize`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify(request),
  });

  if (!response.ok) {
    throw new Error(`ICM normalize failed: ${response.statusText}`);
  }

  return response.json();
}

/**
 * Execute command via backend API
 */
export async function executeCommandRemote(
  input: string,
  sessionId: string,
  context: ExecutionContext = 'submission'
): Promise<ICMResponse> {
  const request: ICMRequest = {
    raw: input,
    context,
    sessionId,
  };

  const response = await fetch(`${ICM_API_BASE}/execute`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify(request),
  });

  if (!response.ok) {
    throw new Error(`ICM execute failed: ${response.statusText}`);
  }

  return response.json();
}

/**
 * Preview command via backend API
 */
export async function previewCommandRemote(
  input: string,
  sessionId: string
): Promise<ICMResponse> {
  return executeCommandRemote(input, sessionId, 'preview');
}

/**
 * Combined validation + normalization (frontend fallback)
 * Uses local logic when backend unavailable
 */
export async function validateAndNormalize(
  input: string,
  sessionId: string,
  useBackend: boolean = true
): Promise<ICMResponse> {
  if (useBackend) {
    try {
      return await normalizeCommandRemote(input, sessionId);
    } catch {
      // Fall back to frontend
    }
  }

  // Frontend fallback
  const startTime = Date.now();
  const validationError = validateCommand(input);
  const normalized = normalizeCommand(input);

  return {
    isInternal: normalized.isInternal,
    operator: normalized.operator,
    command: normalized.command,
    normalized: normalized.canonical,
    resolved: normalized.canonical,
    error: validationError || undefined,
    processingTime: Date.now() - startTime,
  };
}

/**
 * Process command for submission
 * Determines if command should go to ICM or pass through to MUD
 */
export async function processCommand(
  input: string,
  sessionId: string,
  useBackend: boolean = true
): Promise<{
  shouldPassThrough: boolean;
  processedValue: string;
  response?: ICMResponse;
  error?: ICMError;
}> {
  // First, check if it's an internal command
  const classification = recognizeCommand(input);

  if (!classification.isInternal) {
    return {
      shouldPassThrough: true,
      processedValue: input,
    };
  }

  // It's an internal command - process through ICM
  try {
    if (useBackend) {
      const response = await executeCommandRemote(input, sessionId);
      
      if (response.error) {
        return {
          shouldPassThrough: false,
          processedValue: '',
          error: response.error,
        };
      }

      return {
        shouldPassThrough: false,
        processedValue: response.resolved || response.normalized || input,
        response,
      };
    }

    // Frontend fallback
    const error = validateCommand(input);
    if (error) {
      return {
        shouldPassThrough: false,
        processedValue: '',
        error,
      };
    }

    const normalized = normalizeCommand(input);
    return {
      shouldPassThrough: false,
      processedValue: normalized.canonical,
    };
  } catch (err) {
    // On error, allow pass-through for backwards compatibility
    return {
      shouldPassThrough: true,
      processedValue: input,
    };
  }
}
