import { useEffect, useRef, useState } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import { useSession } from '../context/SessionContext';
import { getRecentConnections } from '../services/api';
import { SavedConnection } from '../types';

// Enable text selection in terminal
const terminalSelectionStyle = {
  userSelect: 'text' as const,
  WebkitUserSelect: 'text' as const,
};

export default function PlayScreen() {
  const { 
    connectionState, 
    wsManager,
    isInputLocked,
    connect,
    error,
  } = useSession();
  
  // Recent connections state
  const [recentConnections, setRecentConnections] = useState<SavedConnection[]>([]);
  const [isLoadingRecent, setIsLoadingRecent] = useState(false);
  
  // Quick connect form state
  const [quickHost, setQuickHost] = useState('');
  const [quickPort, setQuickPort] = useState(23);
  const [isConnecting, setIsConnecting] = useState(false);
  const [connectError, setConnectError] = useState<string | null>(null);
  
  // Load recent connections
  useEffect(() => {
    const loadRecent = async () => {
      setIsLoadingRecent(true);
      try {
        const connections = await getRecentConnections();
        setRecentConnections(connections.slice(0, 5)); // Limit to 5
      } catch (err) {
        console.error('Failed to load recent connections:', err);
      } finally {
        setIsLoadingRecent(false);
      }
    };
    
    if (connectionState !== 'connected') {
      loadRecent();
    }
  }, [connectionState]);
  
  // Sync error from session context
  useEffect(() => {
    if (error) {
      setConnectError(error);
    } else {
      setConnectError(null);
    }
  }, [error]);
  
  const terminalRef = useRef<HTMLDivElement>(null);
  const terminalInstanceRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);

  // Initialize terminal
  useEffect(() => {
    if (!terminalRef.current) return;

    const terminal = new Terminal({
      cursorBlink: true,
      fontFamily: 'Courier New, monospace',
      fontSize: 14,
      theme: {
        background: '#0a0a0a',
        foreground: '#00ff00',
        cursor: '#00ff00',
      },
      allowTransparency: true,
    });

    // Apply selection style to terminal element
    if (terminalRef.current) {
      const xtermElement = terminalRef.current.querySelector('.xterm') as HTMLElement;
      if (xtermElement) {
        Object.assign(xtermElement.style, terminalSelectionStyle);
      }
    }

    const fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);
    terminal.open(terminalRef.current);
    fitAddon.fit();

    // Handle Ctrl+C / Cmd+C for clipboard copy when text is selected
    terminal.attachCustomKeyEventHandler((event: KeyboardEvent) => {
      // Check for Ctrl+C (Windows/Linux) or Cmd+C (Mac)
      const isCopyShortcut = (event.ctrlKey || event.metaKey) && event.key === 'c';
      
      if (isCopyShortcut) {
        const selection = terminal.getSelection();
        if (selection) {
          // Copy selection to clipboard
          navigator.clipboard.writeText(selection).catch(() => {
            // Fallback for clipboard permission issues
          });
          // Return false to prevent xterm from handling it (no ^C sent to MUD)
          return false;
        }
        // If no selection, let xterm handle it (though we prefer to do nothing)
        return false;
      }
      // Let all other keys pass through to xterm
      return true;
    });

    terminalInstanceRef.current = terminal;
    fitAddonRef.current = fitAddon;

    // Handle resize
    const handleResize = () => {
      fitAddon.fit();
    };
    window.addEventListener('resize', handleResize);

    // Initial message
    terminal.writeln('Welcome to MUDPuppy!');
    terminal.writeln('Click Play in the sidebar to connect to a MUD server.');
    terminal.writeln('');

    return () => {
      window.removeEventListener('resize', handleResize);
      terminal.dispose();
    };
  }, []);

  // Handle WebSocket messages
  useEffect(() => {
    if (!wsManager || !terminalInstanceRef.current) return;

    const terminal = terminalInstanceRef.current;

    const handleData = (data: string) => {
      terminal.write(data);
    };

    const handleError = (err: string) => {
      terminal.writeln(`\r\n[ERROR] ${err}\r\n`);
    };

    const handleDisconnect = () => {
      terminal.writeln('\r\n[Disconnected]\r\n');
    };

    wsManager.onMessage(handleData);
    wsManager.onError(handleError);
    wsManager.onDisconnect(handleDisconnect);

    return () => {
      // Handlers are automatically removed when component unmounts
    };
  }, [wsManager]);

  // Auto-fit terminal on window resize
  useEffect(() => {
    const handleResize = () => {
      if (fitAddonRef.current) {
        fitAddonRef.current.fit();
      }
    };
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  const handleCommand = (command: string) => {
    // SP03PH03: Gate command submission when modal is open
    // This prevents keystrokes from leaking to MUD while modal is active
    if (isInputLocked) {
      return;
    }
    
    if (wsManager && connectionState === 'connected') {
      wsManager.sendCommand(command + '\n');
      // Echo command locally
      if (terminalInstanceRef.current) {
        terminalInstanceRef.current.write(command + '\r\n');
      }
    }
  };

  // Quick connect from form
  const handleQuickConnect = async () => {
    if (!quickHost.trim()) {
      setConnectError('Host is required');
      return;
    }
    
    setIsConnecting(true);
    setConnectError(null);
    
    try {
      await connect(quickHost.trim(), quickPort);
    } catch (err) {
      setConnectError(err instanceof Error ? err.message : 'Connection failed');
    } finally {
      setIsConnecting(false);
    }
  };

  // Quick connect from recent connection
  const handleRecentConnect = async (conn: SavedConnection) => {
    setIsConnecting(true);
    setConnectError(null);
    
    try {
      await connect(conn.host, conn.port);
    } catch (err) {
      setConnectError(err instanceof Error ? err.message : 'Connection failed');
    } finally {
      setIsConnecting(false);
    }
  };

  const isDisconnected = connectionState === 'disconnected' || connectionState === 'error';
  
  return (
    <div className="play-screen">
      {/* Quick Connect Panel - shown when not connected */}
      {isDisconnected && (
        <div className="quick-connect-panel">
          <h3>Recent Servers</h3>
          {isLoadingRecent ? (
            <p className="loading-text">Loading...</p>
          ) : recentConnections.length > 0 ? (
            <div className="recent-connections">
              {recentConnections.map((conn) => (
                <button
                  key={conn.id}
                  className="recent-connection-btn"
                  onClick={() => handleRecentConnect(conn)}
                  disabled={isConnecting}
                >
                  <span className="conn-name">{conn.name}</span>
                  <span className="conn-address">{conn.host}:{conn.port}</span>
                </button>
              ))}
            </div>
          ) : (
            <p className="no-recent">No recent connections</p>
          )}
          
          <h3>Connect</h3>
          <div className="quick-connect-form">
            <div className="form-row">
              <input
                type="text"
                className="form-input"
                placeholder="Host (e.g., game.example.com)"
                value={quickHost}
                onChange={(e) => setQuickHost(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    handleQuickConnect();
                  }
                }}
              />
              <input
                type="number"
                className="form-input form-input-port"
                placeholder="Port"
                value={quickPort}
                onChange={(e) => setQuickPort(parseInt(e.target.value) || 23)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    handleQuickConnect();
                  }
                }}
              />
              <button
                className="btn btn-primary"
                onClick={handleQuickConnect}
                disabled={isConnecting}
              >
                {isConnecting ? 'Connecting...' : 'Connect'}
              </button>
            </div>
            {connectError && (
              <p className="error-text">{connectError}</p>
            )}
          </div>
        </div>
      )}

      {/* Output Panel - Full height terminal */}
      <div className="output-panel output-panel-full">
        <div className="terminal-container" ref={terminalRef} />
      </div>

      {/* Command Input */}
      <div className="input-row">
        <input
          type="text"
          className="form-input"
          placeholder={connectionState === 'connected' ? 'Type command and press Enter...' : 'Connect to a MUD server via Play in sidebar'}
          disabled={connectionState !== 'connected'}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              const input = e.currentTarget;
              const command = input.value; // Don't trim - allow blank lines for MUDs
              handleCommand(command); // Always send (even empty = just CR)
              input.value = '';
            }
          }}
        />
      </div>
    </div>
  );
}
