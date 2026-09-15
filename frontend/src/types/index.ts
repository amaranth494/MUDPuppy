// Session connection states
export type ConnectionState = 'disconnected' | 'connecting' | 'connected' | 'error';

// User type
export interface User {
  id: string;
  email: string;
  created_at: string;
}

// Session status response from backend
export interface SessionStatus {
  state: ConnectionState;
  connected_at?: string;
  host?: string;
  port?: number;
  last_activity_at?: string;
  last_error?: string;
  disconnect_reason?: string;
  // 02-04-02: carried on every response (no omitempty server-side); the badge's
  // refresh-correctness mechanism after a page reload (D-10)
  autopilot_state?: 'on' | 'waiting' | 'off';
}

// Connect request
export interface ConnectRequest {
  host: string;
  port: number;
  connection_id?: string;
}

// Connect response
export interface ConnectResponse {
  state: ConnectionState;
  session_id?: string;
  error?: string;
}

// Disconnect response
export interface DisconnectResponse {
  state: ConnectionState;
  reason?: string;
  error?: string;
}

// WebSocket message types
// 02-04-02: 'autopilot' added — best-effort live push of state changes (D-10)
export type WSMessageType = 'connect' | 'disconnect' | 'data' | 'error' | 'status' | 'autopilot';

export interface WSMessage {
  type: WSMessageType;
  host?: string;
  port?: number;
  data?: string;
  error?: string;
  status?: string;
  // 02-04-02: outbound-only — who sent this data message ('user' | 'alias' | 'trigger');
  // the server's wheel-grab reads this and treats absent as human, the safe direction
  source?: string;
}

// Error mapping
export interface ErrorMapping {
  backendError: string;
  userMessage: string;
}

export const ERROR_MAPPINGS: ErrorMapping[] = [
  { backendError: 'port not allowed', userMessage: 'Port not allowed by whitelist' },
  { backendError: 'private IP address', userMessage: 'Private addresses not allowed' },
  { backendError: 'user already has active session', userMessage: 'You already have an active connection' },
  { backendError: 'connection refused', userMessage: 'Connection refused by server' },
  { backendError: 'no such host', userMessage: 'Could not resolve host' },
  { backendError: 'i/o timeout', userMessage: 'Connection timed out' },
  { backendError: 'session expired', userMessage: 'Session expired, please reconnect' },
];

export function mapBackendError(error: string): string {
  for (const mapping of ERROR_MAPPINGS) {
    if (error.toLowerCase().includes(mapping.backendError.toLowerCase())) {
      return mapping.userMessage;
    }
  }
  return error;
}

// Saved Connection type (from backend)
export interface SavedConnection {
  id: string;
  user_id: string;
  name: string;
  host: string;
  port: number;
  protocol: string;
  created_at: string;
  updated_at: string;
  last_connected_at?: string;
  has_credentials: boolean;
  auto_login_enabled: boolean;
  username?: string;
}

// Create connection request
export interface CreateConnectionRequest {
  name: string;
  host: string;
  port: number;
  protocol?: string;
}

// Update connection request
export interface UpdateConnectionRequest {
  name: string;
  host: string;
  port: number;
  protocol?: string;
}

// Set credentials request
export interface SetCredentialsRequest {
  username: string;
  password: string;
  auto_login: boolean;
}

// Credential status response
export interface CredentialStatus {
  username: string;
  has_credentials: boolean;
  auto_login_enabled: boolean;
}

// PR01PH08: Automation credentials response (includes password if auto_login enabled)
export interface AutomationCredentials {
  username: string;
  password: string;
}

// Profile settings (SP04)
export interface ProfileSettings {
  scrollback_limit: number;
  echo_input: boolean;
  timestamp_output: boolean;
  word_wrap: boolean;
  // SP06PH07: Automation enabled setting (persisted per connection)
  automation_enabled?: boolean;
}

// Profile type (SP04)
export interface Profile {
  id: string;
  user_id: string;
  connection_id: string;
  keybindings: Record<string, string>;
  settings: ProfileSettings;
  aliases?: AutomationAliases;
  triggers?: AutomationTriggers;
  variables?: AutomationVariables;
  created_at: string;
  updated_at: string;
}

// Helper to normalize automation fields (handles legacy profiles created before SP05)
export function normalizeAutomationFields(profile: Profile): { aliases: AutomationAliases; triggers: AutomationTriggers; variables: AutomationVariables } {
  return {
    aliases: profile.aliases ?? { items: [] },
    triggers: profile.triggers ?? { items: [] },
    variables: profile.variables ?? { items: [] },
  };
}

// Update profile request (SP04)
export interface UpdateProfileRequest {
  keybindings?: Record<string, string>;
  settings?: ProfileSettings;
}

// ============================================
// Automation Types (SP05)
// ============================================

// Alias type - transforms user input commands
// Pattern matching: exact match (full input must equal pattern)
// Replacement: use %1, %2, etc. to reference captured groups (remaining input after pattern)
export interface Alias {
  id: string;
  pattern: string;
  replacement: string;
  enabled: boolean;
}

// Trigger type - executes commands based on output
export interface Trigger {
  id: string;
  match: string;
  type: 'contains';
  action: string;
  cooldown_ms: number;
  enabled: boolean;
}

// Variable type - reusable values for automation
export type VariableType = 'string' | 'number' | 'array' | 'boolean';

export interface Variable {
  id: string;
  name: string;
  value: string;
  type?: VariableType;
}

// Timer type - time-based automation
export interface Timer {
  id: string;
  name: string;
  duration: number; // in milliseconds
  repeat: boolean;
  commands: string;
  enabled: boolean;
}

// Automation response wrappers
export interface AliasesResponse {
  items: Alias[];
}

export interface TriggersResponse {
  items: Trigger[];
}

export interface VariablesResponse {
  items: Variable[];
}

export interface TimersResponse {
  items: Timer[];
}

// AI Player types (Phase 1 — profile foundation and policy gate)
export interface AISettings {
  model_name: string;
  call_cap: number | null;
  disengage_threshold: number | null;
}

export interface AISettingsResponse {
  conduct_rules: string;
  approach_guidance: string;
  ai_settings: AISettings;
}

export interface PolicyResponse {
  text: string;
  version: string;
  accepted: boolean;
  accepted_at: string | null;
  accepted_version: string | null;
}

export interface EngageGateResponse {
  allowed: boolean;
  message?: string;
}

// Automation wrapper types (SP05)
export interface AutomationAliases {
  items: Alias[];
}

export interface AutomationTriggers {
  items: Trigger[];
}

export interface AutomationVariables {
  items: Variable[];
}

// ============================================
// Help System Types (SP06)
// ============================================

export interface HelpSection {
  slug: string;
  title: string;
  description: string;
  sections: HelpSubsection[];
}

export interface HelpSubsection {
  title: string;
  content: string;
}

export interface HelpSummary {
  slug: string;
  title: string;
  description: string;
}
