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
  // Code review C3: the profile the switch is engaged or parked on, present while
  // on or waiting, so #AUTO OFF can be aimed at it after a page refresh.
  autopilot_connection_id?: string;
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
// 03-10: 'ai' added — the AI decision/system-notice push (plan 03-09, D-08)
export type WSMessageType = 'connect' | 'disconnect' | 'data' | 'error' | 'status' | 'autopilot' | 'ai';

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
  // 03-10: inbound-only, present on MsgTypeAI messages (plan 03-09's AIDecisionPayload,
  // matched field for field with internal/session/websocket.go)
  decision?: AIDecisionPayload;
}

// 03-10: the websocket payload of an "ai" message (plan 03-09, D-08). Kind is
// "decision" (reasoning/command are set, outcome is "sent") or "system" (message
// carries a locked failure/refusal notice). No window_text field — the server
// deliberately never sends the game text snapshot the model saw (T-3-15).
//
// 04-04: the outcome union gains 'cap' (D-14), 'blocked-repeatedly' (D-17),
// 'transient' and 'retrying' (D-15, D-16); 'goal' is included even though
// plan 04-06 is what emits it, so the colour lookup is written once
// (04-UI-SPEC.md). The state/count fields are meaningful on any event the
// driver emits during a stint (D-18, DR-3-03) and mirror
// internal/session/websocket.go's AIDecisionPayload field for field.
export interface AIDecisionPayload {
  id: string;
  kind: 'decision' | 'system';
  reasoning?: string;
  command?: string;
  outcome?: 'sent' | 'refused' | 'failed' | 'blocked' | 'cap' | 'blocked-repeatedly' | 'transient' | 'retrying' | 'goal';
  message?: string;
  timestamp: string;
  state?: 'on' | 'waiting' | 'off';
  calls?: number;
  call_cap?: number;
  call_cap_set?: boolean;
  failures?: number;
  blocks?: number;
  threshold?: number;
  // session_memory carries the current game session's full curated Session
  // Memory list (D-10, plan 04-08) on every event the driver emits during a
  // stint — the whole list, not a diff. Undefined on a message that carries
  // no memory snapshot (e.g. the goal-changed system line).
  session_memory?: string[];
}

// 03-10: one row of GET /api/v1/profiles/{connection_id}/decisions (D-12), the
// shape the AI Assist panel reloads on mount to rebuild its history after a refresh.
export interface StoredDecision {
  id: string;
  created_at: string;
  reasoning: string;
  command: string;
  outcome: string;
  failure_kind: string;
  notice: string;
}

// 03-11: one entry of GET /api/v1/profiles/{connection_id}/sessions (D-17) — the
// Logs page's left pane, matching internal/profiles/logs.go's
// SessionSummaryResponse field for field.
export interface GameSessionSummary {
  id: string;
  started_at: string;
  ended_at: string | null;
  line_count: number;
}

// 03-11: one line of GET /api/v1/profiles/{connection_id}/sessions/{session_id}
// (D-17) — the Logs page's right pane, matching internal/profiles/logs.go's
// SessionLineResponse field for field.
export interface TranscriptLine {
  seq: number;
  source: 'human' | 'ai' | 'game' | 'marker';
  text: string;
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
  never_issue_list: string;
  ai_settings: AISettings;
}

// GoalResponse is the GET response and PUT request body for the ai-goal
// sub-resource (D-01), matching internal/profiles/handler.go's GoalResponse
// field for field.
export interface GoalResponse {
  goal: string;
}

// SessionMemoryResponse is the GET response for the ai-memory sub-resource
// (D-10), matching internal/profiles/handler.go's SessionMemoryResponse
// field for field. Read-only — there is no PUT counterpart this phase.
export interface SessionMemoryResponse {
  session_memory: string[];
}

// DeleteCapturedTextResponse is the response for the owner's immediate
// "delete captured text now" action (D-21), matching
// internal/profiles/handler.go's DeleteCapturedTextResponse field for
// field.
export interface DeleteCapturedTextResponse {
  snapshots_cleared: number;
  transcript_lines_deleted: number;
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
