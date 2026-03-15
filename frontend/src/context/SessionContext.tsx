import { createContext, useContext, useState, useEffect, useCallback, useRef, ReactNode } from 'react';
import { ConnectionState, User, Profile, Timer, TimersResponse } from '../types';
import { 
  checkAuth, 
  getSessionStatus, 
  connectToMud, 
  disconnectFromMud,
  WebSocketManager,
  getProfileByConnection,
  getAliases,
  getTriggers,
  getEnvironment,
  getTimers,
  putTimers
} from '../services/api';
import { mapBackendError } from '../types';
import { normalizeKeybindings } from '../services/keybindings';
import { getAutomationEngine, AutomationEngine } from '../services/automation';
import { VariableValue } from '../services/automation/evaluator';
import { SavedTimer } from '../services/automation/timer';

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
  automationDisabled: boolean;
  resumeAutomation: () => void;
  disableAutomation: () => void;
  enableAutomation: () => void;
  // PR01PH06: Variable refresh trigger for sync
  variablesRefreshTrigger: number;
  triggerVariablesRefresh: () => void;
  // Reconnection modal state (MVP)
  hasPendingReconnect: boolean;
  pendingReconnectData: { host: string; port: number; connectionId?: string } | null;
  clearPendingReconnect: () => void;
  forceReconnect: () => Promise<void>;
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
  const [automationDisabled, setAutomationDisabled] = useState(false);
  
  // PR01PH06: Variable refresh trigger for sync
  const [variablesRefreshTrigger, setVariablesRefreshTrigger] = useState(0);
  const triggerVariablesRefresh = useCallback(() => {
    setVariablesRefreshTrigger(prev => prev + 1);
  }, []);
  
  // Reconnection modal state (MVP) - tracks when user has existing session
  const [hasPendingReconnect, setHasPendingReconnect] = useState(false);
  const [pendingReconnectData, setPendingReconnectData] = useState<{ host: string; port: number; connectionId?: string } | null>(null);
  
  // Clear pending reconnect state
  const clearPendingReconnect = useCallback(() => {
    setHasPendingReconnect(false);
    setPendingReconnectData(null);
  }, []);
  
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

  // Track connection sessions to prevent stale updates
  // This is incremented on each connect, allowing us to ignore stale profile updates
  const connectionSessionRef = useRef(0);

  // Disable automation entirely (SP06PH07)
  // This sets a flag that prevents automation from processing and persists to the server
  const disableAutomation = useCallback(async () => {
    console.log('[Automation] disableAutomation called');
    // Capture the current session ID - ignore if session changed
    const sessionId = connectionSessionRef.current;
    
    // Don't persist if we don't have a valid connection or profile
    if (!currentConnectionId || !profile) {
      console.log('[Automation] disableAutomation: No connection or profile, disabling');
      setAutomationDisabled(true);
      setAutomationError(null);
      return;
    }
    
    setAutomationDisabled(true);
    setAutomationError(null);
    
    // Persist to server via profile settings - but check session first
    if (connectionSessionRef.current !== sessionId) {
      console.log('disableAutomation: session changed, skipping profile update');
      return;
    }
    
    try {
      const { updateProfile } = await import('../services/api');
      // Use profile.id, NOT currentConnectionId!
      await updateProfile(profile.id, {
        settings: {
          ...profile.settings,
          automation_enabled: false,
        },
      });
    } catch (err) {
      // Profile might not exist - just log, don't fail
      console.error('Failed to persist automation disabled state:', err);
    }
  }, [currentConnectionId, profile]);

  // Re-enable automation after user disabled it (SP06PH07)
  // This persists to the server
  const enableAutomation = useCallback(async () => {
    console.log('enableAutomation called, session:', connectionSessionRef.current, 'currentConnectionId:', currentConnectionId, 'profile:', profile?.id);
    
    // Capture the current session ID - ignore if session changed
    const sessionId = connectionSessionRef.current;
    
    // Don't persist if we don't have a valid connection or profile
    if (!currentConnectionId || !profile) {
      setAutomationDisabled(false);
      return;
    }
    
    // Check if session changed - if so, skip the update
    if (connectionSessionRef.current !== sessionId) {
      console.log('enableAutomation: session changed, skipping profile update');
      setAutomationDisabled(false);
      return;
    }
    
    setAutomationDisabled(false);
    
    // Persist to server via profile settings
    try {
      const { updateProfile } = await import('../services/api');
      // Use profile.id, NOT currentConnectionId!
      await updateProfile(profile.id, {
        settings: {
          ...profile.settings,
          automation_enabled: true,
        },
      });
    } catch (err) {
      // Profile might not exist - just log, don't fail
      console.error('Failed to persist automation enabled state:', err);
    }
  }, [currentConnectionId, profile]);

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
    console.log('[Automation] connect() called - mudHost:', mudHost, 'mudPort:', mudPort, 'connectionId:', connectionId);
    
    // Increment session ID to invalidate any pending profile updates from previous sessions
    connectionSessionRef.current++;
    
    setError(null);
    setConnectionState('connecting');
    
    // SP04: Fetch profile BEFORE connecting WebSocket
    // This ensures keybindings are available immediately after connect
    if (connectionId) {
      try {
        console.log('[Automation] Fetching profile for connectionId:', connectionId);
        const fetchedProfile = await getProfileByConnection(connectionId);
        // Normalize keybindings to canonical format
        const normalizedProfile: Profile = {
          ...fetchedProfile,
          keybindings: normalizeKeybindings(fetchedProfile.keybindings || {}),
        };
        setProfile(normalizedProfile);
        setCurrentConnectionId(connectionId);
        console.log('[Automation] Profile loaded:', normalizedProfile?.id, 'automation_enabled:', normalizedProfile?.settings?.automation_enabled);
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Failed to load profile';
        setError(mapBackendError(errorMessage));
        setConnectionState('error');
        return; // Do NOT proceed with connection if profile fails
      }
    } else {
      // Quick connect without saved connection - use defaults
      console.log('[Automation] Quick connect - setting profile to null');
      setProfile(null);
      setCurrentConnectionId(null);
    }
    
    // SP05: Initialize automation engine after profile is loaded
    let engine: ReturnType<typeof getAutomationEngine> | null = null;
    try {
      engine = getAutomationEngine();
      
      // Fetch automation data if we have a connection ID
      // Fetch automation data if we have a connection ID
      let timers: TimersResponse | null = null;
      if (connectionId) {
        const [aliases, triggers, variables, fetchedTimers] = await Promise.all([
          getAliases(connectionId),
          getTriggers(connectionId),
          getEnvironment(connectionId),
          getTimers(connectionId),
        ]);
        timers = fetchedTimers;
        
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
        console.log('[Automation] Circuit breaker tripped:', reason);
        setAutomationError(reason);
      });
      
      // PR01PH06: Set up variable change callback for UI sync
      engine.setVariableChangeCallback((name: string, value: VariableValue) => {
        console.log('[Automation] Variable changed:', name, '=', value);
        triggerVariablesRefresh();
      });
      
      // PR01PH04: Set up timer persistence callbacks
      if (connectionId) {
        engine.setTimerCallbacks(
          async (timer: SavedTimer) => {
            // Save timer to profile
            try {
              const currentTimers = await getTimers(connectionId);
              // Check if timer already exists
              const existingIndex = currentTimers.items.findIndex(t => t.name === timer.name);
              let updatedTimers: Timer[];
              
              if (existingIndex >= 0) {
                // Update existing timer
                updatedTimers = [...currentTimers.items];
                updatedTimers[existingIndex] = {
                  ...updatedTimers[existingIndex],
                  name: timer.name,
                  duration: timer.duration,
                  repeat: timer.repeat,
                  commands: timer.commands.join('\n'), // Join array into string for API
                  enabled: true
                };
              } else {
                // Add new timer
                updatedTimers = [...currentTimers.items, {
                  id: crypto.randomUUID(),
                  name: timer.name,
                  duration: timer.duration,
                  repeat: timer.repeat,
                  commands: timer.commands.join('\n'), // Join array into string for API
                  enabled: true
                }];
              }
              
              await putTimers(connectionId, updatedTimers);
              console.log('[Automation] Timer saved:', timer.name);
            } catch (error) {
              console.error('[Automation] Failed to save timer:', error);
            }
          },
          async (timerName: string) => {
            // Delete timer from profile
            try {
              const currentTimers = await getTimers(connectionId);
              const updatedTimers = currentTimers.items.filter(t => t.name !== timerName);
              await putTimers(connectionId, updatedTimers);
              console.log('[Automation] Timer deleted:', timerName);
            } catch (error) {
              console.error('[Automation] Failed to delete timer:', error);
            }
          }
        );
        
        // Load timers from profile (must be AFTER setTimerCallbacks creates timerManager)
        if (timers && timers.items) {
          const savedTimers = timers.items.map((t: Timer) => ({
            name: t.name,
            duration: t.duration,
            repeat: t.repeat,
            commands: typeof t.commands === 'string' ? t.commands.split('\n').filter((c: string) => c.trim()) : t.commands,
          }));
          engine.loadTimers(savedTimers);
          console.log('[Automation] Timers loaded:', savedTimers.length);
        }
      }
      
      setAutomationEngine(engine);
      setAutomationError(null);
      
      // SP06PH07: Load automation_enabled from profile settings (persisted per connection)
      const isAutomationEnabled = profile?.settings?.automation_enabled ?? true;
      console.log('[Automation] Loading profile settings - automation_enabled:', isAutomationEnabled, 'profile:', profile?.id);
      if (!isAutomationEnabled) {
        console.log('[Automation] Disabled due to profile setting (automation_enabled = false)');
      }
      setAutomationDisabled(!isAutomationEnabled);
    } catch (err) {
      console.error('Failed to initialize automation:', err);
      // Don't fail connection, just disable automation
      setAutomationEngine(null);
      engine = null;
    }
    
    try {
      // Call REST API to initiate connection
      // Pass connection_id so backend can update last_connected_at for recent connections
      const connectRequest: { host: string; port: number; connection_id?: string } = { 
        host: mudHost, 
        port: mudPort 
      };
      if (connectionId) {
        connectRequest.connection_id = connectionId;
      }
      await connectToMud(connectRequest);
      
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
      if (engine) {
        engine.connect();
      }
      
      // Event-driven: verify connection status after connect action
      await refreshStatus();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Connection failed';
      
      // Check for existing active session - offer to reconnect (MVP)
      // Capture values for use in state since they're not in dependency array
      const currentHost = mudHost;
      const currentPort = mudPort;
      const currentConnectionId = connectionId;
      
      if (errorMessage === 'user already has active session') {
        setHasPendingReconnect(true);
        setPendingReconnectData({ host: currentHost, port: currentPort, connectionId: currentConnectionId });
        setError('You already have an active connection');
        setConnectionState('error');
        return;
      }
      
      setError(mapBackendError(errorMessage));
      setConnectionState('error');
      // Event-driven: refresh status on connect failure
      await refreshStatus();
    }
  }, [refreshStatus, automationEngine]);

  // Force reconnect - disconnect existing session then connect (MVP)
  const forceReconnect = useCallback(async () => {
    if (!pendingReconnectData) return;
    
    const { host: reconnectHost, port: reconnectPort, connectionId: reconnectConnectionId } = pendingReconnectData;
    clearPendingReconnect();
    
    // Disconnect first
    try {
      await disconnectFromMud('Force reconnect');
    } catch (e) {
      // Ignore disconnect errors - proceed with connect anyway
    }
    
    // Then connect
    await connect(reconnectHost, reconnectPort, reconnectConnectionId);
  }, [pendingReconnectData, clearPendingReconnect, connect]);

  const disconnect = useCallback(async () => {
    // Always clear state first - even if API call fails, we want to show disconnected
    // This ensures a clean state for reconnection
    setError(null);
    setConnectionState('disconnected');
    setHost('');
    setPort(23);
    setProfile(null);
    setCurrentConnectionId(null);
    
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
      
      // Event-driven: refresh status after disconnect action completes
      await refreshStatus();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Disconnect failed';
      console.error('Disconnect error:', errorMessage);
      // State already cleared above, just refresh status
      // Note: Don't set error state here - we want to show disconnected
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
        automationDisabled,
        resumeAutomation,
        disableAutomation,
        enableAutomation,
        // PR01PH06: Variable refresh for sync
        variablesRefreshTrigger,
        triggerVariablesRefresh,
        hasPendingReconnect,
        pendingReconnectData,
        clearPendingReconnect,
        forceReconnect,
      }}
    >
      {children}
    </SessionContext.Provider>
  );
}
