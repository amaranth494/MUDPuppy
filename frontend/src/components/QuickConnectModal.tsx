import { useState, useEffect } from 'react';
import Modal from './Modal';
import { useSession } from '../context/SessionContext';
import { getRecentConnections } from '../services/api';
import { SavedConnection } from '../types';

interface QuickConnectModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export default function QuickConnectModal({ isOpen, onClose }: QuickConnectModalProps) {
  const { connect, disconnect, connectionState, error, host, port } = useSession();
  
  const [inputHost, setInputHost] = useState('');
  const [inputPort, setInputPort] = useState(23);
  const [isConnecting, setIsConnecting] = useState(false);
  const [localError, setLocalError] = useState<string | null>(null);
  const [recentConnections, setRecentConnections] = useState<SavedConnection[]>([]);
  const [, setLoadingRecent] = useState(false);

  // Reset form when modal opens/closes
  useEffect(() => {
    if (isOpen) {
      // Pre-fill with current connection if connected
      if (connectionState === 'connected' && host) {
        setInputHost(host);
        setInputPort(port || 23);
      } else {
        setInputHost('');
        setInputPort(23);
      }
      setLocalError(null);
      
      // Load recent connections
      loadRecentConnections();
    }
  }, [isOpen, connectionState, host, port]);

  // Load recent connections
  const loadRecentConnections = async () => {
    setLoadingRecent(true);
    try {
      const recent = await getRecentConnections();
      setRecentConnections(recent);
    } catch (err) {
      console.error('Failed to load recent connections:', err);
    } finally {
      setLoadingRecent(false);
    }
  };

  // Sync error from session context
  useEffect(() => {
    if (error) {
      setLocalError(error);
    } else {
      setLocalError(null);
    }
  }, [error]);

  // Close modal only when connection succeeds (transition to connected state)
  // But NOT when already connected (to allow disconnect access)
  useEffect(() => {
    if (isOpen && connectionState === 'connected' && !isConnecting) {
      // Only close if we just completed a connection (was previously connecting)
      // Don't close if already connected (user wants to access disconnect)
      // We track this by checking if we were in the connecting state
      // For now, only auto-close if there's no current host (not already connected)
      if (!host) {
        onClose();
      }
    }
  }, [isOpen, connectionState, isConnecting, onClose, host]);

  const handleConnect = async () => {
    if (!inputHost.trim()) {
      setLocalError('Host is required');
      return;
    }

    setIsConnecting(true);
    setLocalError(null);
    
    try {
      await connect(inputHost.trim(), inputPort);
      // Close modal on successful connect
      onClose();
    } catch (err) {
      // Error is handled by session context and synced via useEffect
      // Modal stays open to show error
      setLocalError(err instanceof Error ? err.message : 'Connection failed');
    } finally {
      setIsConnecting(false);
    }
  };

  const handleDisconnect = async () => {
    setLocalError(null);
    try {
      await disconnect();
      // Close modal after disconnect
      onClose();
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : 'Disconnect failed');
    }
  };

  // Handle clicking on a recent connection - auto-connect
  const handleRecentConnectionClick = async (conn: SavedConnection) => {
    if (!canConnect && !isConnectingState) return;
    
    setInputHost(conn.host);
    setInputPort(conn.port);
    setLocalError(null);
    
    // Auto-connect
    setIsConnecting(true);
    try {
      await connect(conn.host, conn.port);
      onClose();
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : 'Connection failed');
    } finally {
      setIsConnecting(false);
    }
  };

  // State machine for UI
  const canConnect = (connectionState === 'disconnected' || connectionState === 'error') && !isConnecting;
  const canDisconnect = connectionState === 'connected' || connectionState === 'error';
  const isConnectingState = connectionState === 'connecting' || isConnecting;
  
  const showError = localError && connectionState === 'error';

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && canConnect && inputHost.trim()) {
      handleConnect();
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Quick Play"
    >
      <div className="quick-connect-form">
        <div className="form-group">
          <label className="form-label" htmlFor="quick-connect-host">
            Host
          </label>
          <input
            id="quick-connect-host"
            type="text"
            className="form-input"
            placeholder="e.g., example.com"
            value={inputHost}
            onChange={(e) => setInputHost(e.target.value)}
            disabled={!canConnect && !isConnectingState}
            onKeyDown={handleKeyDown}
            autoFocus
          />
        </div>

        <div className="form-group">
          <label className="form-label" htmlFor="quick-connect-port">
            Port
          </label>
          <input
            id="quick-connect-port"
            type="number"
            className="form-input"
            placeholder="23"
            value={inputPort}
            onChange={(e) => setInputPort(parseInt(e.target.value) || 23)}
            disabled={!canConnect && !isConnectingState}
            onKeyDown={handleKeyDown}
          />
        </div>

        {/* Error Message */}
        {(showError || localError) && (
          <div className="message message-error">
            {localError}
          </div>
        )}

        {/* Connection Status */}
        {connectionState === 'connected' && (
          <div className="message message-info">
            Connected to {host}:{port}
          </div>
        )}

        {/* Action Buttons */}
        <div className="quick-connect-actions">
          {canConnect && (
            <button
              className="btn btn-primary"
              onClick={handleConnect}
              disabled={!inputHost.trim() || isConnectingState}
            >
              {isConnectingState ? 'Connecting...' : 'Connect'}
            </button>
          )}

          {isConnectingState && !canConnect && (
            <button className="btn btn-secondary" disabled>
              Connecting...
            </button>
          )}

          {canDisconnect && (
            <>
              <button
                className="btn btn-danger"
                onClick={handleDisconnect}
              >
                Disconnect
              </button>
              <div className="quick-connect-status">
                Currently connected to {host}:{port}
              </div>
            </>
          )}
        </div>

        {/* Recent Connections - Show last 5 with Connect buttons or placeholder */}
        <div className="recent-connections">
          <div className="recent-connections-label">Recent Connections</div>
          {recentConnections.length > 0 ? (
            <div className="recent-connections-list">
              {recentConnections.slice(0, 5).map((conn) => (
                <div key={conn.id} className="recent-connection-row">
                  <span className="recent-connection-name">{conn.name}</span>
                  <button
                    className="btn btn-small btn-primary"
                    onClick={() => handleRecentConnectionClick(conn)}
                    disabled={!canConnect && !isConnectingState}
                    title={`Connect to ${conn.host}:${conn.port}`}
                  >
                    Connect
                  </button>
                </div>
              ))}
            </div>
          ) : (
            <div className="recent-connections-empty">
              No Recent Connections
            </div>
          )}
        </div>
      </div>
    </Modal>
  );
}
