import { useState, useEffect, useCallback } from 'react';
import Modal from './Modal';
import HostPortForm from './HostPortForm';
import { useSession } from '../context/SessionContext';
import {
  getConnections,
  createConnection,
  updateConnection,
  deleteConnection,
  getCredentialStatus,
  setCredentials,
  updateCredentials,
  deleteCredentials,
} from '../services/api';
import { SavedConnection, CreateConnectionRequest, UpdateConnectionRequest, SetCredentialsRequest, mapBackendError } from '../types';

interface ConnectionsHubModalProps {
  isOpen: boolean;
  onClose: () => void;
}

// View types for the modal
type ViewType = 'list' | 'create' | 'edit';

export default function ConnectionsHubModal({ isOpen, onClose }: ConnectionsHubModalProps) {
  const { connectionState, connect: sessionConnect } = useSession();
  
  // State
  const [view, setView] = useState<ViewType>('list');
  const [connections, setConnections] = useState<SavedConnection[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  
  // Selected connection for editing
  const [selectedConnection, setSelectedConnection] = useState<SavedConnection | null>(null);
  
  // Delete confirmation
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null);
  
  // Connecting state for specific connection
  const [connectingId, setConnectingId] = useState<string | null>(null);
  
  // Form state for create/edit
  const [formName, setFormName] = useState('');
  const [formHost, setFormHost] = useState('');
  const [formPort, setFormPort] = useState(23);
  const [formProtocol, setFormProtocol] = useState('telnet');
  
  // Credential form state
  const [credUsername, setCredUsername] = useState('');
  const [credPassword, setCredPassword] = useState('');
  const [credAutoLogin, setCredAutoLogin] = useState(false);
  const [hasCredentials, setHasCredentials] = useState(false);
  const [isLoadingCredStatus, setIsLoadingCredStatus] = useState(false);
  
  // Load connections when modal opens
  const loadConnections = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const data = await getConnections();
      setConnections(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load connections');
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Load credential status for selected connection
  const loadCredentialStatus = useCallback(async (connectionId: string) => {
    setIsLoadingCredStatus(true);
    try {
      const status = await getCredentialStatus(connectionId);
      setHasCredentials(status.has_credentials);
      setCredAutoLogin(status.auto_login_enabled);
    } catch (err) {
      console.error('Failed to load credential status:', err);
      setHasCredentials(false);
    } finally {
      setIsLoadingCredStatus(false);
    }
  }, []);

  // Reset form when modal opens
  useEffect(() => {
    if (isOpen) {
      setView('list');
      setError(null);
      setSuccessMessage(null);
      loadConnections();
    }
  }, [isOpen, loadConnections]);

  // Reset form when switching to create/edit
  const resetForm = () => {
    setFormName('');
    setFormHost('');
    setFormPort(23);
    setFormProtocol('telnet');
    setCredUsername('');
    setCredPassword('');
    setCredAutoLogin(false);
    setHasCredentials(false);
    setError(null);
  };

  // Handle create new connection
  const handleCreate = () => {
    resetForm();
    setView('create');
  };

  // Handle edit connection
  const handleEdit = (conn: SavedConnection) => {
    setSelectedConnection(conn);
    setFormName(conn.name);
    setFormHost(conn.host);
    setFormPort(conn.port);
    setFormProtocol(conn.protocol || 'telnet');
    loadCredentialStatus(conn.id);
    setView('edit');
  };

  // Handle save create
  const handleSaveCreate = async () => {
    if (!formName.trim()) {
      setError('Name is required');
      return;
    }
    if (!formHost.trim()) {
      setError('Host is required');
      return;
    }

    setIsLoading(true);
    setError(null);
    try {
      const request: CreateConnectionRequest = {
        name: formName.trim(),
        host: formHost.trim(),
        port: formPort,
        protocol: formProtocol,
      };
      await createConnection(request);
      
      setSuccessMessage('Connection created');
      await loadConnections();
      setView('list');
      resetForm();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create connection');
    } finally {
      setIsLoading(false);
    }
  };

  // Handle save edit
  const handleSaveEdit = async () => {
    if (!selectedConnection) return;
    if (!formName.trim()) {
      setError('Name is required');
      return;
    }
    if (!formHost.trim()) {
      setError('Host is required');
      return;
    }

    setIsLoading(true);
    setError(null);
    try {
      const request: UpdateConnectionRequest = {
        name: formName.trim(),
        host: formHost.trim(),
        port: formPort,
        protocol: formProtocol,
      };
      await updateConnection(selectedConnection.id, request);
      
      // Handle credentials if password provided or auto-login changed
      if (credPassword || credAutoLogin !== selectedConnection.auto_login_enabled) {
        if (credPassword) {
          const credRequest: SetCredentialsRequest = {
            username: credUsername,
            password: credPassword,
            auto_login: credAutoLogin,
          };
          if (hasCredentials) {
            await updateCredentials(selectedConnection.id, credRequest);
          } else {
            await setCredentials(selectedConnection.id, credRequest);
          }
        } else if (credAutoLogin !== selectedConnection.auto_login_enabled) {
          // Just update auto-login
          const credRequest: SetCredentialsRequest = {
            username: '',
            password: '',
            auto_login: credAutoLogin,
          };
          await updateCredentials(selectedConnection.id, credRequest);
        }
      }
      
      setSuccessMessage('Connection updated');
      await loadConnections();
      setView('list');
      resetForm();
      setSelectedConnection(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update connection');
    } finally {
      setIsLoading(false);
    }
  };

  // Handle delete
  const handleDelete = async (id: string) => {
    setIsLoading(true);
    setError(null);
    try {
      await deleteConnection(id);
      setSuccessMessage('Connection deleted');
      await loadConnections();
      setDeleteConfirmId(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete connection');
    } finally {
      setIsLoading(false);
    }
  };

  // Handle connect from hub
  const handleConnect = async (conn: SavedConnection) => {
    // Check if already connected
    if (connectionState === 'connected') {
      setError('Disconnect from current session before connecting to saved connection');
      return;
    }
    if (connectionState === 'connecting') {
      setError('Already connecting to a server');
      return;
    }

    setConnectingId(conn.id);
    setError(null);
    try {
      // Use session connect which handles both API and WebSocket
      await sessionConnect(conn.host, conn.port);
      // Close modal after successful connect
      onClose();
    } catch (err) {
      setError(err instanceof Error ? mapBackendError(err.message) : 'Failed to connect');
    } finally {
      setConnectingId(null);
    }
  };

  // Handle back to list
  const handleBack = () => {
    setView('list');
    setSelectedConnection(null);
    resetForm();
    setError(null);
  };

  // Render list view
  const renderListView = () => (
    <div className="connections-hub-list">
      {/* Saved Connections */}
      <div className="connections-section">
        <div className="connections-section-header">
          <h3 className="connections-section-title">Saved Connections</h3>
          <button className="btn btn-sm btn-primary" onClick={handleCreate}>
            New Connection
          </button>
        </div>
        
        {isLoading ? (
          <div className="connections-loading">Loading...</div>
        ) : connections.length === 0 ? (
          <div className="connections-empty">
            No saved connections yet. Create one to get started.
          </div>
        ) : (
          <div className="connections-list">
            {connections.map(conn => (
              <div key={conn.id} className="connection-item">
                <div className="connection-info">
                  <span className="connection-name">{conn.name}</span>
                  <span className="connection-address">{conn.host}:{conn.port}</span>
                  {conn.has_credentials && (
                    <span className="connection-badge">Credentials stored</span>
                  )}
                  {conn.auto_login_enabled && (
                    <span className="connection-badge badge-auto">Auto-login</span>
                  )}
                </div>
                <div className="connection-actions">
                  <button
                    className="btn btn-sm btn-primary"
                    onClick={() => handleConnect(conn)}
                    disabled={connectingId === conn.id || connectionState === 'connected' || connectionState === 'connecting'}
                  >
                    {connectingId === conn.id ? 'Connecting...' : 'Connect'}
                  </button>
                  <button
                    className="btn btn-sm btn-secondary"
                    onClick={() => handleEdit(conn)}
                  >
                    Edit
                  </button>
                  {deleteConfirmId === conn.id ? (
                    <div className="delete-confirm">
                      <span>Delete?</span>
                      <button
                        className="btn btn-sm btn-danger"
                        onClick={() => handleDelete(conn.id)}
                        disabled={isLoading}
                      >
                        Yes
                      </button>
                      <button
                        className="btn btn-sm"
                        onClick={() => setDeleteConfirmId(null)}
                      >
                        No
                      </button>
                    </div>
                  ) : (
                    <button
                      className="btn btn-sm btn-danger"
                      onClick={() => setDeleteConfirmId(conn.id)}
                    >
                      Delete
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Error/Success messages */}
      {error && <div className="message message-error">{error}</div>}
      {successMessage && <div className="message message-success">{successMessage}</div>}
    </div>
  );

  // Render create/edit form
  const renderFormView = () => (
    <div className="connections-hub-form">
      <h3 className="form-title">{view === 'create' ? 'New Connection' : 'Edit Connection'}</h3>
      
      <div className="form-group">
        <label className="form-label" htmlFor="conn-name">
          Name
        </label>
        <input
          id="conn-name"
          type="text"
          className="form-input"
          placeholder="My MUD Server"
          value={formName}
          onChange={(e) => setFormName(e.target.value)}
        />
      </div>

      <HostPortForm
        host={formHost}
        port={formPort}
        onHostChange={setFormHost}
        onPortChange={setFormPort}
      />

      <div className="form-group">
        <label className="form-label" htmlFor="conn-protocol">
          Protocol
        </label>
        <select
          id="conn-protocol"
          className="form-input"
          value={formProtocol}
          onChange={(e) => setFormProtocol(e.target.value)}
        >
          <option value="telnet">Telnet</option>
          <option value="ssl">SSL/Telnet</option>
        </select>
      </div>

      {/* Credentials Section */}
      <div className="credentials-section">
        <h4 className="credentials-title">Credentials (Optional)</h4>
        
        {view === 'edit' && isLoadingCredStatus ? (
          <div className="credentials-loading">Loading...</div>
        ) : view === 'edit' && hasCredentials ? (
          <div className="credentials-status">
            <span className="credential-indicator stored">✓ Credentials stored</span>
            {credAutoLogin && <span className="credential-indicator auto">Auto-login enabled</span>}
          </div>
        ) : null}

        <div className="form-group">
          <label className="form-label" htmlFor="cred-username">
            Username
          </label>
          <input
            id="cred-username"
            type="text"
            className="form-input"
            placeholder="Username for auto-login"
            value={credUsername}
            onChange={(e) => setCredUsername(e.target.value)}
          />
        </div>

        <div className="form-group">
          <label className="form-label" htmlFor="cred-password">
            Password
          </label>
          <input
            id="cred-password"
            type="password"
            className="form-input"
            placeholder={hasCredentials ? '••••••••' : 'Enter password'}
            value={credPassword}
            onChange={(e) => setCredPassword(e.target.value)}
          />
          {view === 'edit' && hasCredentials && !credPassword && (
            <span className="form-hint">Leave blank to keep current password</span>
          )}
        </div>

        <div className="form-group form-checkbox">
          <label>
            <input
              type="checkbox"
              checked={credAutoLogin}
              onChange={(e) => setCredAutoLogin(e.target.checked)}
            />
            <span>Use auto-login</span>
          </label>
          <span className="form-hint">Automatically send credentials when connecting</span>
        </div>

        {view === 'edit' && hasCredentials && (
          <button
            className="btn btn-sm btn-danger"
            onClick={async () => {
              if (selectedConnection) {
                try {
                  await deleteCredentials(selectedConnection.id);
                  setHasCredentials(false);
                  setCredAutoLogin(false);
                  setSuccessMessage('Credentials cleared');
                } catch (err) {
                  setError(err instanceof Error ? err.message : 'Failed to clear credentials');
                }
              }
            }}
          >
            Clear Credentials
          </button>
        )}
      </div>

      {/* Error message */}
      {error && <div className="message message-error">{error}</div>}

      {/* Form actions */}
      <div className="form-actions">
        <button className="btn btn-secondary" onClick={handleBack}>
          Cancel
        </button>
        <button
          className="btn btn-primary"
          onClick={view === 'create' ? handleSaveCreate : handleSaveEdit}
          disabled={isLoading}
        >
          {isLoading ? 'Saving...' : 'Save'}
        </button>
      </div>
    </div>
  );

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Connections"
    >
      {view === 'list' ? renderListView() : renderFormView()}
    </Modal>
  );
}
