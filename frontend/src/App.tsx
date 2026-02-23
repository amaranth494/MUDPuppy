import { useState, useCallback } from 'react';
import { SessionProvider, useSession } from './context/SessionContext';
import PlayScreen from './pages/PlayScreen';
import ConnectionsPage from './pages/ConnectionsPage';
import AccountPage from './pages/AccountPage';
import HelpPage from './pages/HelpPage';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import LoginScreen from './pages/LoginScreen';
import Sidebar from './components/Sidebar';
import Modal from './components/Modal';
import QuickConnectModal from './components/QuickConnectModal';

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
  const [isQuickConnectOpen, setIsQuickConnectOpen] = useState(false);
  const location = useLocation();
  const navigate = useNavigate();
  
  // Get setInputLocked from SessionContext (SP03PH03)
  // Modal will manage input lock via onInputLockChange callback
  const { setInputLocked } = useSession();
  
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
  // Note: Input unlock is handled by Modal's useEffect cleanup
  const handleModalClose = useCallback(() => {
    navigate('/play');
  }, [navigate]);
  
  // PlayScreen is always mounted at root level - session persists across route changes
  return (
    <div className={isSidebarOpen ? 'app-shell-with-sidebar' : 'app-shell'}>
      {/* Sidebar - Collapsible drawer (SP03PH02) */}
      {isSidebarOpen && (
        <Sidebar 
          isCollapsed={isSidebarCollapsed}
          onToggle={handleToggleCollapse}
          onPlayClick={() => setIsQuickConnectOpen(true)}
          isPlayDisabled={isOverlayPage}
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
        {/* Focus management and input lock handled by Modal (with useEffect cleanup) */}
        {isOverlayPage && (
          <Modal
            isOpen={isOverlayPage}
            onClose={handleModalClose}
            title={getModalTitle()}
            onInputLockChange={setInputLocked}
          >
            <Routes>
              <Route path="/connections" element={<ConnectionsPage />} />
              <Route path="/account" element={<AccountPage />} />
              <Route path="/help" element={<HelpPage />} />
            </Routes>
          </Modal>
        )}
        
        {/* Quick Connect Modal (SP03PH04) */}
        <QuickConnectModal
          isOpen={isQuickConnectOpen}
          onClose={() => setIsQuickConnectOpen(false)}
        />
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
