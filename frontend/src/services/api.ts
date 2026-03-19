import { User, SessionStatus, ConnectRequest, ConnectResponse, DisconnectResponse, WSMessage, SavedConnection, CreateConnectionRequest, UpdateConnectionRequest, SetCredentialsRequest, CredentialStatus, AutomationCredentials, Profile, UpdateProfileRequest, Alias, Trigger, Variable, Timer, AliasesResponse, TriggersResponse, VariablesResponse, TimersResponse, HelpSection, HelpSummary } from '../types';

const API_BASE = '/api/v1';

// Helper to handle auth errors and redirect to login
function handleAuthError(response: Response): void {
  if (response.status === 401) {
    // Don't redirect if already on login page
    if (!window.location.pathname.startsWith('/login')) {
      window.location.href = '/login';
    }
  }
}

// Check if user is authenticated
export async function checkAuth(): Promise<User | null> {
  try {
    const response = await fetch(`${API_BASE}/me`, {
      credentials: 'include',
    });
    handleAuthError(response);
    if (response.ok) {
      return await response.json();
    }
    return null;
  } catch (error) {
    console.error('Auth check failed:', error);
    return null;
  }
}

// Logout
export async function logout(): Promise<void> {
  await fetch(`${API_BASE}/logout`, {
    method: 'DELETE',
    credentials: 'include',
  });
}

// Get session status - authoritative source for connection state
export async function getSessionStatus(): Promise<SessionStatus> {
  const response = await fetch(`${API_BASE}/session/status`, {
    credentials: 'include',
  });
  handleAuthError(response);
  if (!response.ok) {
    throw new Error('Failed to get session status');
  }
  return await response.json();
}

// Connect to MUD server
export async function connectToMud(request: ConnectRequest): Promise<ConnectResponse> {
  const response = await fetch(`${API_BASE}/session/connect`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify(request),
  });
  
  handleAuthError(response);
  const data = await response.json();
  if (!response.ok) {
    throw new Error(data.error || 'Connection failed');
  }
  return data;
}

// Disconnect from MUD server
export async function disconnectFromMud(reason?: string): Promise<DisconnectResponse> {
  const response = await fetch(`${API_BASE}/session/disconnect`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify({ reason }),
  });
  
  handleAuthError(response);
  const data = await response.json();
  if (!response.ok) {
    throw new Error(data.error || 'Disconnect failed');
  }
  return data;
}

// WebSocket connection manager
export class WebSocketManager {
  private ws: WebSocket | null = null;
  private messageHandlers: ((data: string) => void)[] = [];
  private errorHandlers: ((error: string) => void)[] = [];
  private statusHandlers: ((status: string) => void)[] = [];
  private disconnectHandlers: (() => void)[] = [];

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const wsUrl = `${protocol}//${window.location.host}/api/v1/session/stream`;
      
      this.ws = new WebSocket(wsUrl);
      
