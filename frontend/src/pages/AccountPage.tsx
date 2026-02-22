import { useSession } from '../context/SessionContext';

export default function AccountPage() {
  const { user } = useSession();

  return (
    <div style={{ padding: '2rem', maxWidth: '600px', margin: '0 auto' }}>
      <h2>Account</h2>
      
      <div style={{ marginTop: '1.5rem' }}>
        <div className="form-group">
          <label className="form-label">Email</label>
          <input
            type="email"
            className="form-input"
            value={user?.email || ''}
            disabled
          />
        </div>
        
        <div className="form-group">
          <label className="form-label">Account ID</label>
          <input
            type="text"
            className="form-input"
            value={user?.id || ''}
            disabled
          />
        </div>
      </div>
      
      <div style={{ marginTop: '2rem' }}>
        <button 
          className="btn btn-secondary"
          onClick={() => {
            window.location.href = '/';
          }}
        >
          Sign Out
        </button>
      </div>
    </div>
  );
}
