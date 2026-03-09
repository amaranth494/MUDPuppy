// Automation Templates Library
// SP06PH05 - Templates for common automation patterns
// These templates can be imported with one click into Aliases, Triggers, or Variables

import { Alias, Trigger, Variable } from './types';

// ============================================
// Alias Templates
// ============================================

export interface AliasTemplate {
  name: string;
  description: string;
  pattern: string;
  replacement: string;
}

export const aliasTemplates: AliasTemplate[] = [
  {
    name: 'Combat Attack',
    description: 'Attack a target with offensive stance',
    pattern: 'k',
    replacement: 'kill %1;stance offensive',
  },
  {
    name: 'Quick Look',
    description: 'Short alias for looking around',
    pattern: 'l',
    replacement: 'look',
  },
  {
    name: 'Get Item',
    description: 'Get an item from the room or container',
    pattern: 'g',
    replacement: 'get %1',
  },
  {
    name: 'Cast Spell',
    description: 'Cast a spell at a target',
    pattern: 'c',
    replacement: 'cast %1',
  },
  {
    name: 'Drink Potion',
    description: 'Drink from a variable container',
    pattern: 'dr',
    replacement: 'drink ${heal}',
  },
  {
    name: 'Inventory',
    description: 'Check inventory quickly',
    pattern: 'i',
    replacement: 'inventory',
  },
  {
    name: 'Say Local',
    description: 'Say something to nearby players',
    pattern: 's',
    replacement: 'say %1',
  },
  {
    name: 'Tell Player',
    description: 'Tell something to a specific player',
    pattern: 't',
    replacement: 'tell %1 %2',
  },
];

// ============================================
// Trigger Templates
// ============================================

export interface TriggerTemplate {
  name: string;
  description: string;
  match: string;
  action: string;
  cooldown_ms: number;
}

export const triggerTemplates: TriggerTemplate[] = [
  {
    name: 'Auto-Eat',
    description: 'Automatically eat when hungry',
    match: 'You are hungry',
    action: 'eat bread',
    cooldown_ms: 5000,
  },
  {
    name: 'Auto-Drink',
    description: 'Automatically drink when thirsty',
    match: 'You are thirsty',
    action: 'drink water',
    cooldown_ms: 5000,
  },
  {
    name: 'Low Health Alert',
    description: 'Show warning when health is low',
    match: 'Your health is low',
    action: 'say WARNING: Low health!',
    cooldown_ms: 10000,
  },
  {
    name: 'Gold Pickup',
    description: 'Announce gold pickups',
    match: 'You pick up',
    action: 'gold',
    cooldown_ms: 1000,
  },
  {
    name: 'Death Warning',
    description: 'Alert when near death',
    match: 'You are in very bad shape',
    action: 'say CRITICAL: Near death!',
    cooldown_ms: 10000,
  },
  {
    name: 'Experience Gain',
    description: 'Announce level gains',
    match: 'You have gained',
    action: 'score',
    cooldown_ms: 1000,
  },
  {
    name: 'Door Observation',
    description: 'Note when a door is noticed',
    match: 'You notice a door',
    action: 'look door',
    cooldown_ms: 5000,
  },
];

// ============================================
// Variable Templates
// ============================================

export interface VariableTemplate {
  name: string;
  description: string;
  value: string;
}

export const variableTemplates: VariableTemplate[] = [
  {
    name: 'heal',
    description: 'Default healing item',
    value: 'potion',
  },
  {
    name: 'target',
    description: 'Current combat target',
    value: '',
  },
  {
    name: 'assist',
    description: 'Player to assist in combat',
    value: '',
  },
  {
    name: 'armor',
    description: 'Current armor set',
    value: 'leather',
  },
  {
    name: 'weapon',
    description: 'Current weapon',
    value: 'sword',
  },
];

// ============================================
// Helper Functions
// ============================================

/**
 * Create an Alias from an AliasTemplate
 */
export function createAliasFromTemplate(template: AliasTemplate): Alias {
  return {
    id: crypto.randomUUID(),
    pattern: template.pattern,
    replacement: template.replacement,
    enabled: true,
  };
}

/**
 * Create a Trigger from a TriggerTemplate
 */
export function createTriggerFromTemplate(template: TriggerTemplate): Trigger {
  return {
    id: crypto.randomUUID(),
    match: template.match,
    type: 'contains',
    action: template.action,
    cooldown_ms: template.cooldown_ms,
    enabled: true,
  };
}

/**
 * Create a Variable from a VariableTemplate
 */
export function createVariableFromTemplate(template: VariableTemplate): Variable {
  return {
    id: crypto.randomUUID(),
    name: template.name,
    value: template.value,
  };
}
