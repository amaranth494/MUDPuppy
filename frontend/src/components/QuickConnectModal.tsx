import { useState, useEffect } from 'react';
import Modal from './Modal';
import { useSession } from '../context/SessionContext';
import { getRecentConnections, deleteConnection } from '../services/api';
import { logToConsole } from '../services/log';
import { SavedConnection } from '../types';

// Format a date as relative time (e.g., "5 min ago", "1 day ago")
function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  const diffMin = Math.floor(diffSec / 60);
  const diffHour = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHour / 24);
  const diffWeek = Math.floor(diffDay / 7);
  const diffMonth = Math.floor(diffDay / 30);
  const diffYear = Math.floor(diffDay / 365);

  if (diffSec < 60) {
    return 'just now';
  } else if (diffMin < 60) {
    return `${diffMin} min ago`;
  } else if (diffHour < 24) {
    return `${diffHour} hour${diffHour > 1 ? 's' : ''} ago`;
  } else if (diffDay < 7) {
    return `${diffDay} day${diffDay > 1 ? 's' : ''} ago`;
  } else if (diffWeek < 4) {
    return `${diffWeek} week${diffWeek > 1 ? 's' : ''} ago`;
  } else if (diffMonth < 12) {
    return `${diffMonth} month${diffMonth > 1 ? 's' : ''} ago`;
  } else {
    return `${diffYear} year${diffYear > 1 ? 's' : ''} ago`;
  }
}

interface QuickConnectModalProps {
  isOpen: boolean;
  onClose: () => void;
  onInputLockChange?: (locked: boolean) => void;
}

export default function QuickConnectModal({ isOpen, onClose, onInputLockChange }: QuickConnectModalProps) {
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
        logToConsole('[QuickConnectModal.tsx:loadRecentConnections] QuickConnect: Loading recent connections...');
    setLoadingRecent(true);
    try {
      const recent = await getRecentConnections();
            logToConsole('[QuickConnectModal.tsx:loadRecentConnections] QuickConnect: Got recent connections: ' + recent.length);
      setRecentConnections(recent);
    } catch (err) {
      console.error('[QuickConnect] Failed to load recent connections:', err);
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
    
    // Auto-connect - pass connectionId so profile can be fetched (SP04)
    setIsConnecting(true);
    try {
      await connect(conn.host, conn.port, conn.id);
      onClose();
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : 'Connection failed');
    } finally {
      setIsConnecting(false);
    }
  };
  
  // Handle removing a recent connection
  const handleRemoveConnection = async (conn: SavedConnection, e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      await deleteConnection(conn.id);
      // Reload recent connections after deletion
      loadRecentConnections();
    } catch (err) {
      console.error('Failed to delete connection:', err);
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
      onInputLockChange={onInputLockChange}
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

        {/* Recent Connections */}
        <div className="recent-connections">
          <div className="recent-connections-label">Recent Connections</div>
          <div className="recent-connections-list">
            {recentConnections.length > 0 ? (
              recentConnections.slice(0, 5).map((conn) => (
                <div key={conn.id} className="recent-connection-row">
                  <span className="recent-connection-name">
                    {conn.name}{conn.username ? ` as ${conn.username}` : ''}
                  </span>
                  <span className="recent-connection-time">
                    Last connect: {conn.last_connected_at ? formatRelativeTime(conn.last_connected_at) : 'Never'}
                  </span>
                  <div className="recent-connection-actions">
                    <button 
                      className="btn btn-sm btn-icon icon-green" 
                      onClick={() => handleRecentConnectionClick(conn)}
                      disabled={!canConnect && !isConnectingState}
                      title="Connect"
                    >
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z"></path></svg>
                    </button>
                    <button 
                      className="btn btn-sm btn-icon icon-red" 
                      onClick={(e) => handleRemoveConnection(conn, e)}
                      title="Remove this connection"
                    >
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
                    </button>
                  </div>
                </div>
              ))
            ) : (
              <div className="recent-connection-row">
                <span className="recent-connection-name">No Recent Connections</span>
              </div>
            )}
          </div>
        </div>
      </div>
    </Modal>
  );
}
