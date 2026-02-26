import { useState, useCallback } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useSession } from '../context/SessionContext';
import SessionBadge from './SessionBadge';

interface SidebarProps {
  /** Whether the sidebar is collapsed to icon-only mode */
  isCollapsed?: boolean;
  /** Callback when toggle button is clicked */
  onToggle?: () => void;
  /** Callback when Play is clicked - opens Quick Connect modal */
  onPlayClick?: () => void;
  /** Callback when Connections is clicked - opens Connections Hub modal */
  onConnectionsClick?: () => void;
  /** Whether Play button should be disabled (when route modal is open) */
  isPlayDisabled?: boolean;
}

export default function Sidebar({ isCollapsed = false, onToggle, onPlayClick, onConnectionsClick, isPlayDisabled }: SidebarProps) {
  const { user } = useSession();
  const [showDropdown, setShowDropdown] = useState(false);
  const location = useLocation();

  const isActive = (path: string) => location.pathname === path;

  const handleSignOut = useCallback(() => {
    setShowDropdown(false);
    window.location.href = '/';
  }, []);

  return (
    <aside className={`sidebar ${isCollapsed ? 'sidebar-collapsed' : ''}`}>
      {/* Toggle Button */}
      <button 
        className="sidebar-toggle"
        onClick={onToggle}
        title={isCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        aria-label={isCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
      >
        {isCollapsed ? '→' : '←'}
      </button>

      {/* Logo */}
      <div className="sidebar-logo">
        <Link to="/play">
          <img 
            src="/mudpuppy_logo.png" 
            alt="MUDPuppy" 
            className="sidebar-logo-img"
          />
        </Link>
        {!isCollapsed && (
          <span className="sidebar-logo-text">MUDPuppy</span>
        )}
      </div>

      {/* Navigation */}
      <nav className="sidebar-nav">
        {/* Play - opens Quick Connect modal */}
        <button
          className={`sidebar-nav-item ${isActive('/play') ? 'active' : ''}`}
          title={isCollapsed ? 'Play' : undefined}
          onClick={onPlayClick}
          disabled={isPlayDisabled}
        >
          <span className="sidebar-nav-icon">▶</span>
          {!isCollapsed && (
            <span className="sidebar-nav-label">Play</span>
          )}
        </button>
        
        {/* Connections - opens Connections Hub modal */}
        <button
          className={`sidebar-nav-item ${isActive('/connections') ? 'active' : ''}`}
          title={isCollapsed ? 'Connections' : undefined}
          onClick={onConnectionsClick}
        >
          <span className="sidebar-nav-icon">⚡</span>
          {!isCollapsed && (
            <span className="sidebar-nav-label">Connections</span>
          )}
        </button>
        
        {/* Help - navigates to Help */}
        <Link
          to="/help"
          className={`sidebar-nav-item ${isActive('/help') ? 'active' : ''}`}
          title={isCollapsed ? 'Help' : undefined}
        >
          <span className="sidebar-nav-icon">?</span>
          {!isCollapsed && (
            <span className="sidebar-nav-label">Help</span>
          )}
        </Link>
      </nav>

      {/* Spacer to push user menu to bottom */}
      <div className="sidebar-spacer" />

      {/* Session Status Badge - always visible, above user menu */}
      <div className="sidebar-status">
        <SessionBadge />
      </div>

      {/* User Menu */}
      <div className="sidebar-user">
        <button 
          className="sidebar-user-button"
          onClick={() => setShowDropdown(!showDropdown)}
          title={isCollapsed ? user?.email || 'Account' : undefined}
        >
          <span className="sidebar-user-icon">👤</span>
          {!isCollapsed && (
            <>
              <span className="sidebar-user-email">
                {user?.email || 'Account'}
              </span>
              <span className="sidebar-user-caret">▼</span>
            </>
          )}
        </button>
        
        {showDropdown && (
          <div className="sidebar-dropdown">
            <Link 
              to="/account" 
              className="sidebar-dropdown-item"
              onClick={() => setShowDropdown(false)}
            >
              Account
            </Link>
            <Link 
              to="/help" 
              className="sidebar-dropdown-item"
              onClick={() => setShowDropdown(false)}
            >
              Help
            </Link>
            <button 
              className="sidebar-dropdown-item sidebar-signout"
              onClick={handleSignOut}
            >
              Sign Out
            </button>
          </div>
        )}
      </div>
    </aside>
  );
}
