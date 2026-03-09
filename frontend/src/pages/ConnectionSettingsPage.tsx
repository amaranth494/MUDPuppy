import { useState, useEffect, useCallback } from 'react';
import { useParams, Link } from 'react-router-dom';
import { getConnections, getProfileByConnection, updateProfile, getAliases, putAliases, getTriggers, putTriggers } from '../services/api';
import { SavedConnection, Profile, ProfileSettings, UpdateProfileRequest, Alias, Trigger } from '../types';
import { normalizeKeybindings, eventToCanonicalKey, isValidKeybindingFormat, isValidCommand, canonicalizeKeybinding, isModifierOnly } from '../services/keybindings';
import { useSession } from '../context/SessionContext';
import EnvironmentPanel from '../components/EnvironmentPanel';
import Modal from '../components/Modal';
import { aliasTemplates, triggerTemplates, createAliasFromTemplate, createTriggerFromTemplate } from '../templates';

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

export default function ConnectionSettingsPage() {
  const { id: connectionId } = useParams<{ id: string }>();
  const { updateProfile: updateSessionProfile, connectionState, currentConnectionId } = useSession();
  
  // Check if currently connected to this specific connection
  const isConnectedToThis = connectionState === 'connected' && currentConnectionId === connectionId;
  
  // Connection and profile state
  const [connection, setConnection] = useState<SavedConnection | null>(null);
  const [profile, setProfile] = useState<Profile | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  
  // Active section
  const [activeSection, setActiveSection] = useState<SettingsSection>('general');
  
  // Form state - Keybindings
  const [keybindings, setKeybindings] = useState<Record<string, string>>({});
  const [newKey, setNewKey] = useState('');
  const [newCommand, setNewCommand] = useState('');
  const [isCapturingKey, setIsCapturingKey] = useState(false);
  
  // Form state - Aliases
  const [aliases, setAliases] = useState<Alias[]>([]);
  const [isSavingAliases, setIsSavingAliases] = useState(false);
  const [aliasErrors, setAliasErrors] = useState<Record<string, string>>({});
  
  // Template modal state
  const [showAliasTemplateModal, setShowAliasTemplateModal] = useState(false);
  const [showTriggerTemplateModal, setShowTriggerTemplateModal] = useState(false);

  // Validate single alias
  const validateAlias = (alias: Alias): string | null => {
    if (!alias.pattern || !alias.pattern.trim()) {
      return 'Pattern is required';
    }
    if (!alias.replacement || !alias.replacement.trim()) {
      return 'Replacement is required';
    }
    return null;
  };

  // Validate all aliases
  const validateAliases = (): boolean => {
    const errors: Record<string, string> = {};
    let hasErrors = false;
    
    aliases.forEach(alias => {
      const error = validateAlias(alias);
      if (error) {
        errors[alias.id] = error;
        hasErrors = true;
      }
    });
    
    setAliasErrors(errors);
    return !hasErrors;
  };
  
  // Form state - Triggers
  const [triggers, setTriggers] = useState<Trigger[]>([]);
  const [isSavingTriggers, setIsSavingTriggers] = useState(false);
  const [triggerErrors, setTriggerErrors] = useState<Record<string, string>>({});

  // Validate single trigger
  const validateTrigger = (trigger: Trigger): string | null => {
    if (!trigger.match || !trigger.match.trim()) {
      return 'Match is required';
    }
    if (!trigger.action || !trigger.action.trim()) {
      return 'Action is required';
    }
    if (trigger.cooldown_ms < 0) {
      return 'Cooldown must be positive';
    }
    return null;
  };

  // Validate all triggers
  const validateTriggers = (): boolean => {
    const errors: Record<string, string> = {};
    let hasErrors = false;
    
    triggers.forEach(trigger => {
      const error = validateTrigger(trigger);
      if (error) {
        errors[trigger.id] = error;
        hasErrors = true;
      }
    });
    
    setTriggerErrors(errors);
    return !hasErrors;
  };
  
  // Form state - Settings
  const [settings, setSettings] = useState<ProfileSettings>({
    scrollback_limit: 1000,
    echo_input: false,
    timestamp_output: false,
    word_wrap: true,
  });
  const [isSavingProfile, setIsSavingProfile] = useState(false);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Load connection and profile data
  const loadData = useCallback(async () => {
    if (!connectionId) return;
    
    setIsLoading(true);
    setError(null);
    try {
      // Load connection details
      const connections = await getConnections();
      const conn = connections.find(c => c.id === connectionId);
      if (!conn) {
        setError('Connection not found');
        return;
      }
      setConnection(conn);
      
      // Load profile
      const profileData = await getProfileByConnection(connectionId);
      setProfile(profileData);
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
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load connection settings');
    } finally {
      setIsLoading(false);
    }
  }, [connectionId]);

  useEffect(() => {
    loadData();
  }, [loadData]);

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
    if (!profile || !connectionId) return;

    setIsSavingProfile(true);
    setError(null);
    setSuccessMessage(null);

    try {
      const updateRequest: UpdateProfileRequest = {
        keybindings,
        settings,
      };

      await updateProfile(profile.id, updateRequest);
      
      if (isConnectedToThis) {
        setSuccessMessage('Settings saved. Key bindings, aliases, triggers, and variables updated live.');
      } else {
        setSuccessMessage('Settings saved successfully');
      }
      
      // Update session context for real-time keybinding updates
      updateSessionProfile({
        ...profile,
        keybindings,
        settings,
      });

      // Reload to get fresh data
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
    
    // Validate aliases first
    if (!validateAliases()) {
      setError('Please fix validation errors before saving');
      return;
    }
    
    // Check max limit
    if (aliases.length > 200) {
      setError('Maximum 200 aliases allowed');
      return;
    }
    
    setIsSavingAliases(true);
    setError(null);
    setSuccessMessage(null);
    
    try {
      await putAliases(connectionId, aliases);
      if (isConnectedToThis) {
        setSuccessMessage('Aliases saved and updated live.');
      } else {
        setSuccessMessage('Aliases saved successfully');
      }
      setAliasErrors({});
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
    
    // Validate triggers first
    if (!validateTriggers()) {
      setError('Please fix validation errors before saving');
      return;
    }
    
    // Check max limit
    if (triggers.length > 200) {
      setError('Maximum 200 triggers allowed');
      return;
    }
    
    setIsSavingTriggers(true);
    setError(null);
    setSuccessMessage(null);
    
    try {
      await putTriggers(connectionId, triggers);
      if (isConnectedToThis) {
        setSuccessMessage('Triggers saved and updated live.');
      } else {
        setSuccessMessage('Triggers saved successfully');
      }
      setTriggerErrors({});
      await loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save triggers');
    } finally {
      setIsSavingTriggers(false);
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
    // Check max limit
    if (aliases.length >= 200) {
      setError('Maximum 200 aliases allowed');
      return;
    }
    
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
    try {
      await putAliases(connectionId!, newAliases);
    } catch (err) {
      console.error('Failed to persist alias reorder:', err);
    }
  };

  // Move alias down (reorder)
  const handleMoveAliasDown = async (index: number) => {
    if (index >= aliases.length - 1) return;
    const newAliases = [...aliases];
    [newAliases[index], newAliases[index + 1]] = [newAliases[index + 1], newAliases[index]];
    setAliases(newAliases);
    // Persist immediately to server
    try {
      await putAliases(connectionId!, newAliases);
    } catch (err) {
      console.error('Failed to persist alias reorder:', err);
    }
  };

  // Import alias from template
  const handleImportAliasTemplate = (templateIndex: number) => {
    if (aliases.length >= 200) {
      setError('Maximum 200 aliases allowed');
      setShowAliasTemplateModal(false);
      return;
    }
    
    const template = aliasTemplates[templateIndex];
    if (template) {
      const newAlias = createAliasFromTemplate(template);
      setAliases([...aliases, newAlias]);
      setShowAliasTemplateModal(false);
    }
  };

  // Add trigger
  const handleAddTrigger = () => {
    // Check max limit
    if (triggers.length >= 200) {
      setError('Maximum 200 triggers allowed');
      return;
    }
    
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
    try {
      await putTriggers(connectionId!, newTriggers);
    } catch (err) {
      console.error('Failed to persist trigger reorder:', err);
    }
  };

  // Move trigger down (reorder)
  const handleMoveTriggerDown = async (index: number) => {
    if (index >= triggers.length - 1) return;
    const newTriggers = [...triggers];
    [newTriggers[index], newTriggers[index + 1]] = [newTriggers[index + 1], newTriggers[index]];
    setTriggers(newTriggers);
    // Persist immediately to server
    try {
      await putTriggers(connectionId!, newTriggers);
    } catch (err) {
      console.error('Failed to persist trigger reorder:', err);
    }
  };

  // Import trigger from template
  const handleImportTriggerTemplate = (templateIndex: number) => {
    if (triggers.length >= 200) {
      setError('Maximum 200 triggers allowed');
      setShowTriggerTemplateModal(false);
      return;
    }
    
    const template = triggerTemplates[templateIndex];
    if (template) {
      const newTrigger = createTriggerFromTemplate(template);
      setTriggers([...triggers, newTrigger]);
      setShowTriggerTemplateModal(false);
    }
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

  if (isLoading) {
    return (
      <div className="connection-settings-page">
        <div className="settings-loading">Loading connection settings...</div>
      </div>
    );
  }

  if (!connection) {
    return (
      <div className="connection-settings-page">
        <div className="settings-error">
          <p>Connection not found</p>
          <Link to="/connections" className="btn btn-secondary">Back to Connections</Link>
        </div>
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

      {/* Active session warning - show when editing settings for connected session */}
      {isConnectedToThis && (
        <div className="message message-info" style={{ marginBottom: '1rem' }}>
          <strong>Active Session:</strong> You are currently connected to this server. 
          Key bindings, aliases, triggers, and environment variables all support live updates.
        </div>
      )}

      {/* Header */}
      <div className="settings-header">
        <h2>Connection Settings: {connection.name}</h2>
        <Link to="/connections" className="btn btn-secondary">Back to Connections</Link>
      </div>

      <div className="settings-layout">
        {/* Sidebar navigation */}
        <nav className="settings-nav">
          {SECTIONS.map(section => (
            <button
              key={section.id}
              className={`settings-nav-item ${activeSection === section.id ? 'active' : ''}`}
              onClick={() => setActiveSection(section.id)}
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
                <p className="form-hint">
                  Display commands locally before server response. Disable if server echoes commands.
                </p>
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
                <p className="form-hint">
                  Prefix each line with a timestamp
                </p>
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
                <p className="form-hint">
                  Wrap long lines at terminal width
                </p>
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
                    <label className="form-label">
                      Key
                      <span className="tooltip-icon" title="Press a key or key combination. Avoid browser shortcuts like Ctrl+T, Ctrl+W, F5.">?</span>
                    </label>
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
                    <label className="form-label">
                      Command
                      <span className="tooltip-icon" title="The command to send when the key is pressed.">?</span>
                    </label>
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
                <p className="profile-hint">
                  Max 50 keybindings, max 500 characters per command
                </p>
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
              
              {/* Contextual Guidance */}
              <div className="contextual-help">
                <p className="help-text">
                  <strong>Aliases transform typed commands before they are sent to the server.</strong>
                </p>
                <p className="help-example">
                  Examples: <code>l</code> → <code>look</code>, <code>k goblin</code> → <code>kill goblin</code>
                </p>
              </div>

              <div className="automation-list">
                {aliases.map((alias, index) => (
                  <div key={alias.id} className="automation-row">
                    <div className="automation-row-priority">
                      <span className="priority-number" title="Priority order (1 = highest)">#{index + 1}</span>
                    </div>
                    <div className="automation-row-content">
                      <div className="form-row">
                        <div className="form-group">
                          <label className="form-label">
                            Pattern
                            <span className="tooltip-icon" title="The text to match in your command. Use 'exact' for full match or 'prefix' for partial match.">?</span>
                          </label>
                          <input
                            type="text"
                            className={`form-input ${aliasErrors[alias.id]?.includes('Pattern') ? 'form-input-error' : ''}`}
                            placeholder="e.g., l"
                            value={alias.pattern}
                            onChange={(e) => {
                              handleUpdateAlias(alias.id, { pattern: e.target.value });
                              // Clear error when user types
                              if (aliasErrors[alias.id]) {
                                setAliasErrors(prev => {
                                  const newErrors = { ...prev };
                                  delete newErrors[alias.id];
                                  return newErrors;
                                });
                              }
                            }}
                          />
                          <p className="form-hint">Use %1, %2, %3 for arguments (e.g., "get %1" for "g sword" → "get sword")</p>
                        </div>
                        <div className="form-group">
                          <label className="form-label">Replacement</label>
                          <input
                            type="text"
                            className={`form-input ${aliasErrors[alias.id]?.includes('Replacement') ? 'form-input-error' : ''}`}
                            placeholder="e.g., look %1 or get %1"
                            value={alias.replacement}
                            onChange={(e) => {
                              handleUpdateAlias(alias.id, { replacement: e.target.value });
                              // Clear error when user types
                              if (aliasErrors[alias.id]) {
                                setAliasErrors(prev => {
                                  const newErrors = { ...prev };
                                  delete newErrors[alias.id];
                                  return newErrors;
                                });
                              }
                            }}
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
                      {aliasErrors[alias.id] && (
                        <div className="form-error">{aliasErrors[alias.id]}</div>
                      )}
                    </div>
                    <div className="automation-row-actions">
                      <button
                        className="btn btn-small btn-icon"
                        onClick={() => handleMoveAliasUp(index)}
                        disabled={index === 0}
                        title="Move up (higher priority)"
                      >
                        ↑
                      </button>
                      <button
                        className="btn btn-small btn-icon"
                        onClick={() => handleMoveAliasDown(index)}
                        disabled={index === aliases.length - 1}
                        title="Move down (lower priority)"
                      >
                        ↓
                      </button>
                      <button
                        className="btn btn-small btn-danger"
                        onClick={() => handleRemoveAlias(alias.id)}
                      >
                        Remove
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
                <button
                  className="btn btn-secondary"
                  onClick={handleAddAlias}
                >
                  Add Alias
                </button>
                <button
                  className="btn btn-secondary"
                  onClick={() => setShowAliasTemplateModal(true)}
                >
                  Add Example
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
                Triggers monitor incoming MUD output and automatically execute commands when a condition is matched.
              </p>
              
              {/* Contextual Guidance */}
              <div className="contextual-help">
                <p className="help-text">
                  <strong>Triggers activate when matching server output is received.</strong>
                </p>
                <p className="help-note">
                  Note: Matching is currently <strong>case-sensitive substring matching</strong>.
                </p>
              </div>

              <div className="automation-list">
                {triggers.map((trigger, index) => (
                  <div key={trigger.id} className="automation-row">
                    <div className="automation-row-priority">
                      <span className="priority-number" title="Priority order (1 = highest)">#{index + 1}</span>
                    </div>
                    <div className="automation-row-content">
                      <div className="form-row">
                        <div className="form-group">
                          <label className="form-label">
                            Match
                            <span className="tooltip-icon" title="Case-sensitive substring match. The trigger fires when this text appears anywhere in the server output.">?</span>
                          </label>
                          <input
                            type="text"
                            className={`form-input ${triggerErrors[trigger.id]?.includes('Match') ? 'form-input-error' : ''}`}
                            placeholder="e.g., You are hungry"
                            value={trigger.match}
                            onChange={(e) => {
                              handleUpdateTrigger(trigger.id, { match: e.target.value });
                              // Clear error when user types
                              if (triggerErrors[trigger.id]) {
                                setTriggerErrors(prev => {
                                  const newErrors = { ...prev };
                                  delete newErrors[trigger.id];
                                  return newErrors;
                                });
                              }
                            }}
                          />
                        </div>
                        <div className="form-group">
                          <label className="form-label">Action</label>
                          <input
                            type="text"
                            className={`form-input ${triggerErrors[trigger.id]?.includes('Action') ? 'form-input-error' : ''}`}
                            placeholder="e.g., eat bread"
                            value={trigger.action}
                            onChange={(e) => {
                              handleUpdateTrigger(trigger.id, { action: e.target.value });
                              // Clear error when user types
                              if (triggerErrors[trigger.id]) {
                                setTriggerErrors(prev => {
                                  const newErrors = { ...prev };
                                  delete newErrors[trigger.id];
                                  return newErrors;
                                });
                              }
                            }}
                          />
                        </div>
                        <div className="form-group">
                          <label className="form-label">Cooldown (ms)</label>
                          <input
                            type="number"
                            className={`form-input ${triggerErrors[trigger.id]?.includes('Cooldown') ? 'form-input-error' : ''}`}
                            placeholder="2000"
                            value={trigger.cooldown_ms}
                            onChange={(e) => {
                              handleUpdateTrigger(trigger.id, { cooldown_ms: parseInt(e.target.value) || 2000 });
                              // Clear error when user types
                              if (triggerErrors[trigger.id]) {
                                setTriggerErrors(prev => {
                                  const newErrors = { ...prev };
                                  delete newErrors[trigger.id];
                                  return newErrors;
                                });
                              }
                            }}
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
                      {triggerErrors[trigger.id] && (
                        <div className="form-error">{triggerErrors[trigger.id]}</div>
                      )}
                    </div>
                    <div className="automation-row-actions">
                      <button
                        className="btn btn-small btn-icon"
                        onClick={() => handleMoveTriggerUp(index)}
                        disabled={index === 0}
                        title="Move up (higher priority)"
                      >
                        ↑
                      </button>
                      <button
                        className="btn btn-small btn-icon"
                        onClick={() => handleMoveTriggerDown(index)}
                        disabled={index === triggers.length - 1}
                        title="Move down (lower priority)"
                      >
                        ↓
                      </button>
                      <button
                        className="btn btn-small btn-danger"
                        onClick={() => handleRemoveTrigger(trigger.id)}
                      >
                        Remove
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
                <button
                  className="btn btn-secondary"
                  onClick={handleAddTrigger}
                >
                  Add Trigger
                </button>
                <button
                  className="btn btn-secondary"
                  onClick={() => setShowTriggerTemplateModal(true)}
                >
                  Add Example
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
            <EnvironmentPanel
              connectionId={connectionId || ''}
              isConnectedToThis={isConnectedToThis}
            />
          )}
        </div>
      </div>

      {/* Alias Template Modal */}
      <Modal
        isOpen={showAliasTemplateModal}
        onClose={() => setShowAliasTemplateModal(false)}
        title="Add Alias Example"
        className="template-modal"
      >
        <div className="template-list">
          {aliasTemplates.map((template, index) => (
            <div key={index} className="template-item">
              <div className="template-info">
                <div className="template-name">{template.name}</div>
                <div className="template-description">{template.description}</div>
                <div className="template-preview">
                  <code>{template.pattern}</code> → <code>{template.replacement}</code>
                </div>
              </div>
              <button
                className="btn btn-small btn-primary"
                onClick={() => handleImportAliasTemplate(index)}
              >
                Add
              </button>
            </div>
          ))}
        </div>
      </Modal>

      {/* Trigger Template Modal */}
      <Modal
        isOpen={showTriggerTemplateModal}
        onClose={() => setShowTriggerTemplateModal(false)}
        title="Add Trigger Example"
        className="template-modal"
      >
        <div className="template-list">
          {triggerTemplates.map((template, index) => (
            <div key={index} className="template-item">
              <div className="template-info">
                <div className="template-name">{template.name}</div>
                <div className="template-description">{template.description}</div>
                <div className="template-preview">
                  When: <code>{template.match}</code> → <code>{template.action}</code>
                  <span className="template-cooldown"> (cooldown: {template.cooldown_ms}ms)</span>
                </div>
              </div>
              <button
                className="btn btn-small btn-primary"
                onClick={() => handleImportTriggerTemplate(index)}
              >
                Add
              </button>
            </div>
          ))}
        </div>
      </Modal>
    </div>
  );
}
