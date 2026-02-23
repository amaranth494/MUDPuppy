import { useNavigate, Routes, Route, useLocation } from 'react-router-dom';
import { SessionProvider, useSession } from './context/SessionContext';
import PlayScreen from './pages/PlayScreen';
import ConnectionsPage from './pages/ConnectionsPage';
import AccountPage from './pages/AccountPage';
import HelpPage from './pages/HelpPage';
import LoginScreen from './pages/LoginScreen';
import Header from './components/Header';

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
  const location = useLocation();
  
  // Pages that should render as overlays (not unmount PlayScreen)
  const isOverlayPage = 
    location.pathname === '/connections' || 
    location.pathname === '/account' || 
    location.pathname === '/help';
  
  // PlayScreen is always mounted at root level - session persists across route changes
  return (
    <div className="app-shell">
      <Header />
      <main className="app-main">
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
