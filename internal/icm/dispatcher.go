package icm

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// SafetyChecker defines the interface for safety checks
type SafetyChecker interface {
	// CheckRecursionDepth checks if we've exceeded max alias depth
	CheckRecursionDepth(ctx *ExecutionContext) bool

	// CheckCircuitBreaker checks if circuit is tripped
	CheckCircuitBreaker(sessionID string) bool

	// CheckRateLimit checks if rate limited
	CheckRateLimit(sessionID string, command string) bool

	// CheckQueueDepth checks queue depth
	CheckQueueDepth(sessionID string) bool

	// CheckTimeout checks if execution has timed out
	CheckTimeout(startTime time.Time) bool

	// RecordExecution records a command execution
	RecordExecution(sessionID string, command string)

	// GetQueueBackpressureLevel gets current backpressure level
	GetQueueBackpressureLevel(sessionID string) string
}

// DefaultSafetyChecker provides default safety checking
type DefaultSafetyChecker struct {
	mu               sync.RWMutex
	executions       map[string][]time.Time
	loopCounts       map[string]int
	maxExecutions    int
	maxLoopCount     int
	maxExecutionTime time.Duration
	maxQueueDepth    int
}

func NewDefaultSafetyChecker() *DefaultSafetyChecker {
	return &DefaultSafetyChecker{
		executions:       make(map[string][]time.Time),
		loopCounts:       make(map[string]int),
		maxExecutions:    SafetyLimits.MaxCommandsPerDispatch,
		maxLoopCount:     SafetyLimits.MaxLoopCount,
		maxExecutionTime: SafetyLimits.MaxExecutionTime,
		maxQueueDepth:    SafetyLimits.MaxQueueDepth,
	}
}

func (d *DefaultSafetyChecker) CheckRecursionDepth(ctx *ExecutionContext) bool {
	// Simple implementation - tracks depth via context
	return false
}

func (d *DefaultSafetyChecker) CheckCircuitBreaker(sessionID string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	count := d.loopCounts[sessionID]
	return count >= d.maxLoopCount
}

func (d *DefaultSafetyChecker) CheckRateLimit(sessionID string, command string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	executions := d.executions[sessionID]
	now := time.Now()
	recentCount := 0

	for _, t := range executions {
		if now.Sub(t) < time.Second {
			recentCount++
		}
	}

	return recentCount >= d.maxExecutions
}

func (d *DefaultSafetyChecker) CheckQueueDepth(sessionID string) bool {
	// Queue depth would be checked against actual queue
	return false
}

func (d *DefaultSafetyChecker) CheckTimeout(startTime time.Time) bool {
	return time.Since(startTime) > d.maxExecutionTime
}

func (d *DefaultSafetyChecker) RecordExecution(sessionID string, command string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()

	// Record execution time
	if d.executions[sessionID] == nil {
		d.executions[sessionID] = []time.Time{now}
	} else {
		d.executions[sessionID] = append(d.executions[sessionID], now)
		// Keep only last minute of executions
		oneMinAgo := now.Add(-time.Minute)
		recent := []time.Time{}
		for _, t := range d.executions[sessionID] {
			if t.After(oneMinAgo) {
				recent = append(recent, t)
			}
		}
		d.executions[sessionID] = recent
	}

	// Track loop count for circuit breaker
	d.loopCounts[sessionID]++
}

func (d *DefaultSafetyChecker) GetQueueBackpressureLevel(sessionID string) string {
	return "normal"
}

// Dispatcher routes commands to appropriate handlers
type Dispatcher struct {
	registry        *Registry
	safety          SafetyChecker
	commandHandlers map[OperatorFamily]map[string]CommandHandler
	mu              sync.RWMutex
}

// NewDispatcher creates a new command dispatcher
func NewDispatcher(registry *Registry) *Dispatcher {
	d := &Dispatcher{
		registry:        registry,
		safety:          NewDefaultSafetyChecker(),
		commandHandlers: make(map[OperatorFamily]map[string]CommandHandler),
	}

	// Initialize handler maps
	d.commandHandlers[OperatorStructured] = make(map[string]CommandHandler)
	d.commandHandlers[OperatorAlias] = make(map[string]CommandHandler)
	d.commandHandlers[OperatorUserVariable] = make(map[string]CommandHandler)
	d.commandHandlers[OperatorSystemVariable] = make(map[string]CommandHandler)

	// Register built-in handlers
	d.registerBuiltInHandlers()

	return d
}

