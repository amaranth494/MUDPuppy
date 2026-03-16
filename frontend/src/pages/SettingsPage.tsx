import { useState, useEffect, useCallback } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { getConnections, getProfileByConnection, updateProfile, getAliases, putAliases, getTriggers, putTriggers, getEnvironment, putEnvironment, getTimers, putTimers } from '../services/api';
import { SavedConnection, ProfileSettings, UpdateProfileRequest, Alias, Trigger, Variable, Timer } from '../types';
import { normalizeKeybindings, eventToCanonicalKey, isValidKeybindingFormat, isValidCommand, canonicalizeKeybinding, isModifierOnly } from '../services/keybindings';
import { useSession } from '../context/SessionContext';
import Modal from '../components/Modal';
import { parser, validateSyntax, ParseError } from '../services/automation/parser';

// Section types
type SettingsSection = 'general' | 'keybindings' | 'aliases' | 'triggers' | 'timers' | 'environment';

// Section definition
interface Section {
  id: SettingsSection;
  label: string;
  icon: string;
}

const SECTIONS: Section[] = [
  { id: 'general', label: 'General', icon: '⚙' },
  { id: 'keybindings', label: 'Key Bindings', icon: '⌨' },
  { id: 'aliases', label: 'Aliases', icon: '⚡' },
  { id: 'triggers', label: 'Triggers', icon: '⚓' },
  { id: 'timers', label: 'Timers', icon: '⏱' },
  { id: 'environment', label: 'Environment', icon: '📦' },
];