      this.ws.onopen = () => {
        resolve();
      };
      
      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        reject(new Error('WebSocket connection failed'));
      };
      
      this.ws.onmessage = (event) => {
        try {
          const message: WSMessage = JSON.parse(event.data);
          this.handleMessage(message);
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error);
        }
      };
      
      this.ws.onclose = () => {
        this.disconnectHandlers.forEach(handler => handler());
        this.ws = null;
      };
    });
  }

  private handleMessage(message: WSMessage): void {
    switch (message.type) {
      case 'data':
        if (message.data) {
          this.messageHandlers.forEach(handler => handler(message.data!));
        }
        break;
      case 'error':
        if (message.error) {
          this.errorHandlers.forEach(handler => handler(message.error!));
        }
        break;
      case 'status':
        if (message.status) {
          this.statusHandlers.forEach(handler => handler(message.status!));
        }
        break;
      case 'disconnect':
        this.disconnectHandlers.forEach(handler => handler());
        break;
    }
  }

  sendCommand(command: string): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        type: 'data',
        data: command,
      }));
    }
  }

  sendConnect(host: string, port: number): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        type: 'connect',
        host,
        port,
      }));
    }
  }

  sendDisconnect(): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        type: 'disconnect',
      }));
    }
  }

  onMessage(handler: (data: string) => void): void {
    this.messageHandlers.push(handler);
  }
  
  offMessage(handler: (data: string) => void): void {
    const index = this.messageHandlers.indexOf(handler);
    if (index > -1) {
      this.messageHandlers.splice(index, 1);
    }
  }

  onError(handler: (error: string) => void): void {
    this.errorHandlers.push(handler);
  }
  
  offError(handler: (error: string) => void): void {
    const index = this.errorHandlers.indexOf(handler);
    if (index > -1) {
      this.errorHandlers.splice(index, 1);
    }
  }

  onStatus(handler: (status: string) => void): void {
    this.statusHandlers.push(handler);
  }
  
  offStatus(handler: (status: string) => void): void {
    const index = this.statusHandlers.indexOf(handler);
    if (index > -1) {
      this.statusHandlers.splice(index, 1);
    }
  }

  onDisconnect(handler: () => void): void {
    this.disconnectHandlers.push(handler);
  }
  
  offDisconnect(handler: () => void): void {
    const index = this.disconnectHandlers.indexOf(handler);
    if (index > -1) {
      this.disconnectHandlers.splice(index, 1);
    }
  }

  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }
}

// Connection API endpoints

