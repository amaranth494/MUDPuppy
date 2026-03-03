import { useState, useEffect, useCallback } from 'react';
import Modal from './Modal';
import { getProfileByConnection, updateProfile } from '../services/api';
import { Profile, ProfileSettings, UpdateProfileRequest } from '../types';

interface ProfileModalProps {
  isOpen: boolean;
  onClose: () => void;
  connectionId: string;
}

export default function ProfileModal({ isOpen, onClose, connectionId }: ProfileModalProps) {
  const [profile, setProfile] = useState<Profile | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Form state
  const [keybindings, setKeybindings] = useState<Record<string, string>>({});
  const [newKey, setNewKey] = useState('');
  const [newCommand, setNewCommand] = useState('');
  const [settings, setSettings] = useState<ProfileSettings>({
    scrollback_limit: 1000,
    echo_input: false,
    timestamp_output: false,
    word_wrap: true,
  });

  // Load profile when modal opens
  const loadProfile = useCallback(async () => {
    if (!connectionId || !isOpen) return;
    
    setIsLoading(true);
    setError(null);
    try {
      const data = await getProfileByConnection(connectionId);
      setProfile(data);
      setKeybindings(data.keybindings || {});
      setSettings(data.settings || {
        scrollback_limit: 1000,
        echo_input: false,
        timestamp_output: false,
        word_wrap: true,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load profile');
    } finally {
      setIsLoading(false);
    }
  }, [connectionId, isOpen]);

  useEffect(() => {
    loadProfile();
  }, [loadProfile]);

  // Reset form when closing
  useEffect(() => {
    if (!isOpen) {
      setKeybindings({});
      setNewKey('');
      setNewCommand('');
      setError(null);
      setSuccessMessage(null);
    }
  }, [isOpen]);

  // Add a new keybinding
  const handleAddKeybinding = () => {
    if (!newKey.trim() || !newCommand.trim()) {
      return;
    }

    // Check max 50 keybindings
    if (Object.keys(keybindings).length >= 50) {
      setError('Maximum 50 keybindings allowed');
      return;
    }

    // Check command max 500 chars
    if (newCommand.length > 500) {
      setError('Command must be 500 characters or less');
      return;
    }

    setKeybindings(prev => ({
      ...prev,
      [newKey.trim()]: newCommand.trim(),
    }));
    setNewKey('');
    setNewCommand('');
    setError(null);
  };

  // Remove a keybinding
  const handleRemoveKeybinding = (key: string) => {
    const newBindings = { ...keybindings };
    delete newBindings[key];
    setKeybindings(newBindings);
  };

  // Save profile
  const handleSave = async () => {
    if (!profile) return;

    setIsSaving(true);
    setError(null);
    setSuccessMessage(null);

    try {
      const updateRequest: UpdateProfileRequest = {
        keybindings,
        settings,
      };

      await updateProfile(profile.id, updateRequest);
      setSuccessMessage('Profile saved successfully');
      
      // Close after brief delay
      setTimeout(() => {
        onClose();
      }, 1000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save profile');
    } finally {
      setIsSaving(false);
    }
  };

  // Handle settings change
  const handleSettingChange = (key: keyof ProfileSettings, value: number | boolean) => {
    setSettings(prev => ({
      ...prev,
      [key]: value,
    }));
  };

  // Validate scrollback
  const handleScrollbackChange = (value: string) => {
    const num = parseInt(value) || 100;
    const clamped = Math.max(100, Math.min(10000, num));
    setSettings(prev => ({
      ...prev,
      scrollback_limit: clamped,
    }));
  };

  if (isLoading) {
    return (
      <Modal isOpen={isOpen} onClose={onClose} title="Profile Settings">
        <div className="profile-modal-loading">
          Loading profile...
        </div>
      </Modal>
    );
  }

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Profile Settings">
      <div className="profile-modal">
        {error && (
          <div className="message message-error">
            {error}
          </div>
        )}

        {successMessage && (
          <div className="message message-success">
            {successMessage}
          </div>
        )}

        <div className="profile-section">
          <h3>Keybindings</h3>
          <p className="profile-section-description">
            Map keys to commands. Press the key combination to execute the command.
          </p>

          {/* Keybindings list */}
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

          {/* Add new keybinding */}
          <div className="keybinding-add">
            <div className="form-row">
              <div className="form-group">
                <label className="form-label">Key</label>
                <input
                  type="text"
                  className="form-input"
                  placeholder="e.g., F1, Ctrl+1, Alt+W"
                  value={newKey}
                  onChange={(e) => setNewKey(e.target.value)}
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
                disabled={!newKey.trim() || !newCommand.trim()}
              >
                Add
              </button>
            </div>
            <p className="profile-hint">
              Max 50 keybindings, max 500 characters per command
            </p>
          </div>
        </div>

        <div className="profile-section">
          <h3>Settings</h3>

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
            <p className="profile-hint">Number of lines to keep in terminal history (100-10000)</p>
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
            <p className="profile-hint">
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
            <p className="profile-hint">
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
            <p className="profile-hint">
              Wrap long lines at terminal width
            </p>
          </div>
        </div>

        <div className="profile-actions">
          <button
            className="btn btn-primary"
            onClick={handleSave}
            disabled={isSaving}
          >
            {isSaving ? 'Saving...' : 'Save Profile'}
          </button>
          <button
            className="btn btn-secondary"
            onClick={onClose}
          >
            Cancel
          </button>
        </div>

        <p className="profile-note">
          Note: Profile changes apply on next connect. Current session continues with previous settings.
        </p>
      </div>
    </Modal>
  );
}
