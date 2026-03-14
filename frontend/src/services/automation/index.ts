/**
 * Automation Module Index
 * 
 * Exports parser and evaluator modules for automation logic:
 * - Parser: Tokenization and syntax validation
 * - Evaluator: Condition parsing, evaluation, and #IF/#ELSE/#ENDIF execution
 * 
 * Variable Resolution Precedence:
 *   session (%1, %2) > profile (${name}) > system (%TIME, %CHARACTER)
 * 
 * #SET Persistence:
 * - Backend write fails → value does not persist
 * - Automation continues, failure is logged
 * 
 * Session Variables:
 * - %1, %2 are temporary variables populated from alias arguments or trigger captures
 * - Cleared on disconnect and new connect
 */

export * from './parser';
export * from './evaluator';
export * from './timer';

// Re-export types for convenience
export type { VariableValue, VariableResolver, VariableStore } from './evaluator';
export { SimpleVariableStore, isSystemVariable } from './evaluator';
