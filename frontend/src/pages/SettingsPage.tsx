import { useState, useEffect, useCallback } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { getConnections, getProfileByConnection, updateProfile, getAliases, putAliases, getTriggers, putTriggers, getEnvironment, putEnvironment } from '../services/api';
import { SavedConnection, ProfileSettings, UpdateProfileRequest, Alias, Trigger, Variable } from '../types';
import { normalizeKeybindings, eventToCanonicalKey, isValidKeybindingFormat, isValidCommand, canonicalizeKeybinding, isModifierOnly } from '../services/keybindings';
import { useSession } from '../context/SessionContext';
import Modal from '../components/Modal';

// Section types
type SettingsSection = 'general' | 'keybindings' | 'aliases' | 'triggers' | 'environment';

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
  { id: 'environment', label: 'Environment', icon: '📦' },
];

export default function SettingsPage() {
  // Use useLocation for URL detection instead of useParams to ensure proper re-renders
  const location = useLocation();
  const navigate = useNavigate();
  const { updateProfile: updateSessionProfile, connectionState, currentConnectionId, automationEngine, automationError, automationDisabled, disableAutomation, enableAutomation, resumeAutomation } = useSession();
  
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
  
  // Form state - Triggers
  const [triggers, setTriggers] = useState<Trigger[]>([]);
  const [isSavingTriggers, setIsSavingTriggers] = useState(false);
  
  // Form state - Variables
  const [variables, setVariables] = useState<Variable[]>([]);
  const [isSavingVariables, setIsSavingVariables] = useState(false);
  
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
    
    setIsSavingAliases(true);
    setError(null);
    setSuccessMessage(null);
    
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
    
    setIsSavingTriggers(true);
    setError(null);
    setSuccessMessage(null);
    
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
                      className="btn btn-small btn-danger"
                      onClick={() => handleRemoveKeybinding(key)}
                    >
                      Remove
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
                {aliases.map(alias => (
                  <div key={alias.id} className="automation-row">
                    <div className="automation-row-content">
                      <div className="form-row">
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
                          <input
                            type="text"
                            className="form-input"
                            placeholder="e.g., look %1 or get %1"
                            value={alias.replacement}
                            onChange={(e) => handleUpdateAlias(alias.id, { replacement: e.target.value })}
                          />
                          <p className="form-hint">Use %1, %2, %3 for arguments (e.g., "get %1" for "g sword" → "get sword")</p>
                        </div>
                        <div className="form-group form-checkbox">
                          <label>
                            <input
                              type="checkbox"
                              checked={alias.enabled}
                              onChange={(e) => handleUpdateAlias(alias.id, { enabled: e.target.checked })}
                            />
                            Enabled
                          </label>
                        </div>
                      </div>
                    </div>
                    <button
                      className="btn btn-small btn-danger"
                      onClick={() => handleRemoveAlias(alias.id)}
                    >
                      Remove
                    </button>
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
                {triggers.map(trigger => (
                  <div key={trigger.id} className="automation-row">
                    <div className="automation-row-content">
                      <div className="form-row">
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
                          <input
                            type="text"
                            className="form-input"
                            placeholder="e.g., eat bread"
                            value={trigger.action}
                            onChange={(e) => handleUpdateTrigger(trigger.id, { action: e.target.value })}
                          />
                        </div>
                        <div className="form-group">
                          <label className="form-label">Cooldown (ms)</label>
                          <input
                            type="number"
                            className="form-input"
                            value={trigger.cooldown_ms}
                            onChange={(e) => handleUpdateTrigger(trigger.id, { cooldown_ms: parseInt(e.target.value) || 2000 })}
                            min={0}
                          />
                        </div>
                        <div className="form-group form-checkbox">
                          <label>
                            <input
                              type="checkbox"
                              checked={trigger.enabled}
                              onChange={(e) => handleUpdateTrigger(trigger.id, { enabled: e.target.checked })}
                            />
                            Enabled
                          </label>
                        </div>
                      </div>
                    </div>
                    <button
                      className="btn btn-small btn-danger"
                      onClick={() => handleRemoveTrigger(trigger.id)}
                    >
                      Remove
                    </button>
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
                      className="btn btn-small btn-danger"
                      onClick={() => handleRemoveVariable(variable.id)}
                    >
                      Remove
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
