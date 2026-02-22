export default function HelpPage() {
  return (
    <div style={{ padding: '2rem', maxWidth: '800px', margin: '0 auto' }}>
      <h2>Help</h2>
      
      <div style={{ marginTop: '1.5rem' }}>
        <h3 style={{ marginBottom: '1rem' }}>Getting Started</h3>
        
        <div style={{ marginBottom: '1.5rem' }}>
          <h4 style={{ marginBottom: '0.5rem', opacity: 0.8 }}>1. Connect to a MUD</h4>
          <p style={{ opacity: 0.7, fontSize: '0.9rem' }}>
            On the Play screen, enter a MUD server hostname (like "example.com") 
            and port (default is 23). Click "Connect" to start playing.
          </p>
        </div>
        
        <div style={{ marginBottom: '1.5rem' }}>
          <h4 style={{ marginBottom: '0.5rem', opacity: 0.8 }}>2. Send Commands</h4>
          <p style={{ opacity: 0.7, fontSize: '0.9rem' }}>
            Once connected, type commands in the input field at the bottom 
            and press Enter to send them to the MUD server.
          </p>
        </div>
        
        <div style={{ marginBottom: '1.5rem' }}>
          <h4 style={{ marginBottom: '0.5rem', opacity: 0.8 }}>3. Disconnect</h4>
          <p style={{ opacity: 0.7, fontSize: '0.9rem' }}>
            Click "Disconnect" to end your session. You can also reconnect 
            at any time.
          </p>
        </div>
      </div>
      
      <div style={{ marginTop: '2rem' }}>
        <h3 style={{ marginBottom: '1rem' }}>Connection Limits</h3>
        
        <ul style={{ opacity: 0.7, fontSize: '0.9rem', paddingLeft: '1.5rem' }}>
          <li>One active connection per user at a time</li>
          <li>30-minute idle timeout (auto-disconnect after inactivity)</li>
          <li>24-hour maximum session duration</li>
          <li>Only port 23 (telnet) is allowed by default</li>
          <li>Private IP addresses are blocked for security</li>
        </ul>
      </div>
      
      <div style={{ marginTop: '2rem' }}>
        <h3 style={{ marginBottom: '1rem' }}>Troubleshooting</h3>
        
        <div style={{ marginBottom: '1rem' }}>
          <h4 style={{ marginBottom: '0.5rem', opacity: 0.8 }}>Connection Refused</h4>
          <p style={{ opacity: 0.7, fontSize: '0.9rem' }}>
            The MUD server may be down or the port may be incorrect. 
            Try a different port (common ones: 23, 8080, 7777).
          </p>
        </div>
        
        <div style={{ marginBottom: '1rem' }}>
          <h4 style={{ marginBottom: '0.5rem', opacity: 0.8 }}>Could Not Resolve Host</h4>
          <p style={{ opacity: 0.7, fontSize: '0.9rem' }}>
            Check that the hostname is correct. Some MUDs may have 
            temporary DNS issues.
          </p>
        </div>
        
        <div style={{ marginBottom: '1rem' }}>
          <h4 style={{ marginBottom: '0.5rem', opacity: 0.8 }}>Connection Timed Out</h4>
          <p style={{ opacity: 0.7, fontSize: '0.9rem' }}>
            The server took too long to respond. This can happen with 
            distant servers or during high load.
          </p>
        </div>
      </div>
    </div>
  );
}