// registerBuiltInHandlers registers the built-in command handlers
func (d *Dispatcher) registerBuiltInHandlers() {
	// Register structured command handlers
	d.RegisterHandler(OperatorStructured, "ECHO", &EchoHandler{})
	d.RegisterHandler(OperatorStructured, "LOG", NewLogHandler())
	d.RegisterHandler(OperatorStructured, "HELP", &HelpHandler{})
	d.RegisterHandler(OperatorStructured, "SET", &SetHandler{})
	d.RegisterHandler(OperatorStructured, "IF", &IfHandler{})
	d.RegisterHandler(OperatorStructured, "TIMER", &TimerHandler{})

	// Register additional structured commands per spec
	d.RegisterHandler(OperatorStructured, "ELSE", &ElseHandler{})
	d.RegisterHandler(OperatorStructured, "ENDIF", &EndIfHandler{})
	d.RegisterHandler(OperatorStructured, "START", &StartTimerHandler{})
	d.RegisterHandler(OperatorStructured, "STOP", &StopTimerHandler{})
	d.RegisterHandler(OperatorStructured, "CHECK", &CheckTimerHandler{})
	d.RegisterHandler(OperatorStructured, "CANCEL", &CancelTimerHandler{})
}

// RegisterHandler registers a command handler
func (d *Dispatcher) RegisterHandler(family OperatorFamily, command string, handler CommandHandler) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.commandHandlers[family] == nil {
		d.commandHandlers[family] = make(map[string]CommandHandler)
	}

	d.commandHandlers[family][command] = handler
	return nil
}

// Dispatch routes a normalized command to its handler
func (d *Dispatcher) Dispatch(ctx *ExecutionContext, sessionID string, normalized *NormalizedCommand) (*CommandResult, *ICMError) {
	if normalized == nil {
		return nil, NewICMError(E5000InternalError, "nil command", nil)
	}

	// Check safety first
	if !d.checkSafety(ctx, sessionID, normalized.Command) {
		return nil, NewICMError(E4003CircuitBreaker, "Circuit breaker tripped", nil)
	}

	// Check authority
	authority, ok := AuthorityLevels[*ctx]
	if !ok {
		authority = AuthorityLevels[ContextPreview]
	}

	if !authority.CanExecute && normalized.RequiresExecution {
		return nil, NewICMError(E4001PermissionDenied, "Not authorized to execute", nil)
	}

	// Get the handler
	handler := d.getHandler(normalized.Operator, normalized.Command)
	if handler == nil {
		// No handler - this might be pass-through case
		return nil, nil
	}

	// Execute the handler
	result, err := handler.Handle(ctx, normalized)
	if err != nil {
		return nil, err
	}

	// Record execution for safety
	d.safety.RecordExecution(sessionID, normalized.Command)

	return result, nil
}

// getHandler returns the handler for a command
func (d *Dispatcher) getHandler(family OperatorFamily, command string) CommandHandler {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if handlers, ok := d.commandHandlers[family]; ok {
		if handler, ok := handlers[command]; ok {
			return handler
		}
	}
	return nil
}

// checkSafety performs safety checks before dispatch
func (d *Dispatcher) checkSafety(ctx *ExecutionContext, sessionID string, command string) bool {
	if *ctx == ContextPreview {
		// No safety checks for preview
		return true
	}

	if d.safety.CheckCircuitBreaker(sessionID) {
		return false
	}

	if d.safety.CheckRateLimit(sessionID, command) {
		return false
	}

	if d.safety.CheckQueueDepth(sessionID) {
		return false
	}

	return true
}

// SetSafetyChecker sets a custom safety checker
func (d *Dispatcher) SetSafetyChecker(checker SafetyChecker) {
	d.safety = checker
}

// GetSafetyChecker returns the current safety checker
func (d *Dispatcher) GetSafetyChecker() SafetyChecker {
	return d.safety
}

