import { User, SessionStatus, ConnectRequest, ConnectResponse, DisconnectResponse, WSMessage } from '../types';

const API_BASE = '/api/v1';

// Helper to handle auth errors and redirect to login
function handleAuthError(response: Response): void {
  if (response.status === 401) {
    window.location.href = '/login';
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
    method: 'POST',
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
        console.log('WebSocket connected');
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
      
      this.ws.onclose = (event) => {
        console.log('WebSocket closed:', event.code, event.reason);
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

  onError(handler: (error: string) => void): void {
    this.errorHandlers.push(handler);
  }

  onStatus(handler: (status: string) => void): void {
    this.statusHandlers.push(handler);
  }

  onDisconnect(handler: () => void): void {
    this.disconnectHandlers.push(handler);
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
