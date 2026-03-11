import { useEffect, useRef, useCallback, useState } from 'react';
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
    automationDisabled,
    resumeAutomation,
    disableAutomation,
    enableAutomation,
  } = useSession();
  
  // SP06PH08: Command history state
  const [commandHistory, setCommandHistory] = useState<string[]>([]);
  const [historyPosition, setHistoryPosition] = useState<number>(-1);
  
  const terminalRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const terminalInstanceRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  
  // SP06PH08: Reset command history when disconnected
  useEffect(() => {
    if (connectionState === 'disconnected') {
      setCommandHistory([]);
      setHistoryPosition(-1);
    }
  }, [connectionState]);
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

    const fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);
    terminal.open(terminalRef.current);
    
    // [SP03PH05T06] Debug: Apply selection style AFTER terminal.open()
    if (terminalRef.current) {
      const xtermElement = terminalRef.current.querySelector('.xterm') as HTMLElement;
      console.log('[SP03PH05T06] xtermElement found:', !!xtermElement);
      if (xtermElement) {
        Object.assign(xtermElement.style, terminalSelectionStyle);
        console.log('[SP03PH05T06] Applied selection style');
      }
    }

    // Custom fit function that aligns to row height (16px) to prevent cut-off
    const customFit = () => {
      if (!terminalInstanceRef.current || !terminalRef.current || !fitAddonRef.current) return;
      
      const terminal = terminalInstanceRef.current;
      const container = terminalRef.current;
      
      // Get the viewport element to determine available height
      const viewport = container.querySelector('.xterm-viewport') as HTMLElement;
      if (!viewport) return;
      
      // Get container/viewport dimensions
      const viewportHeight = viewport.clientHeight;
      const viewportWidth = viewport.clientWidth;
      
      // Character dimensions
      const charWidth = 8;
      const charHeight = 16;
      
      // Calculate max rows that fit in viewport (floor to nearest multiple of 16)
      const maxRows = Math.floor(viewportHeight / charHeight);
      const rows = maxRows;
      
      // Calculate max cols that fit
      const maxCols = Math.floor(viewportWidth / charWidth);
      
      // Resize terminal to exact fit (aligned to row height)
      terminal.resize(maxCols, rows);
    };

    terminalInstanceRef.current = terminal;
    fitAddonRef.current = fitAddon;
    
    // Sync xterm selection to browser selection so right-click menu works
    const syncSelectionToBrowser = () => {
      const selection = terminal.getSelection();
      if (selection && selection.length > 0) {
        // Create a hidden element with the selected text to sync browser selection
        let syncEl = document.getElementById('xterm-selection-sync');
        if (!syncEl) {
          syncEl = document.createElement('div');
          syncEl.id = 'xterm-selection-sync';
          syncEl.style.cssText = 'position:absolute;left:-9999px;white-space:pre-wrap;word-wrap:break-word;';
          document.body.appendChild(syncEl);
        }
        syncEl.textContent = selection;
        
        // Select the hidden element
        const range = document.createRange();
        range.selectNodeContents(syncEl);
        const browserSelection = window.getSelection();
        if (browserSelection) {
          browserSelection.removeAllRanges();
          browserSelection.addRange(range);
        }
      }
    };
    
    // Handle selection changes to sync with browser
    terminal.onSelectionChange(() => {
      syncSelectionToBrowser();
    });
    
    // Create custom context menu for right-click
    const contextMenu = document.createElement('div');
    contextMenu.id = 'terminal-context-menu';
    contextMenu.style.cssText = `
      position: fixed;
      background: #1a1a1a;
      border: 1px solid #444;
      border-radius: 6px;
      padding: 4px 0;
      z-index: 10000;
      display: none;
      box-shadow: 0 4px 12px rgba(0,0,0,0.5);
      min-width: 120px;
    `;
    
    const copyItem = document.createElement('div');
    copyItem.textContent = 'Copy';
    copyItem.style.cssText = `
      padding: 8px 16px;
      cursor: pointer;
      color: #eee;
      font-size: 13px;
      font-family: system-ui, sans-serif;
    `;
    copyItem.addEventListener('mouseenter', () => copyItem.style.background = '#444');
    copyItem.addEventListener('mouseleave', () => copyItem.style.background = 'transparent');
    copyItem.addEventListener('click', () => {
      const selection = terminal.getSelection();
      if (selection && selection.length > 0) {
        navigator.clipboard.writeText(selection);
      }
      contextMenu.style.display = 'none';
    });
    contextMenu.appendChild(copyItem);
    document.body.appendChild(contextMenu);
    
    const handleContextMenu = (e: MouseEvent) => {
      const selection = terminal.getSelection();
      if (selection && selection.length > 0) {
        e.preventDefault();
        contextMenu.style.display = 'block';
        contextMenu.style.left = e.clientX + 'px';
        contextMenu.style.top = e.clientY + 'px';
      }
    };
    
    const hideContextMenu = () => contextMenu.style.display = 'none';
    terminalRef.current.addEventListener('contextmenu', handleContextMenu);
    document.addEventListener('click', hideContextMenu);
    
    // Handle Ctrl+C globally on document level - works regardless of what's focused
    const handleGlobalKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 'c') {
        const selection = terminal.getSelection();
        if (selection && selection.length > 0) {
          e.preventDefault();
          navigator.clipboard.writeText(selection);
        }
      }
    };
    document.addEventListener('keydown', handleGlobalKeyDown);
    
    // Delay customFit to ensure container has final layout dimensions
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        customFit();
      });
    });

    // Handle resize
    const handleResize = () => {
      customFit();
    };
    window.addEventListener('resize', handleResize);

    // Initial message
    terminal.writeln('Welcome to MUDPuppy!');
    terminal.writeln('Click Play in the sidebar to connect to a MUD server.');
    terminal.writeln('');

    return () => {
      window.removeEventListener('resize', handleResize);
      document.removeEventListener('keydown', handleGlobalKeyDown);
      if (terminalRef.current) {
        terminalRef.current.removeEventListener('contextmenu', handleContextMenu);
      }
      document.removeEventListener('click', hideContextMenu);
      const menu = document.getElementById('terminal-context-menu');
      if (menu) menu.remove();
      const syncEl = document.getElementById('xterm-selection-sync');
      if (syncEl) syncEl.remove();
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
      // SP06PH07: Skip if automation is disabled
      if (automationEngine && command !== '' && !automationDisabled) {
        const processedCommands = automationEngine.processUserInput(command);
        
        // If no commands (e.g., circuit breaker tripped), skip
        if (processedCommands.length === 0) {
          return;
        }
        
        // Don't echo here - the automation engine's callback will echo 
        // the processed command(s) when they're actually sent
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
  }, [wsManager, connectionState, isInputLocked, profile, automationEngine, automationDisabled]);

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

      {/* SP05: Automation circuit breaker notification (SP06PH07: enhanced with Disable button) */}
      {(automationError || automationDisabled) && (
        <div className="automation-error-banner">
          {automationError ? (
            <>
              <span>⚠️ Automation Paused: {automationError}</span>
              <button onClick={resumeAutomation} className="btn btn-small">
                Resume Automation
              </button>
              <button onClick={disableAutomation} className="btn btn-small btn-secondary">
                Disable Automation
              </button>
            </>
          ) : (
            <>
              <span>⚠️ Automation Disabled</span>
              <button onClick={enableAutomation} className="btn btn-small">
                Enable Automation
              </button>
            </>
          )}
        </div>
      )}

      {/* Command Input */}
      <div className="input-row">
        <input
          ref={inputRef}
          type="text"
          className="form-input"
          placeholder={connectionState === 'connected' ? 'Type command and press Enter...' : 'Click Play in sidebar to connect'}
          disabled={connectionState !== 'connected'}
          onKeyDown={(e) => {
            const input = e.currentTarget;
            
            // SP03PH05T06: Handle Ctrl+C / Cmd+C - copy from terminal if text is selected there
            if ((e.ctrlKey || e.metaKey) && e.key === 'c') {
              console.log('[SP03PH05T06] Input field: Ctrl+C detected');
              // Try to copy from terminal selection
              if (terminalInstanceRef.current) {
                const selection = terminalInstanceRef.current.getSelection();
                console.log('[SP03PH05T06] Input field: terminal selection:', selection ? `"${selection.substring(0, 20)}..."` : 'empty');
                if (selection && selection.length > 0) {
                  e.preventDefault();
                  navigator.clipboard.writeText(selection);
                  return;
                }
              }
              // If no terminal selection, let default copy happen (from input)
            }
            
            // SP06PH08: Handle Up/Down arrows for command history
            if (e.key === 'ArrowUp') {
              e.preventDefault();
              // Don't navigate if already at the beginning
              if (commandHistory.length === 0) return;
              
              // If we're at -1 (not recalling), start from the last command
              // If we're already recalling, go back one more
              const newPosition = historyPosition === -1 
                ? commandHistory.length - 1 
                : Math.max(0, historyPosition - 1);
              
              setHistoryPosition(newPosition);
              input.value = commandHistory[newPosition];
              return;
            }
            
            if (e.key === 'ArrowDown') {
              e.preventDefault();
              // Don't navigate if not currently recalling
              if (historyPosition === -1) return;
              
              // Go forward one position
              const newPosition = historyPosition + 1;
              
              if (newPosition >= commandHistory.length) {
                // Gone past the end - reset to typing mode
                setHistoryPosition(-1);
                input.value = '';
              } else {
                setHistoryPosition(newPosition);
                input.value = commandHistory[newPosition];
              }
              return;
            }
            
            // SP06PH08: Reset history position when user types (not just recalling)
            if (e.key !== 'Enter') {
              if (historyPosition !== -1) {
                // User is typing while in recall mode - they're modifying a recalled command
                // Keep the current position but track that input has changed
              }
            }
            
            if (e.key === 'Enter') {
              const command = input.value; // Don't trim - allow blank lines for MUDs
              
              // SP06PH08: Add to history only if:
              // 1. Not in recall mode (historyPosition === -1)
              // 2. Command is not empty
              // 3. Command is different from last command in history (or history is empty)
              if (historyPosition === -1 && command !== '') {
                setCommandHistory(prev => {
                  // Only add if different from the last command
                  if (prev.length > 0 && prev[prev.length - 1] === command) {
                    return prev;
                  }
                  return [...prev, command];
                });
              }
              
              // Submit the command through the canonical pipeline
              submitCommand('typing', command);
              
              // Reset input and history position
              input.value = '';
              setHistoryPosition(-1);
            }
          }}
        />
      </div>
    </div>
  );
}
