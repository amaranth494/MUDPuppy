import { useState, useEffect, useRef } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import { useSession } from '../context/SessionContext';

// Enable text selection in terminal
const terminalSelectionStyle = {
  userSelect: 'text' as const,
  WebkitUserSelect: 'text' as const,
};

export default function PlayScreen() {
  const { 
    connectionState, 
    error, 
    connect, 
    disconnect,
    wsManager
  } = useSession();

  const [inputHost, setInputHost] = useState('');
  const [inputPort, setInputPort] = useState(23);
  const [isConnecting, setIsConnecting] = useState(false);
  
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
    terminal.writeln('Enter a host and port to connect to a MUD server.');
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

  const handleConnect = async () => {
    if (!inputHost.trim()) return;
    
    setIsConnecting(true);
    try {
      await connect(inputHost.trim(), inputPort);
      // Note: MUD output arrives via WebSocket - no need for "Connecting..." message
      // The button already shows "Connecting..." state
    } catch (err) {
      // Error is handled by context
    } finally {
      setIsConnecting(false);
    }
  };

  const handleDisconnect = async () => {
    try {
      await disconnect();
      if (terminalInstanceRef.current) {
        terminalInstanceRef.current.writeln('\r\n[Disconnected by user]\r\n');
      }
    } catch (err) {
      // Error is handled by context
    }
  };

  const handleCommand = (command: string) => {
    if (wsManager && connectionState === 'connected') {
      wsManager.sendCommand(command + '\n');
      // Echo command locally
      if (terminalInstanceRef.current) {
        terminalInstanceRef.current.write(command + '\r\n');
      }
    }
  };

  // State machine for UI
  const canConnect = (connectionState === 'disconnected' || connectionState === 'error') && !isConnecting;
  const canDisconnect = connectionState === 'connected' || connectionState === 'error';
  const isConnectingState = connectionState === 'connecting' || isConnecting;
  const showError = error && connectionState === 'error';

  return (
    <div className="play-screen">
      {/* Connection Panel */}
      <div className="connection-panel">
        <div className="form-group">
          <label className="form-label">Host</label>
          <input
            type="text"
            className="form-input"
            placeholder="e.g., example.com"
            value={inputHost}
            onChange={(e) => setInputHost(e.target.value)}
            disabled={!canConnect}
          />
        </div>
        
        <div className="form-group">
          <label className="form-label">Port</label>
          <input
            type="number"
            className="form-input"
            placeholder="23"
            value={inputPort}
            onChange={(e) => setInputPort(parseInt(e.target.value) || 23)}
            disabled={!canConnect}
          />
        </div>
        
        {canConnect && (
          <button 
            className="btn btn-primary"
            onClick={handleConnect}
            disabled={!inputHost.trim() || isConnectingState}
          >
            Connect
          </button>
        )}
        
        {canDisconnect && (
          <button 
            className="btn btn-danger"
            onClick={handleDisconnect}
          >
            Disconnect
          </button>
        )}
        
        {isConnectingState && (
          <button className="btn btn-secondary" disabled>
            Connecting...
          </button>
        )}
      </div>

      {/* Error Message */}
      {showError && (
        <div className="message message-error">
          {error}
        </div>
      )}

      {/* Output Panel */}
      <div className="output-panel">
        <div className="terminal-container" ref={terminalRef} />
      </div>

      {/* Command Input */}
      <div className="input-row">
        <input
          type="text"
          className="form-input"
          placeholder={connectionState === 'connected' ? 'Type command and press Enter...' : 'Connect to a MUD server first'}
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
