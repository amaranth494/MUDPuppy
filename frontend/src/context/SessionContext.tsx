import { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';
import { ConnectionState, User } from '../types';
import { 
  checkAuth, 
  getSessionStatus, 
  connectToMud, 
  disconnectFromMud,
  WebSocketManager 
} from '../services/api';
import { mapBackendError } from '../types';

interface SessionContextType {
  user: User | null;
  connectionState: ConnectionState;
  error: string | null;
  isLoading: boolean;
  host?: string;
  port?: number;
  connect: (host: string, port: number) => Promise<void>;
  disconnect: () => Promise<void>;
  refreshStatus: () => Promise<void>;
  wsManager: WebSocketManager | null;
}

const SessionContext = createContext<SessionContextType | null>(null);

export function useSession(): SessionContextType {
  const context = useContext(SessionContext);
  if (!context) {
    throw new Error('useSession must be used within a SessionProvider');
  }
  return context;
}

interface SessionProviderProps {
  children: ReactNode;
}

export function SessionProvider({ children }: SessionProviderProps): JSX.Element {
  const [user, setUser] = useState<User | null>(null);
  const [connectionState, setConnectionState] = useState<ConnectionState>('disconnected');
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [host, setHost] = useState<string>('');
  const [port, setPort] = useState<number>(23);
  const [wsManager, setWsManager] = useState<WebSocketManager | null>(null);

  // Check authentication on mount
  useEffect(() => {
    async function init() {
      const currentUser = await checkAuth();
      setUser(currentUser);
      if (currentUser) {
        await refreshStatus();
      }
      setIsLoading(false);
    }
    init();
  }, []);

  const refreshStatus = useCallback(async () => {
    try {
      const status = await getSessionStatus();
      setConnectionState(status.state);
      setHost(status.host || '');
      setPort(status.port || 23);
      if (status.last_error) {
        setError(mapBackendError(status.last_error));
      } else {
        setError(null);
      }
    } catch (err) {
      console.error('Failed to get session status:', err);
    }
  }, []);

  const connect = useCallback(async (mudHost: string, mudPort: number) => {
    setError(null);
    setConnectionState('connecting');
    
    try {
      // Call REST API to initiate connection
      await connectToMud({ host: mudHost, port: mudPort });
      
      // Connect WebSocket for streaming
      const manager = new WebSocketManager();
      
      manager.onMessage((_data) => {
        // This is handled by the Terminal component
      });
      
      manager.onError((err) => {
        setError(mapBackendError(err));
        setConnectionState('error');
      });
      
      manager.onStatus((status) => {
        if (status === 'connected') {
          setConnectionState('connected');
          setHost(mudHost);
          setPort(mudPort);
        }
      });
      
      manager.onDisconnect(() => {
        setConnectionState('disconnected');
        refreshStatus();
      });
      
      await manager.connect();
      setWsManager(manager);
      
      // Send connect message via WebSocket
      manager.sendConnect(mudHost, mudPort);
      
      // Verify connection status
      await refreshStatus();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Connection failed';
      setError(mapBackendError(errorMessage));
      setConnectionState('error');
    }
  }, [refreshStatus]);

  const disconnect = useCallback(async () => {
    setError(null);
    
    try {
      // Close WebSocket first
      if (wsManager) {
        wsManager.disconnect();
        setWsManager(null);
      }
      
      // Call REST API to disconnect
      await disconnectFromMud('user');
      
      setConnectionState('disconnected');
      setHost('');
      setPort(23);
      await refreshStatus();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Disconnect failed';
      setError(mapBackendError(errorMessage));
    }
  }, [wsManager, refreshStatus]);

  return (
    <SessionContext.Provider
      value={{
        user,
        connectionState,
        error,
        isLoading,
        host,
        port,
        connect,
        disconnect,
        refreshStatus,
        wsManager,
      }}
    >
      {children}
    </SessionContext.Provider>
  );
}
