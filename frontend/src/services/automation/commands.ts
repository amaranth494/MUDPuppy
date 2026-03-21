/**
 * Command Registry - Single source of truth for automation commands
 * 
 * This file defines all known automation commands in a data-driven format.
 * The parser and evaluator should read from this registry rather than
 * hardcoding commands in switch statements.
 * 
 * PR02PH09: Created to make command registration scalable - new commands
 * only need to be added here to be automatically recognized.
 */

// Command categories
export const CommandCategories = {
  CONDITIONAL: 'conditional',
  VARIABLE: 'variable',
  TIMER: 'timer',
  OUTPUT: 'output',
  FLOW: 'flow',
} as const;

// Command definitions - single source of truth
export interface CommandDefinition {
  name: string;
  category: typeof CommandCategories[keyof typeof CommandCategories];
  requiresArgs: boolean;
  description: string;
  handler?: string; // Handler function name in evaluator
}

export const CommandRegistry: Record<string, CommandDefinition> = {
  // Conditional commands
  'IF': {
    name: 'IF',
    category: CommandCategories.CONDITIONAL,
    requiresArgs: true,
    description: 'Begin conditional block',
  },
  'ELSE': {
    name: 'ELSE',
    category: CommandCategories.CONDITIONAL,
    requiresArgs: false,
    description: 'Alternate condition branch',
  },
  'ENDIF': {
    name: 'ENDIF',
    category: CommandCategories.CONDITIONAL,
    requiresArgs: false,
    description: 'End conditional block',
  },
  
  // Variable commands
  'SET': {
    name: 'SET',
    category: CommandCategories.VARIABLE,
    requiresArgs: true,
    description: 'Set variable value',
  },
  'ADD': {
    name: 'ADD',
    category: CommandCategories.VARIABLE,
    requiresArgs: true,
    description: 'Add to numeric variable',
  },
  'SUB': {
    name: 'SUB',
    category: CommandCategories.VARIABLE,
    requiresArgs: true,
    description: 'Subtract from numeric variable',
  },
  
  // Timer commands
  'TIMER': {
    name: 'TIMER',
    category: CommandCategories.TIMER,
    requiresArgs: true,
    description: 'Create or update timer',
  },
  'ENDTIMER': {
    name: 'ENDTIMER',
    category: CommandCategories.TIMER,
    requiresArgs: false,
    description: 'End timer definition',
  },
  'START': {
    name: 'START',
    category: CommandCategories.TIMER,
    requiresArgs: true,
    description: 'Start a timer',
  },
  'STOP': {
    name: 'STOP',
    category: CommandCategories.TIMER,
    requiresArgs: true,
    description: 'Stop a timer',
  },
  'CHECK': {
    name: 'CHECK',
    category: CommandCategories.TIMER,
    requiresArgs: true,
    description: 'Check timer status',
  },
  'CANCEL': {
    name: 'CANCEL',
    category: CommandCategories.TIMER,
    requiresArgs: true,
    description: 'Cancel a timer',
  },
  
  // Output commands
  'ECHO': {
    name: 'ECHO',
    category: CommandCategories.OUTPUT,
    requiresArgs: false,
    description: 'Output message to terminal',
  },
  'LOG': {
    name: 'LOG',
    category: CommandCategories.OUTPUT,
    requiresArgs: true,
    description: 'Output to log',
  },
  'HELP': {
    name: 'HELP',
    category: CommandCategories.OUTPUT,
    requiresArgs: false,
    description: 'Show help',
  },
  'GAG': {
    name: 'GAG',
    category: CommandCategories.OUTPUT,
    requiresArgs: false,
    description: 'Suppress output',
  },
  'DELAY': {
    name: 'DELAY',
    category: CommandCategories.FLOW,
    requiresArgs: true,
    description: 'Delay command execution',
  },
  'EXPAND': {
    name: 'EXPAND',
    category: CommandCategories.FLOW,
    requiresArgs: false,
    description: 'Toggle alias expansion',
  },
};

// Get all command names as a Set for easy lookup
export const KNOWN_COMMANDS = new Set(Object.keys(CommandRegistry));

// Get commands by category
export function getCommandsByCategory(category: string): CommandDefinition[] {
  return Object.values(CommandRegistry).filter(cmd => cmd.category === category);
}

// Check if command requires arguments
export function commandRequiresArgs(commandName: string): boolean {
  const cmd = CommandRegistry[commandName];
  return cmd ? cmd.requiresArgs : false;
}
