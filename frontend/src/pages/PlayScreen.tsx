import { useEffect, useRef, useCallback } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import { useSession } from '../context/SessionContext';
import { useInputInterceptor } from '../hooks/useInputInterceptor';
import { ProfileSettings } from '../types';

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
    profile,
    automationEngine,
    automationError,
    resumeAutomation,
  } = useSession();
  
  const terminalRef = useRef<HTMLDivElement>(null);
  const terminalInstanceRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  
  // SP04PH05: Apply profile settings when profile changes
  // This effect applies settings when a connection is established
  useEffect(() => {
    if (!terminalInstanceRef.current || !profile) return;
    
    const terminal = terminalInstanceRef.current;
    const settings = profile.settings as ProfileSettings;
    
    // Apply scrollback limit (min 100, max 10000)
    const scrollbackLimit = Math.max(100, Math.min(10000, settings.scrollback_limit || 1000));
    terminal.options.scrollback = scrollbackLimit;
    
    // Apply word wrap (SP04PH05T05)
    // In xterm.js v5+, use options.wraparound to control word wrap
    (terminal.options as any).wraparound = settings.word_wrap ?? true;
    
  }, [profile]);

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
  // Use refs to store handlers and dependencies for stable references across renders
  const handleDataRef = useRef<((data: string) => void) | null>(null);
  const handleErrorRef = useRef<((err: string) => void) | null>(null);
  const handleDisconnectRef = useRef<(() => void) | null>(null);
  const automationEngineRef = useRef(automationEngine);
  const connectionStateRef = useRef(connectionState);
  
  useEffect(() => {
    if (!wsManager || !terminalInstanceRef.current) return;

    const terminal = terminalInstanceRef.current;
    
    // Buffer to collect complete lines for trigger processing
    let lineBuffer = '';

    // Update refs with current values
    automationEngineRef.current = automationEngine;
    connectionStateRef.current = connectionState;

    // Create handlers that access current values via refs
    handleDataRef.current = (data: string) => {
      // Write to terminal
      terminal.write(data);
      
      // SP05: Process through trigger engine (use ref for current values)
      if (automationEngineRef.current && connectionStateRef.current === 'connected') {
        // Collect data and split into lines
        lineBuffer += data;
        const lines = lineBuffer.split(/\r?\n/);
        // Keep the last incomplete line in buffer
        lineBuffer = lines.pop() || '';
        
        // Process complete lines through trigger engine
        if (lines.length > 0) {
          automationEngineRef.current.processServerOutput(lines);
        }
      }
    };

    handleErrorRef.current = (err: string) => {
      terminal.writeln(`\r\n[ERROR] ${err}\r\n`);
    };

    handleDisconnectRef.current = () => {
      terminal.writeln('\r\n[Disconnected]\r\n');
    };

    // Register handlers
    wsManager.onMessage(handleDataRef.current);
    wsManager.onError(handleErrorRef.current);
    wsManager.onDisconnect(handleDisconnectRef.current);

    // Clean up handlers on unmount or when dependencies change
    return () => {
      if (handleDataRef.current) {
        wsManager.offMessage(handleDataRef.current);
      }
      if (handleErrorRef.current) {
        wsManager.offError(handleErrorRef.current);
      }
      if (handleDisconnectRef.current) {
        wsManager.offDisconnect(handleDisconnectRef.current);
      }
    };
  }, [wsManager, automationEngine, connectionState]);

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

  // SP05: Set up automation engine callback for command submission
  useEffect(() => {
    if (automationEngine && wsManager) {
      // Set up the callback for automation to submit commands
      automationEngine.setSubmitCommandCallback((command: string) => {
        wsManager.sendCommand(command + '\n');
        
        // Echo the command (automation commands)
        const settings = profile?.settings;
        const shouldEcho = settings?.echo_input ?? true;
        if (shouldEcho && terminalInstanceRef.current) {
          let echoText = command + '\r\n';
          if (settings?.timestamp_output) {
            const now = new Date();
            const timestamp = now.toLocaleTimeString('en-US', { 
              hour12: false, 
              hour: '2-digit', 
              minute: '2-digit', 
              second: '2-digit' 
            });
            echoText = `[${timestamp}] ${command}\r\n`;
          }
          terminalInstanceRef.current.write(echoText);
        }
      });
    }
  }, [automationEngine, wsManager, profile]);

  // SP04: Single submitCommand function - canonical entry point for all commands
  // Both typing and keybindings route through this same function
  const submitCommand = useCallback((_source: 'typing' | 'keybinding', text: string) => {
    // SP03PH03: Gate command submission when modal is open
    if (isInputLocked) {
      return;
    }
    
    // Don't trim - allow blank lines for MUDs
    const command = text;
    
    if (wsManager && connectionState === 'connected') {
      // SP05: Process through automation engine (aliases, variables)
      if (automationEngine && command !== '') {
        const processedCommands = automationEngine.processUserInput(command);
        
        // If no commands (e.g., circuit breaker tripped), skip
        if (processedCommands.length === 0) {
          return;
        }
        
        // Do local echo for the original user input
        const settings = profile?.settings;
        const shouldEcho = settings?.echo_input ?? true;
        if (shouldEcho && terminalInstanceRef.current) {
          let echoText = command + '\r\n';
          if (settings?.timestamp_output) {
            const now = new Date();
            const timestamp = now.toLocaleTimeString('en-US', { 
              hour12: false, 
              hour: '2-digit', 
              minute: '2-digit', 
              second: '2-digit' 
            });
            echoText = `[${timestamp}] ${command}\r\n`;
          }
          terminalInstanceRef.current.write(echoText);
        }
      } else {
        // No automation engine OR empty command - send directly to WebSocket
        // Empty commands bypass automation (some MUDs need blank lines)
        wsManager.sendCommand(command + '\n');
        
        // SP04PH05: Local echo - controlled by profile settings.echo_input
        const settings = profile?.settings;
        const shouldEcho = settings?.echo_input ?? true;
        if (shouldEcho && terminalInstanceRef.current) {
          let echoText = command + '\r\n';
          if (settings?.timestamp_output) {
            const now = new Date();
            const timestamp = now.toLocaleTimeString('en-US', { 
              hour12: false, 
              hour: '2-digit', 
              minute: '2-digit', 
              second: '2-digit' 
            });
            echoText = `[${timestamp}] ${command}\r\n`;
          }
          terminalInstanceRef.current.write(echoText);
        }
      }
    }
  }, [wsManager, connectionState, isInputLocked, profile, automationEngine]);

  // SP04: Set up keybinding interceptor
  useInputInterceptor({
    onKeybindingDispatch: (command: string) => {
      // Route through submitCommand for consistent behavior
      submitCommand('keybinding', command);
    },
    enabled: connectionState === 'connected' && !isInputLocked,
  });
  
  return (
    <div className="play-screen">
      {/* Output Panel - Full height terminal */}
      <div className="output-panel output-panel-full">
        <div className="terminal-container" ref={terminalRef} />
      </div>

      {/* SP05: Automation circuit breaker notification */}
      {automationError && (
        <div className="automation-error-banner">
          <span>⚠️ Automation Paused: {automationError}</span>
          <button onClick={resumeAutomation} className="btn btn-small">
            Resume Automation
          </button>
        </div>
      )}

      {/* Command Input */}
      <div className="input-row">
        <input
          type="text"
          className="form-input"
          placeholder={connectionState === 'connected' ? 'Type command and press Enter...' : 'Click Play in sidebar to connect'}
          disabled={connectionState !== 'connected'}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              const input = e.currentTarget;
              const command = input.value; // Don't trim - allow blank lines for MUDs
              submitCommand('typing', command); // Use canonical submitCommand
              input.value = '';
            }
          }}
        />
      </div>
    </div>
  );
}
