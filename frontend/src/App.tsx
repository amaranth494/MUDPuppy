import { useState, useCallback, useEffect } from 'react';
import { SessionProvider, useSession } from './context/SessionContext';
import PlayScreen from './pages/PlayScreen';
import ConnectionsPage from './pages/ConnectionsPage';
import AccountPage from './pages/AccountPage';
import HelpPage from './pages/HelpPage';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import LoginScreen from './pages/LoginScreen';
import Sidebar from './components/Sidebar';
import Modal from './components/Modal';

function AuthGuard({ children }: { children: React.ReactNode }) {
  const { user, isLoading } = useSession();
  
  if (isLoading) {
    return (
      <div className="loading">
        <div className="spinner" />
      </div>
    );
  }
  
  if (!user) {
    return <LoginScreen />;
  }
  
  return <>{children}</>;
}

function AppContent() {
  // Drawer state - ephemeral (in-memory only, per Constitution V.a)
  // Does NOT persist to localStorage
  const [isSidebarOpen, setIsSidebarOpen] = useState(true);
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);
  const location = useLocation();
  const navigate = useNavigate();
  
  // Get input lock state from SessionContext (SP03PH03)
  const { isInputLocked, setInputLocked } = useSession();
  
  // Drawer toggle handlers - have no session impact (SP03PH02T02)
  const handleToggleSidebar = useCallback(() => {
    setIsSidebarOpen(prev => !prev);
  }, []);
  
  const handleToggleCollapse = useCallback(() => {
    setIsSidebarCollapsed(prev => !prev);
  }, []);
  
  // Pages that should render as overlays (not unmount PlayScreen)
  const isOverlayPage = 
    location.pathname === '/connections' || 
    location.pathname === '/account' || 
    location.pathname === '/help';
  
  // Get modal title based on current route
  const getModalTitle = () => {
    switch (location.pathname) {
      case '/connections':
        return 'Connections';
      case '/account':
        return 'Account';
      case '/help':
        return 'Help';
      default:
        return '';
    }
  };
  
  // Handle modal close - navigate back to Play
  const handleModalClose = useCallback(() => {
    navigate('/play');
    // Unlock input when modal closes (SP03PH03)
    setInputLocked(false);
  }, [navigate, setInputLocked]);
  
  // Lock input when modal opens (SP03PH03)
  useEffect(() => {
    if (isOverlayPage && !isInputLocked) {
      setInputLocked(true);
    }
  }, [isOverlayPage, isInputLocked, setInputLocked]);
  
  // PlayScreen is always mounted at root level - session persists across route changes
  return (
    <div className={isSidebarOpen ? 'app-shell-with-sidebar' : 'app-shell'}>
      {/* Sidebar - Collapsible drawer (SP03PH02) */}
      {isSidebarOpen && (
        <Sidebar 
          isCollapsed={isSidebarCollapsed}
          onToggle={handleToggleCollapse}
        />
      )}
      <main className={isSidebarOpen ? 'app-main-with-sidebar' + (isSidebarCollapsed ? ' sidebar-collapsed' : '') : 'app-main'}>
        {/* Drawer toggle button - toggles sidebar open/close (SP03PH02T02) */}
        <button
          className="drawer-toggle-button"
          onClick={handleToggleSidebar}
          title={isSidebarOpen ? 'Close sidebar' : 'Open sidebar'}
          aria-label={isSidebarOpen ? 'Close sidebar' : 'Open sidebar'}
        >
          {isSidebarOpen ? '☰' : '▶'}
        </button>
        
        {/* PlayScreen is ALWAYS mounted - this is the key to session persistence */}
        <div className="play-layer">
          <PlayScreen />
        </div>
        
        {/* Modal overlay for Connections, Account, Help pages (SP03PH03) */}
        {/* Uses Modal component for full-height overlay, close button, ESC support */}
        {/* Focus management and input lock handled by Modal and SessionContext */}
        {isOverlayPage && (
          <Modal
            isOpen={isOverlayPage}
            onClose={handleModalClose}
            title={getModalTitle()}
          >
            <Routes>
              <Route path="/connections" element={<ConnectionsPage />} />
              <Route path="/account" element={<AccountPage />} />
              <Route path="/help" element={<HelpPage />} />
            </Routes>
          </Modal>
        )}
      </main>
    </div>
  );
}

function AppRoutes() {
  return (
    <AuthGuard>
      <AppContent />
    </AuthGuard>
  );
}

export default function App() {
  return (
    <SessionProvider>
      <AppRoutes />
    </SessionProvider>
  );
}
