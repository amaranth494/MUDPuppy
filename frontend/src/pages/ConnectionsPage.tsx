import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { getConnections } from '../services/api';
import { SavedConnection } from '../types';

export default function ConnectionsPage() {
  const [connections, setConnections] = useState<SavedConnection[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const loadConnections = async () => {
      try {
        const data = await getConnections();
        setConnections(data);
      } catch (err) {
        console.error('Failed to load connections:', err);
      } finally {
        setIsLoading(false);
      }
    };
    loadConnections();
  }, []);

  if (isLoading) {
    return (
      <div style={{ padding: '2rem', textAlign: 'center' }}>
        Loading connections...
      </div>
    );
  }

  return (
    <div style={{ padding: '2rem' }}>
      <h2>Connections</h2>
      <p style={{ marginTop: '1rem', opacity: 0.7 }}>
        Select a connection to configure its settings.
      </p>
      
      {connections.length === 0 ? (
        <div style={{ marginTop: '2rem', padding: '2rem', textAlign: 'center', background: 'var(--color-bg-secondary)', borderRadius: 'var(--radius-sm)' }}>
          <p style={{ opacity: 0.7 }}>No saved connections yet.</p>
          <p style={{ marginTop: '0.5rem', opacity: 0.5, fontSize: '0.875rem' }}>
            Create a connection first from the Play screen.
          </p>
        </div>
      ) : (
        <div style={{ marginTop: '1.5rem', display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
          {connections.map(conn => (
            <div 
              key={conn.id}
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                padding: '1rem',
                background: 'var(--color-bg-secondary)',
                border: '1px solid var(--color-border)',
                borderRadius: 'var(--radius-sm)',
              }}
            >
              <div>
                <div style={{ fontWeight: 'bold' }}>{conn.name}</div>
                <div style={{ fontSize: '0.875rem', opacity: 0.7 }}>{conn.host}:{conn.port}</div>
              </div>
              <Link 
                to={`/connections/${conn.id}/settings`}
                className="btn btn-secondary"
              >
                Settings
              </Link>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
