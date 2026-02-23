import { useState, useEffect, useCallback } from 'react';
import { useSession } from '../context/SessionContext';

/**
 * SessionBadge Component - derives from GET /api/v1/session/status (Constitution IX)
 * 
 * Uses 15s polling + event-driven refresh:
 * - Visibility change (tab becomes visible)
 * - Connect/disconnect action completion (via SessionContext state changes)
 * - WebSocket events (via SessionContext state changes)
 * 
 * This defends correctness while avoiding long-term churn from aggressive polling.
 */
export default function SessionBadge() {
  const { connectionState, refreshStatus } = useSession();
  const [lastRefresh, setLastRefresh] = useState<Date>(new Date());
  
  // Event-driven refresh: refresh immediately when tab becomes visible
  const handleVisibilityChange = useCallback(() => {
    if (!document.hidden) {
      // Tab became visible - refresh status
      refreshStatus().then(() => setLastRefresh(new Date()));
    }
  }, [refreshStatus]);
  
  // Set up visibility listener
  useEffect(() => {
    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [handleVisibilityChange]);
  
  // Reduced polling: 15s interval while connected
  useEffect(() => {
    const interval = setInterval(() => {
      refreshStatus().then(() => setLastRefresh(new Date()));
    }, 15000); // 15 seconds - reduced from 5s
    
    return () => clearInterval(interval);
  }, [refreshStatus]);
  
  const getStateDisplay = () => {
    switch (connectionState) {
      case 'connected':
        return 'Connected';
      case 'connecting':
        return 'Connecting';
      case 'disconnected':
        return 'Disconnected';
      case 'error':
        return 'Error';
      default:
        return 'Disconnected';
    }
  };
  
  const getStatusClass = () => {
    switch (connectionState) {
      case 'connected':
        return 'status-connected';
      case 'connecting':
        return 'status-connecting';
      case 'error':
        return 'status-error';
      default:
        return 'status-disconnected';
    }
  };
  
  return (
    <div 
      className={`session-badge ${getStatusClass()}`} 
      title={`Last updated: ${lastRefresh.toLocaleTimeString()}`}
    >
      <span className="session-badge-dot" />
      <span className="session-badge-text">{getStateDisplay()}</span>
    </div>
  );
}