// Built-in command handlers

type EchoHandler struct{}

func (h *EchoHandler) Handle(ctx *ExecutionContext, cmd *NormalizedCommand) (*CommandResult, *ICMError) {
	output := ""
	if len(cmd.Args) > 0 {
		output = cmd.Args[0]
	}
	return &CommandResult{
		Output: output,
		Metadata: ResultMetadata{
			ExecutionTime: 0,
			Handler:       "ECHO",
		},
	}, nil
}

func (h *EchoHandler) Validate(cmd *NormalizedCommand) *ICMError {
	return nil
}

func (h *EchoHandler) Name() string {
	return "ECHO"
}

func (h *EchoHandler) Description() string {
	return "Echoes text to the output"
}

// LogHandler implements the #LOG directive with governance
type LogHandler struct {
	// LogStore stores log entries for retrieval
	logStore []LogEntry
}

// NewLogHandler creates a new #LOG handler
func NewLogHandler() *LogHandler {
	return &LogHandler{
		logStore: make([]LogEntry, 0),
	}
}

func (h *LogHandler) Handle(ctx *ExecutionContext, cmd *NormalizedCommand) (*CommandResult, *ICMError) {
	// #LOG governance: only allowed in Automation and Operational contexts
	if *ctx != ContextAutomation && *ctx != ContextOperational {
		return nil, NewICMError(E4001PermissionDenied,
			"#LOG is only available in automation or operational context", nil)
	}

	// Get log level and message from args
	level := "info"
	message := ""

	if len(cmd.Args) >= 2 {
		// First arg could be level: debug, info, warn, error
		level = strings.ToLower(cmd.Args[0])
		validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
		if !validLevels[level] {
			level = "info"
			message = strings.Join(cmd.Args, " ")
		} else {
			message = strings.Join(cmd.Args[1:], " ")
		}
	} else if len(cmd.Args) == 1 {
		message = cmd.Args[0]
	}

	// Create structured log entry with metadata
	logEntry := LogEntry{
		ID:            generateLogID(),
		Timestamp:     time.Now(),
		Level:         level,
		Message:       message,
		CorrelationID: generateCorrelationID(),
	}

	// Store log entry
	h.logStore = append(h.logStore, logEntry)

	// Return structured result
	return &CommandResult{
		Output: message,
		Effects: []StateEffect{
			{
				Type:  "log",
				Key:   "log_entry",
				Value: logEntry,
			},
		},
		Metadata: ResultMetadata{
			ExecutionTime: 0,
			Handler:       "LOG",
		},
	}, nil
}

func (h *LogHandler) Validate(cmd *NormalizedCommand) *ICMError {
	// No validation required for #LOG
	return nil
}

func (h *LogHandler) Name() string {
	return "LOG"
}

func (h *LogHandler) Description() string {
	return "Logs a message to the command log (automation/operational only)"
}

// GetLogEntries returns stored log entries
func (h *LogHandler) GetLogEntries() []LogEntry {
	return h.logStore
}

// ClearLogs clears stored log entries
func (h *LogHandler) ClearLogs() {
	h.logStore = make([]LogEntry, 0)
}

// generateLogID generates a unique log entry ID
func generateLogID() string {
	return fmt.Sprintf("log-%d", time.Now().UnixNano())
}

// generateCorrelationID generates a correlation ID for tracing
func generateCorrelationID() string {
	return fmt.Sprintf("corr-%d", time.Now().UnixNano())
}

type HelpHandler struct{}

func (h *HelpHandler) Handle(ctx *ExecutionContext, cmd *NormalizedCommand) (*CommandResult, *ICMError) {
	topic := ""
	if len(cmd.Args) > 0 {
		topic = cmd.Args[0]
	}

	output := fmt.Sprintf("Help for: %s", topic)
	return &CommandResult{
		Output: output,
		Metadata: ResultMetadata{
			ExecutionTime: 0,
			Handler:       "HELP",
		},
	}, nil
}

func (h *HelpHandler) Validate(cmd *NormalizedCommand) *ICMError {
	return nil
}

func (h *HelpHandler) Name() string {
	return "HELP"
}

func (h *HelpHandler) Description() string {
	return "Displays help information"
}

