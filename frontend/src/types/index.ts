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
