import { useState, useCallback } from 'react';
import { SessionProvider, useSession } from './context/SessionContext';
import PlayScreen from './pages/PlayScreen';
import ConnectionsPage from './pages/ConnectionsPage';
import AccountPage from './pages/AccountPage';
import HelpPage from './pages/HelpPage';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import LoginScreen from './pages/LoginScreen';
import Sidebar from './components/Sidebar';

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

function OverlayCloseButton() {
  const navigate = useNavigate();
  
  return (
    <button 
      className="overlay-close"
      onClick={() => navigate('/play')}
    >
      ✕ Close
    </button>
  );
}

function AppContent() {
  // Drawer state - ephemeral (in-memory only, per Constitution V.a)
  // Does NOT persist to localStorage
  const [isSidebarOpen, setIsSidebarOpen] = useState(true);
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);
  const location = useLocation();
  
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
        
        {/* Other pages render as overlays on top of PlayScreen */}
        {isOverlayPage && (
          <div className="overlay-layer">
            <OverlayCloseButton />
            <div className="overlay-content">
              <Routes>
                <Route path="/connections" element={<ConnectionsPage />} />
                <Route path="/account" element={<AccountPage />} />
                <Route path="/help" element={<HelpPage />} />
              </Routes>
            </div>
          </div>
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