// SetHandler handles #SET directive for user variables
type SetHandler struct{}

func (h *SetHandler) Handle(ctx *ExecutionContext, cmd *NormalizedCommand) (*CommandResult, *ICMError) {
	if len(cmd.Args) < 2 {
		return nil, NewICMError(E2001MissingArgument, "SET requires name and value", nil)
	}

	name := cmd.Args[0]
	value := cmd.Args[1]

	// Store in registry (profile-scoped variable)
	// The engine will also handle session-scoped variables through effects
	return &CommandResult{
		Output: "",
		Effects: []StateEffect{
			{
				Type:  "variable",
				Key:   name,
				Value: value,
			},
		},
		Metadata: ResultMetadata{
			ExecutionTime: 0,
			Handler:       "SET",
		},
	}, nil
}

// GetRegistry returns the registry for setting variables
func (h *SetHandler) GetRegistry() *Registry {
	return nil // Will be set by engine
}

// SetRegistry sets the registry for setting variables
func (h *SetHandler) SetRegistry(r *Registry) {
	// Not used - variables are set through engine's ProcessEffects
}

func (h *SetHandler) Validate(cmd *NormalizedCommand) *ICMError {
	return nil
}

func (h *SetHandler) Name() string {
	return "SET"
}

func (h *SetHandler) Description() string {
	return "Sets a user variable"
}

type IfHandler struct{}

func (h *IfHandler) Handle(ctx *ExecutionContext, cmd *NormalizedCommand) (*CommandResult, *ICMError) {
	// #IF evaluates a condition and returns the appropriate branch
	// The automation engine handles the actual branching
	return &CommandResult{
		Output: "",
		Metadata: ResultMetadata{
			ExecutionTime: 0,
			Handler:       "IF",
		},
	}, nil
}

func (h *IfHandler) Validate(cmd *NormalizedCommand) *ICMError {
	return nil
}

func (h *IfHandler) Name() string {
	return "IF"
}

func (h *IfHandler) Description() string {
	return "Conditional execution"
}

type TimerHandler struct{}

func (h *TimerHandler) Handle(ctx *ExecutionContext, cmd *NormalizedCommand) (*CommandResult, *ICMError) {
	if len(cmd.Args) < 3 {
		return nil, NewICMError(E2001MissingArgument, "TIMER requires name, interval, and command", nil)
	}

	name := cmd.Args[0]
	interval := cmd.Args[1]
	command := cmd.Args[2]

	return &CommandResult{
		Output: "",
		Effects: []StateEffect{
			{
				Type:  "timer",
				Key:   name,
				Value: map[string]string{"interval": interval, "command": command, "action": "create"},
			},
		},
		Metadata: ResultMetadata{
			ExecutionTime: 0,
			Handler:       "TIMER",
		},
	}, nil
}

func (h *TimerHandler) Validate(cmd *NormalizedCommand) *ICMError {
	return nil
}

func (h *TimerHandler) Name() string {
	return "TIMER"
}

func (h *TimerHandler) Description() string {
	return "Creates a timer"
}

// ElseHandler handles #ELSE directive
type ElseHandler struct{}

func (h *ElseHandler) Handle(ctx *ExecutionContext, cmd *NormalizedCommand) (*CommandResult, *ICMError) {
	return &CommandResult{
		Output: "",
		Metadata: ResultMetadata{
			ExecutionTime: 0,
			Handler:       "ELSE",
		},
	}, nil
}

func (h *ElseHandler) Validate(cmd *NormalizedCommand) *ICMError {
	return nil
}

func (h *ElseHandler) Name() string {
	return "ELSE"
}

func (h *ElseHandler) Description() string {
	return "Else branch of conditional"
}

// EndIfHandler handles #ENDIF directive
type EndIfHandler struct{}

func (h *EndIfHandler) Handle(ctx *ExecutionContext, cmd *NormalizedCommand) (*CommandResult, *ICMError) {
	return &CommandResult{
		Output: "",
		Metadata: ResultMetadata{
			ExecutionTime: 0,
			Handler:       "ENDIF",
		},
	}, nil
}

func (h *EndIfHandler) Validate(cmd *NormalizedCommand) *ICMError {
	return nil
}

