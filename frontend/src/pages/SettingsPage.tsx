import { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { getProfileByConnection, updateProfile, getAliases, putAliases, getTriggers, putTriggers, getEnvironment, putEnvironment } from '../services/api';
import { ProfileSettings, UpdateProfileRequest, Alias, Trigger, Variable } from '../types';
import { normalizeKeybindings, eventToCanonicalKey, isValidKeybindingFormat, isValidCommand, canonicalizeKeybinding, isModifierOnly } from '../services/keybindings';
import { useSession } from '../context/SessionContext';

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
  const { section } = useParams<{ section: string }>();
  const navigate = useNavigate();
  const { updateProfile: updateSessionProfile, connectionState, currentConnectionId } = useSession();
  
  // Determine active section from URL or default to general
  const activeSection: SettingsSection = (section as SettingsSection) || 'general';
  
  // Connection and profile state - load from active connection or prompt to select
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [connectionId, setConnectionId] = useState<string | null>(currentConnectionId);
  
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

  useEffect(() => {
    // If we have a current connection, use it
    if (currentConnectionId && connectionState === 'connected') {
      setConnectionId(currentConnectionId);
    } else if (!currentConnectionId) {
      // No active connection - redirect to connections page with message
      navigate('/connections', { state: { fromSettings: true } });
    }
  }, [currentConnectionId, connectionState, navigate]);

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
      if (connectionState === 'connected' && currentConnectionId === connectionId) {
        setSuccessMessage('Aliases saved. Changes will take effect on your next connection.');
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
      if (connectionState === 'connected' && currentConnectionId === connectionId) {
        setSuccessMessage('Triggers saved. Changes will take effect on your next connection.');
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
      if (connectionState === 'connected' && currentConnectionId === connectionId) {
        setSuccessMessage('Variables saved. Changes will take effect on your next connection.');
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
      type: 'exact',
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

  // No connection selected - show prompt
  if (!connectionId) {
    return (
      <div className="connection-settings-page">
        <div className="settings-header">
          <h2>Connection Settings</h2>
        </div>
        
        <div className="message message-info" style={{ marginBottom: '1rem' }}>
          <strong>Select a Connection</strong>
          <p>Active Settings require an active connection, or you can edit the saved settings from here.</p>
          <p>Please select a connection from the Connections menu to edit its settings.</p>
        </div>

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
          </nav>

          {/* Content */}
          <div className="settings-content">
            <div className="settings-section">
              <p style={{ opacity: 0.7, textAlign: 'center', padding: '2rem' }}>
                Select a connection to edit its {activeSection} settings.
              </p>
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="connection-settings-page">
        <div className="settings-loading">Loading settings...</div>
      </div>
    );
  }

  return (
    <div className="connection-settings-page">
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

      {/* Active session warning */}
      {connectionState === 'connected' && currentConnectionId === connectionId && (
        <div className="message message-info" style={{ marginBottom: '1rem' }}>
          <strong>Active Session:</strong> You are currently connected. Key bindings update live. 
          Other changes take effect on next connection.
        </div>
      )}

      {/* Header */}
      <div className="settings-header">
        <h2>Connection Settings</h2>
      </div>

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
                          <label className="form-label">Type</label>
                          <select
                            className="form-input"
                            value={alias.type}
                            onChange={(e) => handleUpdateAlias(alias.id, { type: e.target.value as 'exact' | 'prefix' })}
                          >
                            <option value="exact">Exact</option>
                            <option value="prefix">Prefix</option>
                          </select>
                        </div>
                        <div className="form-group">
                          <label className="form-label">Replacement</label>
                          <input
                            type="text"
                            className="form-input"
                            placeholder="e.g., look"
                            value={alias.replacement}
                            onChange={(e) => handleUpdateAlias(alias.id, { replacement: e.target.value })}
                          />
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
  );
}
