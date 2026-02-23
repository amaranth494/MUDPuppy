import { useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useSession } from '../context/SessionContext';
import SessionBadge from './SessionBadge';

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