// Get all connections for the current user
export async function getConnections(): Promise<SavedConnection[]> {
  const response = await fetch(`${API_BASE}/connections`, {
    credentials: 'include',
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to get connections');
  }
  return await response.json();
}

// Get recent connections (last 5 connected)
export async function getRecentConnections(): Promise<SavedConnection[]> {
  const response = await fetch(`${API_BASE}/connections/recent`, {
    credentials: 'include',
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to get recent connections');
  }
  return await response.json();
}

// Get a single connection by ID
export async function getConnection(id: string): Promise<SavedConnection> {
  const response = await fetch(`${API_BASE}/connections/${id}`, {
    credentials: 'include',
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to get connection');
  }
  return await response.json();
}

// Create a new connection
export async function createConnection(request: CreateConnectionRequest): Promise<SavedConnection> {
  const response = await fetch(`${API_BASE}/connections`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify(request),
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to create connection');
  }
  return await response.json();
}

// Update an existing connection
export async function updateConnection(id: string, request: UpdateConnectionRequest): Promise<SavedConnection> {
  const response = await fetch(`${API_BASE}/connections/${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify(request),
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to update connection');
  }
  return await response.json();
}

// Delete a connection
export async function deleteConnection(id: string): Promise<void> {
  const response = await fetch(`${API_BASE}/connections/${id}`, {
    method: 'DELETE',
    credentials: 'include',
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to delete connection');
  }
}

// Get credential status for a connection
export async function getCredentialStatus(connectionId: string): Promise<CredentialStatus> {
  const response = await fetch(`${API_BASE}/connections/${connectionId}/credentials/status`, {
    credentials: 'include',
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to get credential status');
  }
  return await response.json();
}

// PR01PH08: Get credentials for automation (returns password only if auto_login enabled)
export async function getAutomationCredentials(connectionId: string): Promise<AutomationCredentials> {
  const response = await fetch(`${API_BASE}/connections/${connectionId}/credentials/auto`, {
    credentials: 'include',
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to get automation credentials');
  }
  return await response.json();
}

// Set credentials for a connection
export async function setCredentials(connectionId: string, request: SetCredentialsRequest): Promise<void> {
  const response = await fetch(`${API_BASE}/connections/${connectionId}/credentials`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify(request),
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to set credentials');
  }
}

// Update credentials for a connection
export async function updateCredentials(connectionId: string, request: SetCredentialsRequest): Promise<void> {
  const response = await fetch(`${API_BASE}/connections/${connectionId}/credentials`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify(request),
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to update credentials');
  }
}

// Delete credentials for a connection
export async function deleteCredentials(connectionId: string): Promise<void> {
  const response = await fetch(`${API_BASE}/connections/${connectionId}/credentials`, {
    method: 'DELETE',
    credentials: 'include',
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to delete credentials');
  }
}

// Connect to a saved connection (with optional credentials)
export async function connectToSavedConnection(connectionId: string): Promise<ConnectResponse> {
  const response = await fetch(`${API_BASE}/connections/${connectionId}/connect`, {
    method: 'POST',
    credentials: 'include',
  });
  handleAuthError(response);
  const data = await response.json();
  if (!response.ok) {
    throw new Error(data.error || 'Failed to connect');
  }
  return data;
}

// Profile API endpoints (SP04)

// Get a profile by ID
export async function getProfile(profileId: string): Promise<Profile> {
  const response = await fetch(`${API_BASE}/profiles/${profileId}`, {
    credentials: 'include',
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to get profile');
  }
  return await response.json();
}

// Get a profile by connection ID
export async function getProfileByConnection(connectionId: string): Promise<Profile> {
  const response = await fetch(`${API_BASE}/connections/${connectionId}/profile`, {
    credentials: 'include',
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to get profile');
  }
  return await response.json();
}

// Update a profile
export async function updateProfile(profileId: string, request: UpdateProfileRequest): Promise<Profile> {
  const response = await fetch(`${API_BASE}/profiles/${profileId}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify(request),
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to update profile');
  }
  return await response.json();
}

// ============================================
// Automation API endpoints (SP05)
// ============================================

// Get aliases for a connection
export async function getAliases(connectionId: string): Promise<AliasesResponse> {
  const response = await fetch(`${API_BASE}/profiles/${connectionId}/aliases`, {
    credentials: 'include',
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to get aliases');
  }
  return await response.json();
}

// Update aliases for a connection
export async function putAliases(connectionId: string, items: Alias[]): Promise<AliasesResponse> {
  const response = await fetch(`${API_BASE}/profiles/${connectionId}/aliases`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify({ items }),
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to update aliases');
  }
  return await response.json();
}

// Get triggers for a connection
export async function getTriggers(connectionId: string): Promise<TriggersResponse> {
  const response = await fetch(`${API_BASE}/profiles/${connectionId}/triggers`, {
    credentials: 'include',
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to get triggers');
  }
  return await response.json();
}

// Update triggers for a connection
export async function putTriggers(connectionId: string, items: Trigger[]): Promise<TriggersResponse> {
  const response = await fetch(`${API_BASE}/profiles/${connectionId}/triggers`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify({ items }),
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to update triggers');
  }
  return await response.json();
}

// Get environment variables for a connection
export async function getEnvironment(connectionId: string): Promise<VariablesResponse> {
  const response = await fetch(`${API_BASE}/profiles/${connectionId}/environment`, {
    credentials: 'include',
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to get environment');
  }
  return await response.json();
}

// Update environment variables for a connection
export async function putEnvironment(connectionId: string, items: Variable[]): Promise<VariablesResponse> {
  const response = await fetch(`${API_BASE}/profiles/${connectionId}/environment`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify({ items }),
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to update environment');
  }
  return await response.json();
}

// Get timers for a connection
export async function getTimers(connectionId: string): Promise<TimersResponse> {
  const response = await fetch(`${API_BASE}/profiles/${connectionId}/timers`, {
    credentials: 'include',
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to get timers');
  }
  return await response.json();
}

// Update timers for a connection
export async function putTimers(connectionId: string, items: Timer[]): Promise<TimersResponse> {
  const response = await fetch(`${API_BASE}/profiles/${connectionId}/timers`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify({ items }),
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to update timers');
  }
  return await response.json();
}

// ============================================
// Help API endpoints (SP06)
// ============================================

// Get all help sections (summaries)
export async function getHelpSections(): Promise<HelpSummary[]> {
  const response = await fetch(`${API_BASE}/help`, {
    credentials: 'include',
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to get help sections');
  }
  return await response.json();
}

// Get a specific help section by slug
export async function getHelpSection(slug: string): Promise<HelpSection> {
  const response = await fetch(`${API_BASE}/help/${slug}`, {
    credentials: 'include',
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to get help section');
  }
  return await response.json();
}