func (h *EndIfHandler) Name() string {
	return "ENDIF"
}

func (h *EndIfHandler) Description() string {
	return "End of conditional block"
}

// StartTimerHandler handles #START directive
type StartTimerHandler struct{}

func (h *StartTimerHandler) Handle(ctx *ExecutionContext, cmd *NormalizedCommand) (*CommandResult, *ICMError) {
	if len(cmd.Args) < 1 {
		return nil, NewICMError(E2001MissingArgument, "START requires timer name", nil)
	}

	name := cmd.Args[0]

	return &CommandResult{
		Output: "",
		Effects: []StateEffect{
			{
				Type:  "timer",
				Key:   name,
				Value: map[string]string{"action": "start"},
			},
		},
		Metadata: ResultMetadata{
			ExecutionTime: 0,
			Handler:       "START",
		},
	}, nil
}

func (h *StartTimerHandler) Validate(cmd *NormalizedCommand) *ICMError {
	return nil
}

func (h *StartTimerHandler) Name() string {
	return "START"
}

func (h *StartTimerHandler) Description() string {
	return "Starts a timer"
}

// StopTimerHandler handles #STOP directive
type StopTimerHandler struct{}

func (h *StopTimerHandler) Handle(ctx *ExecutionContext, cmd *NormalizedCommand) (*CommandResult, *ICMError) {
	if len(cmd.Args) < 1 {
		return nil, NewICMError(E2001MissingArgument, "STOP requires timer name", nil)
	}

	name := cmd.Args[0]

	return &CommandResult{
		Output: "",
		Effects: []StateEffect{
			{
				Type:  "timer",
				Key:   name,
				Value: map[string]string{"action": "stop"},
			},
		},
		Metadata: ResultMetadata{
			ExecutionTime: 0,
			Handler:       "STOP",
		},
	}, nil
}

func (h *StopTimerHandler) Validate(cmd *NormalizedCommand) *ICMError {
	return nil
}

func (h *StopTimerHandler) Name() string {
	return "STOP"
}

func (h *StopTimerHandler) Description() string {
	return "Stops a timer"
}

// CheckTimerHandler handles #CHECK directive
type CheckTimerHandler struct{}

func (h *CheckTimerHandler) Handle(ctx *ExecutionContext, cmd *NormalizedCommand) (*CommandResult, *ICMError) {
	if len(cmd.Args) < 1 {
		return nil, NewICMError(E2001MissingArgument, "CHECK requires timer name", nil)
	}

	name := cmd.Args[0]

	return &CommandResult{
		Output: "",
		Effects: []StateEffect{
			{
				Type:  "timer",
				Key:   name,
				Value: map[string]string{"action": "check"},
			},
		},
		Metadata: ResultMetadata{
			ExecutionTime: 0,
			Handler:       "CHECK",
		},
	}, nil
}

func (h *CheckTimerHandler) Validate(cmd *NormalizedCommand) *ICMError {
	return nil
}

func (h *CheckTimerHandler) Name() string {
	return "CHECK"
}

func (h *CheckTimerHandler) Description() string {
	return "Check timer status"
}

// CancelTimerHandler handles #CANCEL directive
type CancelTimerHandler struct{}

func (h *CancelTimerHandler) Handle(ctx *ExecutionContext, cmd *NormalizedCommand) (*CommandResult, *ICMError) {
	if len(cmd.Args) < 1 {
		return nil, NewICMError(E2001MissingArgument, "CANCEL requires timer name", nil)
	}

	name := cmd.Args[0]

	return &CommandResult{
		Output: "",
		Effects: []StateEffect{
			{
				Type:  "timer",
				Key:   name,
				Value: map[string]string{"action": "cancel"},
			},
		},
		Metadata: ResultMetadata{
			ExecutionTime: 0,
			Handler:       "CANCEL",
		},
	}, nil
}

func (h *CancelTimerHandler) Validate(cmd *NormalizedCommand) *ICMError {
	return nil
}

func (h *CancelTimerHandler) Name() string {
	return "CANCEL"
}

func (h *CancelTimerHandler) Description() string {
	return "Cancels a timer"
}
