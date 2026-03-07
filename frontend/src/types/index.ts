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
}

// Connect request
export interface ConnectRequest {
  host: string;
  port: number;
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
export type WSMessageType = 'connect' | 'disconnect' | 'data' | 'error' | 'status';

export interface WSMessage {
  type: WSMessageType;
  host?: string;
  port?: number;
  data?: string;
  error?: string;
  status?: string;
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

// Profile settings (SP04)
export interface ProfileSettings {
  scrollback_limit: number;
  echo_input: boolean;
  timestamp_output: boolean;
  word_wrap: boolean;
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
export interface Alias {
  id: string;
  pattern: string;
  type: 'exact' | 'prefix';
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
export interface Variable {
  id: string;
  name: string;
  value: string;
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
