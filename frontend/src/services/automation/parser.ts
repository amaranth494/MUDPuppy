/**
 * Automation Parser Module
 * 
 * Handles tokenization and parsing of automation commands.
 * Commands are defined in commands.ts - edit that file to add new commands.
 * 
 * PR02PH09: Now uses CommandRegistry from commands.ts for scalability.
 */

import { CommandRegistry } from './commands';

export type TokenType = 
  | 'COMMAND'    // #IF, #ELSE, #ENDIF, #SET, #TIMER, #ENDTIMER, #CANCEL
  | 'TEXT'       // Plain text content
  | 'CONDITION'  // Conditional expression in #IF
  | 'VARIABLE';  // ${variable_name} substitution

export interface Token {
  type: TokenType;
  value: string;
  line: number;
  column: number;
}

export interface CommandToken extends Token {
  type: 'COMMAND';
  // PR02PH09: Use string type to accept any command from registry
  // Commands are validated against KNOWN_COMMANDS at parse time
  command: string;
  args?: string;
}

export interface TextToken extends Token {
  type: 'TEXT';
}

export interface ConditionToken extends Token {
  type: 'CONDITION';
  expression: string;
}

export interface VariableToken extends Token {
  type: 'VARIABLE';
  name: string;
}

export type ParsedToken = CommandToken | TextToken | ConditionToken | VariableToken;

export interface ParseResult {
  success: boolean;
  tokens: ParsedToken[];
  errors: ParseError[];
}

export interface ParseError {
  message: string;
  line: number;
  column: number;
}

/**
 * Parser class for automation commands
 */
export class Parser {
  private tokens: ParsedToken[] = [];
  private errors: ParseError[] = [];
  private line: number = 1;
  private column: number = 1;
  private position: number = 0;
  private input: string = '';

  /**
   * Parse an automation string into tokens
   */
  parse(input: string): ParseResult {
    this.reset();
    this.input = input;
    
    if (!input || input.trim() === '') {
      return {
        success: true,
        tokens: [],
        errors: []
      };
    }

    this.tokenize();
    
    return {
      success: this.errors.length === 0,
      tokens: this.tokens,
      errors: this.errors
    };
  }

  /**
   * Reset parser state
   */
  private reset(): void {
    this.tokens = [];
    this.errors = [];
    this.line = 1;
    this.column = 1;
    this.position = 0;
    this.input = '';
  }

  /**
   * Tokenize the input string
   */
  private tokenize(): void {
    while (this.position < this.input.length) {
      const char = this.input[this.position];
      
      // Skip whitespace but preserve newlines for line tracking
      if (char === ' ' || char === '\t') {
        this.position++;
        this.column++;
        continue;
      }
      
      // Track newlines for error reporting
      if (char === '\n') {
        this.line++;
        this.column = 1;
        this.position++;
        continue;
      }
      
      // Check for # commands
      if (char === '#') {
        this.parseCommand();
        continue;
      }
      
      // Check for variable substitution ${...}
      if (char === '$' && this.input.slice(this.position, this.position + 2) === '${') {
        this.parseVariable();
        continue;
      }
      
      // Parse plain text
      this.parseText();
    }
  }

  /**
   * Parse a # command
   */
  private parseCommand(): void {
    const startColumn = this.column;
    this.position++; // Skip #
    this.column++;
    
    // Read command name
    let commandName = '';
    while (this.position < this.input.length) {
      const char = this.input[this.position];
      if (/[A-Z]/.test(char)) {
        commandName += char;
        this.position++;
        this.column++;
      } else {
        break;
      }
    }
    
    // Validate command
    const validCommands = ['IF', 'ELSE', 'ENDIF', 'SET', 'ADD', 'SUB', 'TIMER', 'ENDTIMER', 'CANCEL', 'START', 'STOP', 'CHECK'];
    const upperCommand = commandName.toUpperCase();
    
    if (!validCommands.includes(upperCommand)) {
      this.errors.push({
        message: `Unknown command: #${commandName}`,
        line: this.line,
        column: startColumn
      });
      return;
    }
    
    // Read arguments (rest of line)
    let args = '';
    while (this.position < this.input.length) {
      const char = this.input[this.position];
      if (char === '\n' || char === '\r') {
        break;
      }
      args += char;
      this.position++;
      this.column++;
    }
    
    const token: CommandToken = {
      type: 'COMMAND',
      value: `#${upperCommand}`,
      line: this.line,
      column: startColumn,
      command: upperCommand as CommandToken['command'],
      args: args.trim()
    };
    
    this.tokens.push(token);
  }

