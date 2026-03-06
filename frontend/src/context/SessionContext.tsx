import { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';
import { ConnectionState, User, Profile } from '../types';
import { 
  checkAuth, 
  getSessionStatus, 
  connectToMud, 
  disconnectFromMud,
  WebSocketManager,
  getProfileByConnection,
  getAliases,
  getTriggers,
  getEnvironment
} from '../services/api';
import { mapBackendError } from '../types';
import { normalizeKeybindings } from '../services/keybindings';
import { getAutomationEngine, AutomationEngine } from '../services/automation';

interface SessionContextType {
  user: User | null;
  connectionState: ConnectionState;
  error: string | null;
  isLoading: boolean;
  host?: string;
  port?: number;
  connect: (host: string, port: number, connectionId?: string) => Promise<void>;
  disconnect: () => Promise<void>;
  refreshStatus: () => Promise<void>;
  wsManager: WebSocketManager | null;
  // Input lock state for modal (SP03PH03)
  isInputLocked: boolean;
  setInputLocked: (locked: boolean) => void;
  // Profile state (SP04)
  profile: Profile | null;
  currentConnectionId: string | null;
  updateProfile: (profile: Profile) => void;
  // Automation state (SP05)
  automationEngine: AutomationEngine | null;
  automationError: string | null;
  resumeAutomation: () => void;
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
  
  // Input lock state for modal (SP03PH03)
  // When true, terminal input is locked and commands cannot be sent to MUD
  const [isInputLocked, setInputLocked] = useState(false);

  // Profile state (SP04) - fetched on connect
  const [profile, setProfile] = useState<Profile | null>(null);
  const [currentConnectionId, setCurrentConnectionId] = useState<string | null>(null);
  
  // Automation state (SP05)
  const [automationEngine, setAutomationEngine] = useState<AutomationEngine | null>(null);
  const [automationError, setAutomationError] = useState<string | null>(null);
  
  // Update profile in real-time while connected (SP04PH07)
  const updateProfile = useCallback((updatedProfile: Profile) => {
    setProfile(updatedProfile);
  }, []);
  
  // Resume automation after circuit breaker trip (SP05PH03T13)
  const resumeAutomation = useCallback(() => {
    if (automationEngine) {
      automationEngine.resume();
      setAutomationError(null);
    }
  }, [automationEngine]);

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

  const connect = useCallback(async (mudHost: string, mudPort: number, connectionId?: string) => {
    setError(null);
    setConnectionState('connecting');
    
    // SP04: Fetch profile BEFORE connecting WebSocket
    // This ensures keybindings are available immediately after connect
    if (connectionId) {
      try {
        const fetchedProfile = await getProfileByConnection(connectionId);
        // Normalize keybindings to canonical format
        const normalizedProfile: Profile = {
          ...fetchedProfile,
          keybindings: normalizeKeybindings(fetchedProfile.keybindings || {}),
        };
        setProfile(normalizedProfile);
        setCurrentConnectionId(connectionId);
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Failed to load profile';
        setError(mapBackendError(errorMessage));
        setConnectionState('error');
        return; // Do NOT proceed with connection if profile fails
      }
    } else {
      // Quick connect without saved connection - use defaults
      setProfile(null);
      setCurrentConnectionId(null);
    }
    
    // SP05: Initialize automation engine after profile is loaded
    try {
      const engine = getAutomationEngine();
      
      // Fetch automation data if we have a connection ID
      if (connectionId) {
        const [aliases, triggers, variables] = await Promise.all([
          getAliases(connectionId),
          getTriggers(connectionId),
          getEnvironment(connectionId),
        ]);
        
        engine.configure({
          aliases,
          triggers,
          variables,
          connectionId,
        });
      } else {
        // Quick connect - use empty automation
        engine.configure({
          aliases: { items: [] },
          triggers: { items: [] },
          variables: { items: [] },
          connectionId: '',
        });
      }
      
      // Set up circuit breaker callback
      engine.setCircuitBreakerCallback((reason: string) => {
        setAutomationError(reason);
      });
      
      setAutomationEngine(engine);
      setAutomationError(null);
    } catch (err) {
      console.error('Failed to initialize automation:', err);
      // Don't fail connection, just disable automation
      setAutomationEngine(null);
    }
    
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
        // Event-driven: refresh status on WS error
        refreshStatus();
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
        // Event-driven: refresh status on WS disconnect
        refreshStatus();
      });
      
      await manager.connect();
      setWsManager(manager);
      
      // Send connect message via WebSocket
      manager.sendConnect(mudHost, mudPort);
      
      // SP05: Mark automation engine as connected
      if (automationEngine) {
        automationEngine.connect();
      }
      
      // Event-driven: verify connection status after connect action
      await refreshStatus();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Connection failed';
      setError(mapBackendError(errorMessage));
      setConnectionState('error');
      // Event-driven: refresh status on connect failure
      await refreshStatus();
    }
  }, [refreshStatus, automationEngine]);

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
      
      // SP05: Disconnect automation engine
      if (automationEngine) {
        automationEngine.disconnect();
      }
      
      setConnectionState('disconnected');
      setHost('');
      setPort(23);
      // Clear profile on disconnect (SP04)
      setProfile(null);
      setCurrentConnectionId(null);
      // Event-driven: refresh status after disconnect action completes
      await refreshStatus();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Disconnect failed';
      setError(mapBackendError(errorMessage));
      // Event-driven: refresh status on disconnect failure
      await refreshStatus();
    }
  }, [wsManager, refreshStatus, automationEngine]);

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
        isInputLocked,
        setInputLocked,
        profile,
        currentConnectionId,
        updateProfile,
        automationEngine,
        automationError,
        resumeAutomation,
      }}
    >
      {children}
    </SessionContext.Provider>
  );
}
