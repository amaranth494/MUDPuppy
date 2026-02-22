import { useState } from 'react';

const API_BASE = '/api/v1';

type AuthMode = 'login' | 'register' | 'otp';

export default function LoginScreen() {
  const [mode, setMode] = useState<AuthMode>('login');
  const [email, setEmail] = useState('');
  const [otp, setOtp] = useState('');
  const [message, setMessage] = useState('');
  const [messageType, setMessageType] = useState<'error' | 'success'>('error');
  const [isLoading, setIsLoading] = useState(false);
  const [pendingEmail, setPendingEmail] = useState('');

  const showMessage = (msg: string, type: 'error' | 'success') => {
    setMessage(msg);
    setMessageType(type);
  };

  const clearMessage = () => {
    setMessage('');
  };

  const handleLoginEmail = async (e: React.FormEvent) => {
    e.preventDefault();
    clearMessage();
    setIsLoading(true);

    try {
      const response = await fetch(`${API_BASE}/send-otp`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ email }),
      });

      const data = await response.json();

      if (response.ok) {
        showMessage('Verification code sent! Check your email.', 'success');
        setPendingEmail(email);
        setMode('otp');
      } else if (response.status === 404) {
        showMessage('Email not found. Please register first.', 'error');
      } else {
        showMessage(data.error || 'Failed to send code.', 'error');
      }
    } catch {
      showMessage('Network error. Please try again.', 'error');
    } finally {
      setIsLoading(false);
    }
  };

  const handleLoginOtp = async (e: React.FormEvent) => {
    e.preventDefault();
    clearMessage();
    setIsLoading(true);

    try {
      const response = await fetch(`${API_BASE}/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ email: pendingEmail, otp }),
      });

      const data = await response.json();

      if (response.ok) {
        window.location.reload();
      } else {
        showMessage(data.error || 'Invalid verification code.', 'error');
      }
    } catch {
      showMessage('Network error. Please try again.', 'error');
    } finally {
      setIsLoading(false);
    }
  };

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    clearMessage();
    setIsLoading(true);

    try {
      const response = await fetch(`${API_BASE}/register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ email }),
      });

      const data = await response.json();

      if (response.ok) {
        showMessage('Verification code sent! Check your email.', 'success');
        setPendingEmail(email);
        setMode('otp');
      } else {
        showMessage(data.error || 'Registration failed.', 'error');
      }
    } catch {
      showMessage('Network error. Please try again.', 'error');
    } finally {
      setIsLoading(false);
    }
  };

  const handleResendOtp = async () => {
    clearMessage();
    setIsLoading(true);

    try {
      const response = await fetch(`${API_BASE}/send-otp`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ email: pendingEmail }),
      });

      const data = await response.json();

      if (response.ok) {
        showMessage('New verification code sent!', 'success');
      } else {
        showMessage(data.error || 'Failed to resend code.', 'error');
      }
    } catch {
      showMessage('Network error. Please try again.', 'error');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div style={{
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      minHeight: '100vh',
      padding: '2rem',
    }}>
      <img src="/mudpuppy_logo.png" alt="MUDPuppy" style={{ height: '80px', marginBottom: '1rem' }} />
      <h1 style={{ marginBottom: '0.5rem' }}>MUDPuppy</h1>
      <p style={{ opacity: 0.7, marginBottom: '2rem' }}>A Legendary MUD Client</p>

      <div style={{
        background: '#111',
        border: '1px solid #00ff00',
        borderRadius: '8px',
        padding: '2rem',
        width: '100%',
        maxWidth: '400px',
      }}>
        {mode === 'login' && (
          <>
            <h2 style={{ marginBottom: '1.5rem', textAlign: 'center' }}>Login</h2>
            
            {message && (
              <div className={`message message-${messageType}`}>
                {message}
              </div>
            )}

            <form onSubmit={handleLoginEmail}>
              <div className="form-group">
                <label className="form-label">Email Address</label>
                <input
                  type="email"
                  className="form-input"
                  placeholder="you@example.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                />
              </div>
              <button type="submit" className="btn btn-primary" disabled={isLoading}>
                {isLoading ? 'Sending...' : 'Send One-Time Password'}
              </button>
            </form>

            <div style={{ marginTop: '1.5rem', textAlign: 'center' }}>
              <p style={{ fontSize: '0.9rem' }}>
                Don't have an account?{' '}
                <span
                  style={{ color: '#00ff00', textDecoration: 'underline', cursor: 'pointer' }}
                  onClick={() => { setMode('register'); clearMessage(); }}
                >
                  Register
                </span>
              </p>
            </div>
          </>
        )}

        {mode === 'register' && (
          <>
            <h2 style={{ marginBottom: '1.5rem', textAlign: 'center' }}>Create Account</h2>
            
            {message && (
              <div className={`message message-${messageType}`}>
                {message}
              </div>
            )}

            <form onSubmit={handleRegister}>
              <div className="form-group">
                <label className="form-label">Email Address</label>
                <input
                  type="email"
                  className="form-input"
                  placeholder="you@example.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                />
              </div>
              <p style={{ fontSize: '0.8rem', opacity: 0.7, marginBottom: '1.5rem' }}>
                We'll send a verification code to your email. No password required!
              </p>
              <button type="submit" className="btn btn-primary" disabled={isLoading}>
                {isLoading ? 'Sending...' : 'Send Code'}
              </button>
            </form>

            <div style={{ marginTop: '1.5rem', textAlign: 'center' }}>
              <p style={{ fontSize: '0.9rem' }}>
                Already have an account?{' '}
                <span
                  style={{ color: '#00ff00', textDecoration: 'underline', cursor: 'pointer' }}
                  onClick={() => { setMode('login'); clearMessage(); setEmail(''); }}
                >
                  Login
                </span>
              </p>
            </div>
          </>
        )}

        {mode === 'otp' && (
          <>
            <h2 style={{ marginBottom: '1.5rem', textAlign: 'center' }}>Verify Your Email</h2>
            
            <p style={{ fontSize: '0.9rem', marginBottom: '1rem', opacity: 0.8 }}>
              Verifying: <span style={{ color: '#00ff00' }}>{pendingEmail}</span>
            </p>

            {message && (
              <div className={`message message-${messageType}`}>
                {message}
              </div>
            )}

            <form onSubmit={handleLoginOtp}>
              <div className="form-group">
                <label className="form-label">Verification Code</label>
                <input
                  type="text"
                  className="form-input"
                  placeholder="000000"
                  maxLength={6}
                  value={otp}
                  onChange={(e) => setOtp(e.target.value)}
                  required
                  style={{ letterSpacing: '0.5rem', fontSize: '1.5rem', textAlign: 'center' }}
                />
              </div>
              <button type="submit" className="btn btn-primary" disabled={isLoading}>
                {isLoading ? 'Verifying...' : 'Verify'}
              </button>
            </form>

            <div style={{ marginTop: '1rem', textAlign: 'center' }}>
              <p style={{ fontSize: '0.9rem' }}>
                Didn't receive the code?{' '}
                <span
                  style={{ color: '#00ff00', textDecoration: 'underline', cursor: 'pointer' }}
                  onClick={handleResendOtp}
                >
                  Resend
                </span>
              </p>
            </div>

            <div style={{ marginTop: '0.5rem', textAlign: 'center' }}>
              <p style={{ fontSize: '0.9rem' }}>
                <span
                  style={{ color: '#00ff00', textDecoration: 'underline', cursor: 'pointer' }}
                  onClick={() => { setMode('login'); clearMessage(); setEmail(''); setOtp(''); }}
                >
                  Use different email
                </span>
              </p>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