  /**
   * Parse a variable substitution ${...}
   */
  private parseVariable(): void {
    const startColumn = this.column;
    this.position += 2; // Skip ${
    this.column += 2;
    
    let name = '';
    while (this.position < this.input.length) {
      const char = this.input[this.position];
      if (char === '}') {
        break;
      }
      if (char === '\n') {
        this.errors.push({
          message: 'Unclosed variable substitution',
          line: this.line,
          column: startColumn
        });
        return;
      }
      name += char;
      this.position++;
      this.column++;
    }
    
    if (this.input[this.position] === '}') {
      this.position++;
      this.column++;
    }
    
    const token: VariableToken = {
      type: 'VARIABLE',
      value: `\$\{${name}\}`,
      line: this.line,
      column: startColumn,
      name
    };
    
    this.tokens.push(token);
  }

  /**
   * Parse plain text content
   */
  private parseText(): void {
    const startColumn = this.column;
    let text = '';
    
    while (this.position < this.input.length) {
      const char = this.input[this.position];
      
      // Stop at # commands or variable substitutions
      if (char === '#' || (char === '$' && this.input.slice(this.position, this.position + 2) === '${')) {
        break;
      }
      
      // Track newlines
      if (char === '\n') {
        break;
      }
      
      text += char;
      this.position++;
      this.column++;
    }
    
    if (text.length > 0) {
      const token: TextToken = {
        type: 'TEXT',
        value: text,
        line: this.line,
        column: startColumn
      };
      
      this.tokens.push(token);
    }
  }
}

/**
 * Validate syntax of parsed tokens
 */
export function validateSyntax(tokens: ParsedToken[]): ParseError[] {
  const errors: ParseError[] = [];
  const ifStack: number[] = [];
  
  for (let i = 0; i < tokens.length; i++) {
    const token = tokens[i];
    
    if (token.type === 'COMMAND') {
      // PR02PH09: Use CommandRegistry for validation - scalable approach
      const cmdDef = CommandRegistry[token.command];
      
      if (!cmdDef) {
        // Unknown command - this should not happen if parser is correct
        errors.push({
          message: `Unknown command: #${token.command}`,
          line: token.line,
          column: token.column
        });
      } else {
        // Validate based on command definition
        if (cmdDef.requiresArgs && (!token.args || token.args.trim() === '')) {
          errors.push({
            message: `#${token.command} requires arguments`,
            line: token.line,
            column: token.column
          });
        }
        
        // Special handling for conditional commands
        switch (token.command) {
          case 'IF':
            ifStack.push(token.line);
            break;
          case 'ELSE':
            if (ifStack.length === 0) {
              errors.push({
                message: '#ELSE without matching #IF',
                line: token.line,
                column: token.column
              });
            }
            break;
          case 'ENDIF':
            if (ifStack.length === 0) {
              errors.push({
                message: '#ENDIF without matching #IF',
                line: token.line,
                column: token.column
              });
            } else {
              ifStack.pop();
            }
            break;
        }
      }
    }
  }
  
  // Check for unclosed #IF blocks
  if (ifStack.length > 0) {
    errors.push({
      message: `Unclosed #IF block (started on line ${ifStack[0]})`,
      line: ifStack[ifStack.length - 1],
      column: 1
    });
  }
  
  return errors;
}

// Default parser instance
export const parser = new Parser();