export default function SettingsPage() {
  // Use useLocation for URL detection instead of useParams to ensure proper re-renders
  const location = useLocation();
  const navigate = useNavigate();
  const { updateProfile: updateSessionProfile, connectionState, currentConnectionId, automationEngine, automationError, automationDisabled, disableAutomation, enableAutomation, resumeAutomation, variablesRefreshTrigger } = useSession();
  
  // Sync section from URL to state to ensure re-render on navigation
  const [activeSection, setActiveSection] = useState<SettingsSection>('general');
  
  // Update activeSection when URL section changes (use location.pathname to trigger re-render)
  useEffect(() => {
    const pathParts = location.pathname.split('/');
    const sectionFromUrl = pathParts[2] as SettingsSection;
    const newSection = sectionFromUrl || 'general';
    setActiveSection(newSection);
  }, [location.pathname]);
  
  // Connection and profile state - load from active connection or prompt to select
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [connectionId, setConnectionId] = useState<string | null>(currentConnectionId);
  const [connectionName, setConnectionName] = useState<string>('');
  
  // Connection selector modal state
  const [showConnectionSelector, setShowConnectionSelector] = useState(false);
  const [savedConnections, setSavedConnections] = useState<SavedConnection[]>([]);
  const [isLoadingConnections, setIsLoadingConnections] = useState(false);
  
  // Form state - Keybindings
  const [keybindings, setKeybindings] = useState<Record<string, string>>({});
  const [newKey, setNewKey] = useState('');
  const [newCommand, setNewCommand] = useState('');
  const [isCapturingKey, setIsCapturingKey] = useState(false);
  
  // Form state - Aliases
  const [aliases, setAliases] = useState<Alias[]>([]);
  const [isSavingAliases, setIsSavingAliases] = useState(false);
  
  // Syntax validation state - PR01PH06
  const [aliasSyntaxErrors, setAliasSyntaxErrors] = useState<Record<string, string>>({});
  const [triggerSyntaxErrors, setTriggerSyntaxErrors] = useState<Record<string, string>>({});
  const [validationBanner, setValidationBanner] = useState<string | null>(null);
  
  // Form state - Triggers
  const [triggers, setTriggers] = useState<Trigger[]>([]);
  const [isSavingTriggers, setIsSavingTriggers] = useState(false);
  
  // Form state - Variables
  const [variables, setVariables] = useState<Variable[]>([]);
  const [isSavingVariables, setIsSavingVariables] = useState(false);

  // Form state - Timers
  const [timers, setTimers] = useState<Timer[]>([]);
  const [isSavingTimers, setIsSavingTimers] = useState(false);
  
  // Form state - Settings
  const [settings, setSettings] = useState<ProfileSettings>({
    scrollback_limit: 1000,
    echo_input: false,
    timestamp_output: false,
    word_wrap: true,
  });
  const [isSavingProfile, setIsSavingProfile] = useState(false);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Load data when connectionId changes
  const loadData = useCallback(async () => {
    if (!connectionId) return;
    
    setIsLoading(true);
    setError(null);
    try {
      // Load profile
      const profileData = await getProfileByConnection(connectionId);
      setKeybindings(normalizeKeybindings(profileData.keybindings || {}));
      setSettings(profileData.settings || {
        scrollback_limit: 1000,
        echo_input: false,
        timestamp_output: false,
        word_wrap: true,
      });
      
      // Load automation data
      const aliasesData = await getAliases(connectionId);
      setAliases(aliasesData.items || []);
      
      const triggersData = await getTriggers(connectionId);
      setTriggers(triggersData.items || []);
      
      const variablesData = await getEnvironment(connectionId);
      setVariables(variablesData.items || []);
      
      const timersData = await getTimers(connectionId);
      setTimers(timersData.items || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load settings');
    } finally {
      setIsLoading(false);
    }
  }, [connectionId]);

  // Load saved connections for the selector modal
  const loadSavedConnections = async () => {
    setIsLoadingConnections(true);
    try {
      const connections = await getConnections();
      setSavedConnections(connections);
    } catch (err) {
      console.error('Failed to load connections:', err);
    } finally {
      setIsLoadingConnections(false);
    }
  };

  // Show connection selector when no active connection
  const handleShowConnectionSelector = async () => {
    await loadSavedConnections();
    setShowConnectionSelector(true);
  };

  // Handle selecting a connection
  const handleSelectConnection = (connId: string, connName?: string) => {
    setConnectionId(connId);
    // Find connection name from saved connections if not provided
    if (connName) {
      setConnectionName(connName);
    } else {
      const conn = savedConnections.find(c => c.id === connId);
      if (conn) {
        setConnectionName(conn.name);
      }
    }
    setShowConnectionSelector(false);
  };

  useEffect(() => {
    // If we have a current connection and it's in a connected state, use it
    if (currentConnectionId && connectionState === 'connected') {
      setConnectionId(currentConnectionId);
      // Try to get connection name from saved connections
      getConnections().then(connections => {
        const conn = connections.find(c => c.id === currentConnectionId);
        if (conn) {
          setConnectionName(conn.name);
        }
      });
    } else if (!connectionId) {
      // No connection - show the selector
      handleShowConnectionSelector();
    }
  }, [currentConnectionId, connectionState]);

  useEffect(() => {
    if (connectionId) {
      loadData();
    }
  }, [connectionId, loadData]);

  // Reload data when variables change in the runtime (after #SET)
  useEffect(() => {
    if (connectionId && variablesRefreshTrigger > 0) {
      loadData();
    }
  }, [variablesRefreshTrigger, connectionId, loadData]);

  // Handle key capture
  useEffect(() => {
    if (!isCapturingKey) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      e.preventDefault();
      e.stopPropagation();
      
      const key = eventToCanonicalKey(e);
      
      if (!key || isModifierOnly(key)) {
        return;
      }
      
      if (key === 'escape') {
        setIsCapturingKey(false);
        return;
      }
      
      setNewKey(key);
      setIsCapturingKey(false);
    };

    window.addEventListener('keydown', handleKeyDown, true);
    
    return () => {
      window.removeEventListener('keydown', handleKeyDown, true);
    };
  }, [isCapturingKey]);

  // Save keybindings and settings
  const handleSaveProfile = async () => {
    if (!connectionId) return;

    setIsSavingProfile(true);
    setError(null);
    setSuccessMessage(null);

    try {
      const profile = await getProfileByConnection(connectionId);
      const updateRequest: UpdateProfileRequest = {
        keybindings,
        settings,
      };

      await updateProfile(profile.id, updateRequest);
      
      if (connectionState === 'connected' && currentConnectionId === connectionId) {
        setSuccessMessage('Settings saved. Key bindings updated live.');
      } else {
        setSuccessMessage('Settings saved successfully');
      }
      
      // Update session context for real-time keybinding updates
      updateSessionProfile({
        ...profile,
        keybindings,
        settings,
      });

      await loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save settings');
    } finally {
      setIsSavingProfile(false);
    }
  };

  // Save aliases
  const handleSaveAliases = async () => {
    if (!connectionId) return;
    
    // PR01PH06: Validate all aliases before saving
    const allErrors: Record<string, string> = {};
    for (const alias of aliases) {
      if (alias.replacement && alias.replacement.includes('#')) {
        const errors = validateAutomationScript(alias.replacement);
        if (errors.length > 0) {
          allErrors[alias.id] = errors.map(e => `Line ${e.line}: ${e.message}`).join('; ');
        }
      }
    }
    
    // If there are errors, show banner and block save
    if (Object.keys(allErrors).length > 0) {
      const errorList = Object.entries(allErrors).map(([id, msg]) => {
        const alias = aliases.find(a => a.id === id);
        return `${alias?.pattern || id}: ${msg}`;
      }).join('\n');
      const bannerMessage = `Cannot save aliases due to syntax errors:\n${errorList}`;
      setValidationBanner(bannerMessage);
      console.error('[SettingsPage] Alias validation errors - save blocked:', allErrors);
      return;
    }
    
    setIsSavingAliases(true);
    setError(null);
    setSuccessMessage(null);
    setValidationBanner(null);
    
    try {
      await putAliases(connectionId, aliases);
      
      // Update automation engine immediately for active session
      if (automationEngine && connectionState === 'connected' && currentConnectionId === connectionId) {
        // Get current triggers and variables from the state (loaded via loadData)
        automationEngine.configure({
          aliases: { items: aliases },
          triggers: { items: triggers },
          variables: { items: variables },
          connectionId,
        });
        setSuccessMessage('Aliases saved and updated live.');
      } else {
        setSuccessMessage('Aliases saved successfully');
      }
      await loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save aliases');
    } finally {
      setIsSavingAliases(false);
    }
  };

  // Save triggers
  const handleSaveTriggers = async () => {
    if (!connectionId) return;
    
    // PR01PH06: Validate all triggers before saving
    const allErrors: Record<string, string> = {};
    for (const trigger of triggers) {
      if (trigger.action && trigger.action.includes('#')) {
        const errors = validateAutomationScript(trigger.action);
        if (errors.length > 0) {
          allErrors[trigger.id] = errors.map(e => `Line ${e.line}: ${e.message}`).join('; ');
        }
      }
    }
    
    // If there are errors, show banner and block save
    if (Object.keys(allErrors).length > 0) {
      const errorList = Object.entries(allErrors).map(([id, msg]) => {
        const trigger = triggers.find(t => t.id === id);
        return `${trigger?.match || id}: ${msg}`;
      }).join('\n');
      const bannerMessage = `Cannot save triggers due to syntax errors:\n${errorList}`;
      setValidationBanner(bannerMessage);
      console.error('[SettingsPage] Trigger validation errors - save blocked:', allErrors);
      return;
    }
    
    setIsSavingTriggers(true);
    setError(null);
    setSuccessMessage(null);
    setValidationBanner(null);
    
    try {
      await putTriggers(connectionId, triggers);
      
      // Update automation engine immediately for active session
      if (automationEngine && connectionState === 'connected' && currentConnectionId === connectionId) {
        automationEngine.configure({
          aliases: { items: aliases },
          triggers: { items: triggers },
          variables: { items: variables },
          connectionId,
        });
        setSuccessMessage('Triggers saved and updated live.');
      } else {
        setSuccessMessage('Triggers saved successfully');
      }
      await loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save triggers');
    } finally {
      setIsSavingTriggers(false);
    }
  };

  // Timer syntax errors
  const [timerSyntaxErrors, setTimerSyntaxErrors] = useState<Record<string, string>>({});

  // Add timer
  const handleAddTimer = () => {
    const newTimer: Timer = {
      id: crypto.randomUUID(),
      name: '',
      duration: 60000, // 1 minute default
      repeat: false,
      commands: '',
      enabled: true,
    };
    setTimers([...timers, newTimer]);
  };

  // Update timer
  const handleUpdateTimer = (id: string, updates: Partial<Timer>) => {
    setTimers(timers.map(t => t.id === id ? { ...t, ...updates } : t));
  };

  // Remove timer
  const handleRemoveTimer = (id: string) => {
    setTimers(timers.filter(t => t.id !== id));
    setTimerSyntaxErrors(prev => {
      const newErrors = { ...prev };
      delete newErrors[id];
      return newErrors;
    });
  };

  // Validate timer commands
  const validateAndUpdateTimer = (id: string, updates: Partial<Timer>) => {
    handleUpdateTimer(id, updates);
    if (updates.commands && updates.commands.includes('#')) {
      const errors = validateAutomationScript(updates.commands);
      if (errors.length > 0) {
        setTimerSyntaxErrors(prev => ({ 
          ...prev, 
          [id]: errors.map(e => `Line ${e.line}: ${e.message}`).join('; ') 
        }));
      } else {
        setTimerSyntaxErrors(prev => {
          const newErrors = { ...prev };
          delete newErrors[id];
          return newErrors;
        });
      }
    }
  };

  // Save timers
  const handleSaveTimers = async () => {
    if (!connectionId) return;
    
    // PR01PH06: Validate all timers before saving
    const allErrors: Record<string, string> = {};
    for (const timer of timers) {
      if (timer.commands && timer.commands.includes('#')) {
        const errors = validateAutomationScript(timer.commands);
        if (errors.length > 0) {
          allErrors[timer.id] = errors.map(e => `Line ${e.line}: ${e.message}`).join('; ');
        }
      }
    }
    
    // If there are errors, show banner and block save
    if (Object.keys(allErrors).length > 0) {
      const errorList = Object.entries(allErrors).map(([id, msg]) => {
        const timer = timers.find(t => t.id === id);
        return `${timer?.name || id}: ${msg}`;
      }).join('\n');
      const bannerMessage = `Cannot save timers due to syntax errors:\n${errorList}`;
      setValidationBanner(bannerMessage);
      console.error('[SettingsPage] Timer validation errors - save blocked:', allErrors);
      return;
    }
    
    setIsSavingTimers(true);
    setError(null);
    setSuccessMessage(null);
    setValidationBanner(null);
    
    try {
      await putTimers(connectionId, timers);
      setSuccessMessage('Timers saved successfully');
      setTimeout(() => setSuccessMessage(null), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save timers');
    } finally {
      setIsSavingTimers(false);
    }
  };

  // Save variables
  const handleSaveVariables = async () => {
    if (!connectionId) return;
    
    setIsSavingVariables(true);
    setError(null);
    setSuccessMessage(null);
    
    try {
      await putEnvironment(connectionId, variables);
      
      // Update automation engine immediately for active session
      if (automationEngine && connectionState === 'connected' && currentConnectionId === connectionId) {
        automationEngine.configure({
          aliases: { items: aliases },
          triggers: { items: triggers },
          variables: { items: variables },
          connectionId,
        });
        setSuccessMessage('Variables saved and updated live.');
      } else {
        setSuccessMessage('Variables saved successfully');
      }
      await loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save variables');
    } finally {
      setIsSavingVariables(false);
    }
  };

  // Add keybinding
  const handleAddKeybinding = () => {
    if (!newKey.trim() && !isCapturingKey) {
      setIsCapturingKey(true);
      return;
    }

    if (!newKey.trim() || !newCommand.trim()) {
      return;
    }

    if (!isValidKeybindingFormat(newKey.trim())) {
      setError('Invalid keybinding format');
      return;
    }

    if (!isValidCommand(newCommand.trim())) {
      setError('Invalid command');
      return;
    }

    if (Object.keys(keybindings).length >= 50) {
      setError('Maximum 50 keybindings allowed');
      return;
    }

    const canonicalKey = canonicalizeKeybinding(newKey.trim());
    if (!canonicalKey) {
      setError('Could not canonicalize keybinding');
      return;
    }

    setKeybindings(prev => ({
      ...prev,
      [canonicalKey]: newCommand.trim(),
    }));
    setNewKey('');
    setNewCommand('');
    setError(null);
  };

  // Remove keybinding
  const handleRemoveKeybinding = (key: string) => {
    const newBindings = { ...keybindings };
    delete newBindings[key];
    setKeybindings(newBindings);
  };

  // PR01PH06: Syntax validation helper - only validates if # syntax present
  const validateAutomationScript = (text: string): ParseError[] => {
    // Only validate if there's # syntax in the text
    if (!text || !text.includes('#')) {
      return [];
    }
    // Parse and validate the script
    const result = parser.parse(text);
    if (!result.success) {
      return result.errors;
    }
    // Also run syntax validation on tokens
    return validateSyntax(result.tokens);
  };

  // Validate alias replacement and update error state
  const validateAndUpdateAlias = (id: string, updates: Partial<Alias>) => {
    // Update the alias first
    setAliases(aliases.map(a => a.id === id ? { ...a, ...updates } : a));
    
    // If replacement was updated, validate it
    if (updates.replacement !== undefined) {
      const errors = validateAutomationScript(updates.replacement);
      if (errors.length > 0) {
        const errorMessage = errors.map(e => `Line ${e.line}: ${e.message}`).join('; ');
        setAliasSyntaxErrors(prev => ({ ...prev, [id]: errorMessage }));
        console.warn('[SettingsPage] Alias syntax validation errors:', errors);
      } else {
        // Clear error if no syntax errors
        setAliasSyntaxErrors(prev => {
          const newErrors = { ...prev };
          delete newErrors[id];
          return newErrors;
        });
      }
    }
  };

  // Validate trigger action and update error state
  const validateAndUpdateTrigger = (id: string, updates: Partial<Trigger>) => {
    // Update the trigger first
    setTriggers(triggers.map(t => t.id === id ? { ...t, ...updates } : t));
    
    // If action was updated, validate it
    if (updates.action !== undefined) {
      const errors = validateAutomationScript(updates.action);
      if (errors.length > 0) {
        const errorMessage = errors.map(e => `Line ${e.line}: ${e.message}`).join('; ');
        setTriggerSyntaxErrors(prev => ({ ...prev, [id]: errorMessage }));
        console.warn('[SettingsPage] Trigger syntax validation errors:', errors);
      } else {
        // Clear error if no syntax errors
        setTriggerSyntaxErrors(prev => {
          const newErrors = { ...prev };
          delete newErrors[id];
          return newErrors;
        });
      }
    }
  };

  // Add alias
  const handleAddAlias = () => {
    const newAlias: Alias = {
      id: crypto.randomUUID(),
      pattern: '',
      replacement: '',
      enabled: true,
    };
    setAliases([...aliases, newAlias]);
  };

  // Update alias
  const handleUpdateAlias = (id: string, updates: Partial<Alias>) => {
    setAliases(aliases.map(a => a.id === id ? { ...a, ...updates } : a));
  };

  // Remove alias
  const handleRemoveAlias = (id: string) => {
    setAliases(aliases.filter(a => a.id !== id));
  };

  // Move alias up (reorder)
  const handleMoveAliasUp = async (index: number) => {
    if (index <= 0) return;
    const newAliases = [...aliases];
    [newAliases[index - 1], newAliases[index]] = [newAliases[index], newAliases[index - 1]];
    setAliases(newAliases);
    // Persist immediately to server
    if (connectionId) {
      try {
        await putAliases(connectionId, newAliases);
      } catch (err) {
        console.error('Failed to persist alias reorder:', err);
      }
    }
  };

  // Move alias down (reorder)
  const handleMoveAliasDown = async (index: number) => {
    if (index >= aliases.length - 1) return;
    const newAliases = [...aliases];
    [newAliases[index], newAliases[index + 1]] = [newAliases[index + 1], newAliases[index]];
    setAliases(newAliases);
    // Persist immediately to server
    if (connectionId) {
      try {
        await putAliases(connectionId, newAliases);
      } catch (err) {
        console.error('Failed to persist alias reorder:', err);
      }
    }
  };

  // Add trigger
  const handleAddTrigger = () => {
    const newTrigger: Trigger = {
      id: crypto.randomUUID(),
      match: '',
      type: 'contains',
      action: '',
      cooldown_ms: 2000,
      enabled: true,
    };
    setTriggers([...triggers, newTrigger]);
  };

  // Update trigger
  const handleUpdateTrigger = (id: string, updates: Partial<Trigger>) => {
    setTriggers(triggers.map(t => t.id === id ? { ...t, ...updates } : t));
  };

  // Remove trigger
  const handleRemoveTrigger = (id: string) => {
    setTriggers(triggers.filter(t => t.id !== id));
  };

  // Move trigger up (reorder)
  const handleMoveTriggerUp = async (index: number) => {
    if (index <= 0) return;
    const newTriggers = [...triggers];
    [newTriggers[index - 1], newTriggers[index]] = [newTriggers[index], newTriggers[index - 1]];
    setTriggers(newTriggers);
    // Persist immediately to server
    if (connectionId) {
      try {
        await putTriggers(connectionId, newTriggers);
      } catch (err) {
        console.error('Failed to persist trigger reorder:', err);
      }
    }
  };

  // Move trigger down (reorder)
  const handleMoveTriggerDown = async (index: number) => {
    if (index >= triggers.length - 1) return;
    const newTriggers = [...triggers];
    [newTriggers[index], newTriggers[index + 1]] = [newTriggers[index + 1], newTriggers[index]];
    setTriggers(newTriggers);
    // Persist immediately to server
    if (connectionId) {
      try {
        await putTriggers(connectionId, newTriggers);
      } catch (err) {
        console.error('Failed to persist trigger reorder:', err);
      }
    }
  };

  // Add variable
  const handleAddVariable = () => {
    const newVariable: Variable = {
      id: crypto.randomUUID(),
      name: '',
      value: '',
    };
    setVariables([...variables, newVariable]);
  };

  // Update variable
  const handleUpdateVariable = (id: string, updates: Partial<Variable>) => {
    setVariables(variables.map(v => v.id === id ? { ...v, ...updates } : v));
  };

  // Remove variable
  const handleRemoveVariable = (id: string) => {
    setVariables(variables.filter(v => v.id !== id));
  };

  // Handle settings change
  const handleSettingChange = (key: keyof ProfileSettings, value: number | boolean) => {
    setSettings(prev => ({ ...prev, [key]: value }));
  };

  // Handle scrollback change
  const handleScrollbackChange = (value: string) => {
    const num = parseInt(value) || 100;
    const clamped = Math.max(100, Math.min(10000, num));
    setSettings(prev => ({ ...prev, scrollback_limit: clamped }));
  };

  // Handle section navigation
  const handleSectionChange = (sectionId: SettingsSection) => {
    navigate(`/settings/${sectionId}`);
  };

  // No connection selected - show connection selector modal
  const renderConnectionSelectorModal = () => {
    if (!showConnectionSelector) return null;
    
    return (
      <Modal
        isOpen={showConnectionSelector}
        onClose={() => setShowConnectionSelector(false)}
        title="Select Connection"
      >
        <div style={{ padding: '1rem' }}>
          <p style={{ marginBottom: '1rem', opacity: 0.7 }}>
            Active Settings require an active connection, or you can edit the saved settings from here.
          </p>
          
          {isLoadingConnections ? (
            <div style={{ textAlign: 'center', padding: '2rem' }}>Loading connections...</div>
          ) : savedConnections.length === 0 ? (
            <div style={{ textAlign: 'center', padding: '2rem', opacity: 0.7 }}>
              No saved connections found.
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem', maxHeight: '400px', overflowY: 'auto' }}>
              {savedConnections.map(conn => (
                <div 
                  key={conn.id} 
                  style={{ 
                    display: 'flex', 
                    justifyContent: 'space-between', 
                    alignItems: 'center',
                    padding: '0.75rem',
                    background: 'var(--color-bg-secondary)',
                    borderRadius: 'var(--radius-sm)'
                  }}
                >
                  <div>
                    <div style={{ fontWeight: 500 }}>{conn.name}</div>
                    <div style={{ fontSize: '0.875rem', opacity: 0.7 }}>{conn.host}:{conn.port}</div>
                  </div>
                  <button
                    className="btn btn-sm btn-primary"
                    onClick={() => handleSelectConnection(conn.id)}
                  >
                    Select
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      </Modal>
    );
  };

  // Show connection selector automatically if no connection is selected
  useEffect(() => {
    if (!connectionId && !showConnectionSelector) {
      handleShowConnectionSelector();
    }
  }, [connectionId, showConnectionSelector]);

  if (isLoading) {
    return (
      <div className="connection-settings-page">
        <div className="settings-loading">Loading settings...</div>
      </div>
    );
  }

  return (
    <Modal
      isOpen={true}
      onClose={() => navigate('/play')}
      title={`Connection Settings${connectionName ? ` - ${connectionName}` : ''}`}
      className="modal-large"
    >
      <div className="connection-settings-page">
        {/* Connection Selector Modal */}
        {renderConnectionSelectorModal()}

        {/* Error/Success messages */}
        {error && (
          <div className="message message-error" style={{ marginBottom: '1rem' }}>
            {error}
            <button onClick={() => setError(null)} style={{ marginLeft: '1rem' }}>Dismiss</button>
          </div>
        )}
        
        {successMessage && (
          <div className="message message-success" style={{ marginBottom: '1rem' }}>
            {successMessage}
          </div>
        )}
        
        {/* PR01PH06: Validation error banner */}
        {validationBanner && (
          <div className="message message-error" style={{ marginBottom: '1rem', whiteSpace: 'pre-wrap' }}>
            ⚠️ {validationBanner}
            <button onClick={() => setValidationBanner(null)} style={{ marginLeft: '1rem' }}>Dismiss</button>
          </div>
        )}

        {/* SP06PH07: Automation status banner */}
        {(automationError || automationDisabled) && (
          <div className="automation-error-banner" style={{ marginBottom: '1rem' }}>
            {automationError ? (
              <>
                <span>⚠️ Automation Paused: {automationError}</span>
                <button onClick={resumeAutomation} className="btn btn-small">
                  Resume Automation
                </button>
                <button onClick={disableAutomation} className="btn btn-small btn-secondary">
                  Disable Automation
                </button>
              </>
            ) : (
              <>
                <span>⚠️ Automation Disabled</span>
                <button onClick={enableAutomation} className="btn btn-small">
                  Enable Automation
                </button>
              </>
            )}
          </div>
        )}

        {/* Active session warning */}
        {connectionState === 'connected' && currentConnectionId === connectionId && (
          <div className="message message-info" style={{ marginBottom: '1rem' }}>
            <strong>Active Session:</strong> You are currently connected. Key bindings, aliases, triggers, and variables all update live.
          </div>
        )}

        <div className="settings-layout">
        {/* Sidebar navigation */}
        <nav className="settings-nav">
          {SECTIONS.map(section => (
            <button
              key={section.id}
              className={`settings-nav-item ${activeSection === section.id ? 'active' : ''}`}
              onClick={() => handleSectionChange(section.id)}
            >
              <span className="settings-nav-icon">{section.icon}</span>
              <span className="settings-nav-label">{section.label}</span>
            </button>
          ))}
          {/* Select Connection button in sidebar */}
          <button
            className="settings-nav-item"
            onClick={() => handleShowConnectionSelector()}
          >
            <span className="settings-nav-icon">🔗</span>
            <span className="settings-nav-label">Select Connection</span>
          </button>
        </nav>

        {/* Content */}
        <div className="settings-content">
          {/* General Section */}
          {activeSection === 'general' && (
            <div className="settings-section">
              <h3>General Settings</h3>
              
              <div className="form-group">
                <label className="form-label">Scrollback Lines</label>
                <input
                  type="number"
                  className="form-input"
                  value={settings.scrollback_limit}
                  onChange={(e) => handleScrollbackChange(e.target.value)}
                  min={100}
                  max={10000}
                />
                <p className="form-hint">Number of lines to keep in terminal history (100-10000)</p>
              </div>

              <div className="form-group form-checkbox">
                <label>
                  <input
                    type="checkbox"
                    checked={settings.echo_input}
                    onChange={(e) => handleSettingChange('echo_input', e.target.checked)}
                  />
                  Echo Commands Locally
                </label>
              </div>

              <div className="form-group form-checkbox">
                <label>
                  <input
                    type="checkbox"
                    checked={settings.timestamp_output}
                    onChange={(e) => handleSettingChange('timestamp_output', e.target.checked)}
                  />
                  Timestamp Output
                </label>
              </div>

              <div className="form-group form-checkbox">
                <label>
                  <input
                    type="checkbox"
                    checked={settings.word_wrap}
                    onChange={(e) => handleSettingChange('word_wrap', e.target.checked)}
                  />
                  Word Wrap
                </label>
              </div>

              <div className="settings-actions">
                <button
                  className="btn btn-primary"
                  onClick={handleSaveProfile}
                  disabled={isSavingProfile}
                >
                  {isSavingProfile ? 'Saving...' : 'Save Settings'}
                </button>
              </div>
            </div>
          )}

          {/* Key Bindings Section */}
          {activeSection === 'keybindings' && (
            <div className="settings-section">
              <h3>Key Bindings</h3>
              <p className="section-description">
                Map keys to commands. Press the key combination to execute the command.
              </p>

              <div className="keybindings-list">
                {Object.entries(keybindings).map(([key, command]) => (
                  <div key={key} className="keybinding-row">
                    <span className="keybinding-key">{key}</span>
                    <span className="keybinding-command">{command}</span>
                    <button
                      className="btn btn-sm btn-icon icon-red"
                      onClick={() => handleRemoveKeybinding(key)}
                      title="Remove"
                    >
                      <svg 
                        viewBox="0 0 24 24" 
                        fill="none" 
                        stroke="currentColor" 
                        strokeWidth="2"
                        strokeLinecap="round" 
                        strokeLinejoin="round"
                      >
                        <polyline points="3 6 5 6 21 6" />
                        <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                      </svg>
                    </button>
                  </div>
                ))}
                {Object.keys(keybindings).length === 0 && (
                  <div className="keybindings-empty">
                    No keybindings configured
                  </div>
                )}
              </div>

              <div className="keybinding-add">
                <div className="form-row">
                  <div className="form-group">
                    <label className="form-label">Key</label>
                    <input
                      type="text"
                      className="form-input"
                      placeholder={isCapturingKey ? 'Press any key...' : 'Click Add then press a key'}
                      value={newKey}
                      onChange={(e) => !isCapturingKey && setNewKey(e.target.value)}
                      disabled={isCapturingKey}
                    />
                  </div>
                  <div className="form-group">
                    <label className="form-label">Command</label>
                    <input
                      type="text"
                      className="form-input"
                      placeholder="e.g., score, cast heal"
                      value={newCommand}
                      onChange={(e) => setNewCommand(e.target.value)}
                    />
                  </div>
                  <button
                    className="btn btn-small btn-primary"
                    onClick={handleAddKeybinding}
                    disabled={!!isCapturingKey || (!!newKey.trim() && !newCommand.trim())}
                  >
                    {isCapturingKey ? 'Press a key...' : newKey.trim() ? 'Add' : 'Capture Key'}
                  </button>
                </div>
              </div>

              <div className="settings-actions">
                <button
                  className="btn btn-primary"
                  onClick={handleSaveProfile}
                  disabled={isSavingProfile}
                >
                  {isSavingProfile ? 'Saving...' : 'Save Key Bindings'}
                </button>
              </div>
            </div>
          )}

          {/* Aliases Section */}
          {activeSection === 'aliases' && (
            <div className="settings-section">
              <h3>Aliases</h3>
              <p className="section-description">
                Aliases transform user-entered commands into different commands before submission.
              </p>

              <div className="automation-list">
                {aliases.map((alias, index) => (
                  <div key={alias.id} className="automation-row">
                    <div className="automation-row-priority" style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '2px' }}>
                      <button
                        className="btn btn-small btn-icon"
                        onClick={() => handleMoveAliasUp(index)}
                        disabled={index === 0}
                        title="Move up (higher priority)"
                        style={{ padding: '2px 4px', minWidth: 'auto' }}
                      >
                        ↑
                      </button>
                      <span className="priority-number" title="Priority order (1 = highest)">#{index + 1}</span>
                      <button
                        className="btn btn-small btn-icon"
                        onClick={() => handleMoveAliasDown(index)}
                        disabled={index === aliases.length - 1}
                        title="Move down (lower priority)"
                        style={{ padding: '2px 4px', minWidth: 'auto' }}
                      >
                        ↓
                      </button>
                    </div>
                    <div className="automation-row-content">
                      <div className="form-group">
                        <label className="form-label">Pattern</label>
                        <input
                          type="text"
                          className="form-input"
                          placeholder="e.g., l"
                          value={alias.pattern}
                          onChange={(e) => handleUpdateAlias(alias.id, { pattern: e.target.value })}
                        />
                      </div>
                      <div className="form-group">
                        <label className="form-label">Replacement</label>
                        <textarea
                          className={`form-input form-textarea ${aliasSyntaxErrors[alias.id] ? 'form-input-error' : ''}`}
                          placeholder="e.g., look %1 or get %1
#IF ${target}
attack ${target}
#ENDIF"
                          value={alias.replacement}
                          rows={3}
                          onChange={(e) => validateAndUpdateAlias(alias.id, { replacement: e.target.value })}
                          onBlur={(e) => {
                            // Validate on blur for immediate feedback
                            const errors = validateAutomationScript(e.target.value);
                            if (errors.length > 0) {
                              const errorMessage = errors.map(err => `Line ${err.line}: ${err.message}`).join('; ');
                              setAliasSyntaxErrors(prev => ({ ...prev, [alias.id]: errorMessage }));
                              console.warn('[SettingsPage] Alias validation errors on blur:', errors);
                            }
                          }}
                        />
                        <p className="form-hint">Use %1, %2, %3 for arguments. Supports #IF/#ELSE/#ENDIF, #SET, #TIMER, #CANCEL</p>
                        {aliasSyntaxErrors[alias.id] && (
                          <div className="form-error" style={{ marginTop: '0.5rem' }}>
                            ⚠️ {aliasSyntaxErrors[alias.id]}
                          </div>
                        )}
                      </div>
                    </div>
                    <div className="automation-row-actions">
                      <button
                        className={`btn btn-small ${alias.enabled ? 'btn-primary' : 'btn-secondary'}`}
                        onClick={() => handleUpdateAlias(alias.id, { enabled: !alias.enabled })}
                        title={alias.enabled ? 'Disable alias' : 'Enable alias'}
                      >
                        {alias.enabled ? 'On' : 'Off'}
                      </button>
                      <button
                        className="btn btn-sm btn-icon icon-red"
                        onClick={() => handleRemoveAlias(alias.id)}
                        title="Remove"
                      >
                        <svg 
                          viewBox="0 0 24 24" 
                          fill="none" 
                          stroke="currentColor" 
                          strokeWidth="2"
                          strokeLinecap="round" 
                          strokeLinejoin="round"
                        >
                          <polyline points="3 6 5 6 21 6" />
                          <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                        </svg>
                      </button>
                    </div>
                  </div>
                ))}
                {aliases.length === 0 && (
                  <div className="automation-empty">
                    No aliases configured
                  </div>
                )}
              </div>

              <div className="settings-actions">
                <button className="btn btn-secondary" onClick={handleAddAlias}>
                  Add Alias
                </button>
                <button
                  className="btn btn-primary"
                  onClick={handleSaveAliases}
                  disabled={isSavingAliases}
                >
                  {isSavingAliases ? 'Saving...' : 'Save Aliases'}
                </button>
              </div>
            </div>
          )}

          {/* Triggers Section */}
          {activeSection === 'triggers' && (
            <div className="settings-section">
              <h3>Triggers</h3>
              <p className="section-description">
                Triggers monitor incoming MUD output and automatically execute commands when matched.
              </p>

              <div className="automation-list">
                {triggers.map((trigger, index) => (
                  <div key={trigger.id} className="automation-row">
                    <div className="automation-row-priority" style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '2px' }}>
                      <button
                        className="btn btn-small btn-icon"
                        onClick={() => handleMoveTriggerUp(index)}
                        disabled={index === 0}
                        title="Move up (higher priority)"
                        style={{ padding: '2px 4px', minWidth: 'auto' }}
                      >
                        ↑
                      </button>
                      <span className="priority-number" title="Priority order (1 = highest)">#{index + 1}</span>
                      <button
                        className="btn btn-small btn-icon"
                        onClick={() => handleMoveTriggerDown(index)}
                        disabled={index === triggers.length - 1}
                        title="Move down (lower priority)"
                        style={{ padding: '2px 4px', minWidth: 'auto' }}
                      >
                        ↓
                      </button>
                    </div>
                    <div className="automation-row-content">
                      <div className="form-group">
                        <label className="form-label">Match</label>
                        <input
                          type="text"
                          className="form-input"
                          placeholder="e.g., You are hungry"
                          value={trigger.match}
                          onChange={(e) => handleUpdateTrigger(trigger.id, { match: e.target.value })}
                        />
                      </div>
                      <div className="form-group">
                        <label className="form-label">Action</label>
                        <textarea
                          className={`form-input form-textarea ${triggerSyntaxErrors[trigger.id] ? 'form-input-error' : ''}`}
                          placeholder="e.g., eat bread
#IF ${hp} < 50
cast heal
#ENDIF"
                          value={trigger.action}
                          rows={3}
                          onChange={(e) => validateAndUpdateTrigger(trigger.id, { action: e.target.value })}
                          onBlur={(e) => {
                            // Validate on blur for immediate feedback
                            const errors = validateAutomationScript(e.target.value);
                            if (errors.length > 0) {
                              const errorMessage = errors.map(err => `Line ${err.line}: ${err.message}`).join('; ');
                              setTriggerSyntaxErrors(prev => ({ ...prev, [trigger.id]: errorMessage }));
                              console.warn('[SettingsPage] Trigger validation errors on blur:', errors);
                            }
                          }}
                        />
                        <p className="form-hint">Supports #IF/#ELSE/#ENDIF, #SET, #TIMER, #CANCEL</p>
                        {triggerSyntaxErrors[trigger.id] && (
                          <div className="form-error" style={{ marginTop: '0.5rem' }}>
                            ⚠️ {triggerSyntaxErrors[trigger.id]}
                          </div>
                        )}
                      </div>
                    </div>
                    <div className="automation-row-actions">
                      <input
                        type="number"
                        className="form-input"
                        style={{ width: '126px' }}
                        value={trigger.cooldown_ms}
                        onChange={(e) => handleUpdateTrigger(trigger.id, { cooldown_ms: parseInt(e.target.value) || 2000 })}
                        min={0}
                        title="Cooldown (ms)"
                      />
                      <button
                        className={`btn btn-small ${trigger.enabled ? 'btn-primary' : 'btn-secondary'}`}
                        onClick={() => handleUpdateTrigger(trigger.id, { enabled: !trigger.enabled })}
                        title={trigger.enabled ? 'Disable trigger' : 'Enable trigger'}
                      >
                        {trigger.enabled ? 'On' : 'Off'}
                      </button>
                      <button
                        className="btn btn-sm btn-icon icon-red"
                        onClick={() => handleRemoveTrigger(trigger.id)}
                        title="Remove"
                      >
                        <svg 
                          viewBox="0 0 24 24" 
                          fill="none" 
                          stroke="currentColor" 
                          strokeWidth="2"
                          strokeLinecap="round" 
                          strokeLinejoin="round"
                        >
                          <polyline points="3 6 5 6 21 6" />
                          <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                        </svg>
                      </button>
                    </div>
                  </div>
                ))}
                {triggers.length === 0 && (
                  <div className="automation-empty">
                    No triggers configured
                  </div>
                )}
              </div>

              <div className="settings-actions">
                <button className="btn btn-secondary" onClick={handleAddTrigger}>
                  Add Trigger
                </button>
                <button
                  className="btn btn-primary"
                  onClick={handleSaveTriggers}
                  disabled={isSavingTriggers}
                >
                  {isSavingTriggers ? 'Saving...' : 'Save Triggers'}
                </button>
              </div>
            </div>
          )}

          {/* Timers Section */}
          {activeSection === 'timers' && (
            <div className="settings-section">
              <h3>Timers</h3>
              <p className="section-description">
                Timers execute commands at specified intervals. Use #TIMER, #START, #STOP, #CANCEL in commands.
              </p>

              <div className="automation-list">
                {timers.map((timer) => (
                  <div key={timer.id} className="automation-row">
                    <div className="automation-row-content">
                      <div className="form-group">
                        <label className="form-label">Name</label>
                        <input
                          type="text"
                          className="form-input"
                          placeholder="e.g., heal_check"
                          value={timer.name}
                          onChange={(e) => handleUpdateTimer(timer.id, { name: e.target.value })}
                        />
                      </div>
                      <div className="form-row">
                        <div className="form-group" style={{ flex: 1 }}>
                          <label className="form-label">Duration (ms)</label>
                          <input
                            type="number"
                            className="form-input"
                            placeholder="e.g., 60000"
                            value={timer.duration}
                            onChange={(e) => handleUpdateTimer(timer.id, { duration: parseInt(e.target.value) || 60000 })}
                            min={1000}
                          />
                          <p className="form-hint">Minimum 1000ms</p>
                        </div>
                        <div className="form-group" style={{ flex: 1 }}>
                          <label className="form-label">Repeat</label>
                          <label className="toggle">
                            <input
                              type="checkbox"
                              checked={timer.repeat}
                              onChange={(e) => handleUpdateTimer(timer.id, { repeat: e.target.checked })}
                            />
                            <span className="toggle-slider"></span>
                            <span className="toggle-label">{timer.repeat ? 'Repeating' : 'One-time'}</span>
                          </label>
                        </div>
                      </div>
                      <div className="form-group">
                        <label className="form-label">Commands</label>
                        <textarea
                          className={`form-input form-textarea ${timerSyntaxErrors[timer.id] ? 'form-input-error' : ''}`}
                          placeholder="e.g., cast heal
#IF ${hp} < 30
cast heal
#ENDIF"
                          value={timer.commands}
                          rows={4}
                          onChange={(e) => validateAndUpdateTimer(timer.id, { commands: e.target.value })}
                          onBlur={(e) => {
                            const errors = validateAutomationScript(e.target.value);
                            if (errors.length > 0) {
                              const errorMessage = errors.map(err => `Line ${err.line}: ${err.message}`).join('; ');
                              setTimerSyntaxErrors(prev => ({ ...prev, [timer.id]: errorMessage }));
                              console.warn('[SettingsPage] Timer validation errors on blur:', errors);
                            }
                          }}
                        />
                        <p className="form-hint">Supports #IF/#ELSE/#ENDIF, #SET, #TIMER, #CANCEL</p>
                        {timerSyntaxErrors[timer.id] && (
                          <div className="form-error" style={{ marginTop: '0.5rem' }}>
                            ⚠️ {timerSyntaxErrors[timer.id]}
                          </div>
                        )}
                      </div>
                    </div>
                    <div className="automation-row-actions">
                      {(() => {
                        // Get runtime state if connected
                        const isConnected = connectionState === 'connected';
                        const runtimeState = isConnected && automationEngine?.getTimerManager 
                          ? automationEngine.getTimerManager().getTimerState(timer.name)
                          : null;
                        const isRunning = runtimeState === 'running';
                        
                        return (
                          <button
                            type="button"
                            className={`btn btn-small ${isRunning ? 'btn-primary' : 'btn-secondary'}`}
                            onClick={async () => {
                              // Get current runtime state for the command
                              const currentRuntimeState = isConnected && automationEngine?.getTimerManager
                                ? automationEngine.getTimerManager().getTimerState(timer.name)
                                : (timer.enabled ? 'running' : 'stopped');
                              const shouldStart = currentRuntimeState !== 'running';
                              
                              // Always toggle the UI state first for user feedback
                              handleUpdateTimer(timer.id, { enabled: shouldStart });
                              
                              // If connected, also send the runtime command to the timer
                              if (automationEngine && connectionState === 'connected') {
                                try {
                                  const command = shouldStart ? `#START ${timer.name}` : `#STOP ${timer.name}`;
                                  console.log('[SettingsPage] Sending timer command:', command);
                                  await automationEngine.processUserInput(command);
                                  console.log('[SettingsPage] Timer command sent successfully');
                                } catch (err) {
                                  console.error('[SettingsPage] Failed to start/stop timer:', err);
                                }
                              } else {
                                console.log('[SettingsPage] Not connected - UI toggle only, automationEngine:', !!automationEngine, 'connectionState:', connectionState);
                              }
                            }}
                            title={isRunning ? 'Stop timer (runtime)' : 'Start timer (runtime)'}
                          >
                            {isRunning ? 'On' : 'Off'}
                          </button>
                        );
                      })()}
                      <button
                        className="btn btn-sm btn-icon icon-red"
                        onClick={() => handleRemoveTimer(timer.id)}
                        title="Remove"
                      >
                        <svg 
                          viewBox="0 0 24 24" 
                          fill="none" 
                          stroke="currentColor" 
                          strokeWidth="2"
                          strokeLinecap="round" 
                          strokeLinejoin="round"
                        >
                          <polyline points="3 6 5 6 21 6" />
                          <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                        </svg>
                      </button>
                    </div>
                  </div>
                ))}
                {timers.length === 0 && (
                  <div className="automation-empty">
                    No timers configured
                  </div>
                )}
              </div>

              <div className="settings-actions">
                <button className="btn btn-secondary" onClick={handleAddTimer}>
                  Add Timer
                </button>
                <button
                  className="btn btn-primary"
                  onClick={handleSaveTimers}
                  disabled={isSavingTimers}
                >
                  {isSavingTimers ? 'Saving...' : 'Save Timers'}
                </button>
              </div>
            </div>
          )}

          {/* Environment Section */}
          {activeSection === 'environment' && (
            <div className="settings-section">
              <h3>Environment Variables</h3>
              <p className="section-description">
                Environment variables store reusable values used by automation rules. 
                Use ${'{variable_name}'} syntax in aliases and trigger actions.
              </p>

              <div className="automation-list">
                {variables.map(variable => (
                  <div key={variable.id} className="automation-row">
                    <div className="automation-row-content">
                      <div className="form-row">
                        <div className="form-group">
                          <label className="form-label">Name</label>
                          <input
                            type="text"
                            className="form-input"
                            placeholder="e.g., target"
                            value={variable.name}
                            onChange={(e) => handleUpdateVariable(variable.id, { name: e.target.value })}
                          />
                        </div>
                        <div className="form-group">
                          <label className="form-label">Value</label>
                          <input
                            type="text"
                            className="form-input"
                            placeholder="e.g., goblin"
                            value={variable.value}
                            onChange={(e) => handleUpdateVariable(variable.id, { value: e.target.value })}
                          />
                        </div>
                      </div>
                    </div>
                    <button
                      className="btn btn-sm btn-icon icon-red"
                      onClick={() => handleRemoveVariable(variable.id)}
                      title="Remove"
                    >
                      <svg 
                        viewBox="0 0 24 24" 
                        fill="none" 
                        stroke="currentColor" 
                        strokeWidth="2"
                        strokeLinecap="round" 
                        strokeLinejoin="round"
                      >
                        <polyline points="3 6 5 6 21 6" />
                        <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                      </svg>
                    </button>
                  </div>
                ))}
                {variables.length === 0 && (
                  <div className="automation-empty">
                    No variables configured
                  </div>
                )}
              </div>

              <div className="settings-actions">
                <button className="btn btn-secondary" onClick={handleAddVariable}>
                  Add Variable
                </button>
                <button
                  className="btn btn-primary"
                  onClick={handleSaveVariables}
                  disabled={isSavingVariables}
                >
                  {isSavingVariables ? 'Saving...' : 'Save Variables'}
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
      </div>
    </Modal>
  );
}
