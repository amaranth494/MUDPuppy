import { useState, useEffect } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useSession } from '../context/SessionContext';

// Session Badge Component - derives from GET /api/v1/session/status (Constitution IX)
export function SessionBadge() {
  const { connectionState, refreshStatus } = useSession();
  
  // Refresh status periodically to ensure badge matches backend truth
  useEffect(() => {
    const interval = setInterval(() => {
      refreshStatus();
    }, 5000); // Poll every 5 seconds
    
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
    <div className={`session-badge ${getStatusClass()}`}>
      <span className="session-badge-dot" />
      <span className="session-badge-text">{getStateDisplay()}</span>
    </div>
  );
}

export default function Header() {
  const { user } = useSession();
  const [showDropdown, setShowDropdown] = useState(false);
  const location = useLocation();

  const isActive = (path: string) => location.pathname === path;

  return (
    <header className="header">
      <div className="header-left">
        <Link to="/play">
          <img 
            src="/mudpuppy_logo.png" 
            alt="MUDPuppy" 
            className="header-logo"
          />
        </Link>
        <Link to="/play" className="header-title">
          MUDPuppy
        </Link>
      </div>

      <nav className="nav">
        <Link 
          to="/play" 
          className={`nav-link ${isActive('/play') ? 'active' : ''}`}
        >
          Play
        </Link>
        <Link 
          to="/connections" 
          className={`nav-link ${isActive('/connections') ? 'active' : ''}`}
        >
          Connections
        </Link>
        <Link 
          to="/account" 
          className={`nav-link ${isActive('/account') ? 'active' : ''}`}
        >
          Account
        </Link>
        <Link 
          to="/help" 
          className={`nav-link ${isActive('/help') ? 'active' : ''}`}
        >
          Help
        </Link>
      </nav>

      <div className="header-right">
        {/* Session Badge - Persistent status indicator derived from API */}
        <SessionBadge />

        <div className="account-menu">
          <button 
            className="account-button"
            onClick={() => setShowDropdown(!showDropdown)}
          >
            {user?.email || 'Account'}
            <span>▼</span>
          </button>
          
          {showDropdown && (
            <div className="account-dropdown">
              <Link 
                to="/account" 
                className="account-dropdown-item"
                onClick={() => setShowDropdown(false)}
              >
                Account
              </Link>
              <Link 
                to="/help" 
                className="account-dropdown-item"
                onClick={() => setShowDropdown(false)}
              >
                Help
              </Link>
              <button 
                className="account-dropdown-item"
                onClick={() => {
                  setShowDropdown(false);
                  window.location.href = '/';
                }}
              >
                Sign Out
              </button>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}
