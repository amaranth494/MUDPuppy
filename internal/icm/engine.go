package icm

import (
	"sync"
	"time"
)

// Engine is the main ICM engine that orchestrates the command processing pipeline
type Engine struct {
	// Components
	recognizer *Recognizer
	validator  *Validator
	normalizer *Normalizer
	registry   *Registry
	dispatcher *Dispatcher

	// Handlers
	escapeHandler         *EscapeHandler
	passThroughClassifier *PassThroughClassifier

	// State
	mu       sync.RWMutex
	handlers map[OperatorFamily]CommandHandler

	// Session state (per-session variables)
	sessions map[string]*ExecutionSession

	// Command history for debugging
	commandHistory map[string][]string

	// Effect handlers for state changes
	effectHandlers map[string]StateEffectHandler
}

// StateEffectHandler handles state changes from command execution
type StateEffectHandler interface {
	HandleEffect(effect StateEffect, sessionID string) error
	GetHandlerType() string
}

// NewEngine creates a new ICM engine
func NewEngine() *Engine {
	e := &Engine{
		recognizer:            NewRecognizer(),
		validator:             NewValidator(),
		normalizer:            NewNormalizer(),
		registry:              NewRegistry(),
		dispatcher:            nil, // Set after registry
		escapeHandler:         NewEscapeHandler(),
		passThroughClassifier: nil, // Set after registry
		handlers:              make(map[OperatorFamily]CommandHandler),
		sessions:              make(map[string]*ExecutionSession),
		commandHistory:        make(map[string][]string),
		effectHandlers:        make(map[string]StateEffectHandler),
	}

	// Initialize dispatcher with registry
	e.dispatcher = NewDispatcher(e.registry)

	// Initialize pass-through classifier
	e.passThroughClassifier = NewPassThroughClassifier(e.registry)

	return e
}

// Process processes a command through the ICM pipeline
func (e *Engine) Process(req *ICMRequest) *ICMResponse {
	startTime := time.Now()

	// Initialize response
	resp := &ICMResponse{
		ProcessingTime: 0,
	}

	// Validate request
	if req == nil || req.Raw == "" {
		resp.Error = NewICMError(E2000SyntaxError, "Empty input", nil)
		resp.ProcessingTime = time.Since(startTime).Milliseconds()
		return resp
	}

	// Phase 1: Recognize (Tokenize)
	recognizeResult, err := e.recognizer.Recognize(req.Raw)
	if err != nil {
		resp.Error = err
		resp.ProcessingTime = time.Since(startTime).Milliseconds()
		return resp
	}

	resp.IsInternal = recognizeResult.IsInternal
	resp.Operator = recognizeResult.Operator
	resp.Command = recognizeResult.Command

	// If not internal, handle pass-through
	if !recognizeResult.IsInternal {
		// Check if it should pass through to MUD
		classifyResult := e.passThroughClassifier.ClassifyCommand(req.Raw)
		if classifyResult.ShouldPassThrough {
			// Pass through to MUD server
			resp.IsInternal = false
			resp.Normalized = req.Raw
			resp.ProcessingTime = time.Since(startTime).Milliseconds()
			return resp
		}
	}

	// Phase 2: Validate
	if err := e.validator.Validate(recognizeResult); err != nil {
		resp.Error = err
		resp.ProcessingTime = time.Since(startTime).Milliseconds()
		return resp
	}

	// Phase 3: Normalize
	normalized := e.normalizer.Normalize(recognizeResult)
	resp.Normalized = normalized.Canonical

	// Phase 4: Resolve (variables, aliases)
	resolved, resolveErr := e.resolve(normalized, req.SessionID)
	if resolveErr != nil {
		resp.Error = resolveErr
		resp.ProcessingTime = time.Since(startTime).Milliseconds()
		return resp
	}
	resp.Resolved = resolved

	// Phase 5: Dispatch (if execution context allows)
	authority, _ := AuthorityLevels[req.Context]
	if authority.CanExecute && normalized.RequiresExecution {
		result, dispatchErr := e.dispatcher.Dispatch(&req.Context, req.SessionID, normalized)
		if dispatchErr != nil {
			resp.Error = dispatchErr
			resp.ProcessingTime = time.Since(startTime).Milliseconds()
			return resp
		}
		resp.Result = result
	}

	resp.ProcessingTime = time.Since(startTime).Milliseconds()
	return resp
}

// resolve resolves variables and aliases in a command
func (e *Engine) resolve(cmd *NormalizedCommand, sessionID string) (string, *ICMError) {
	switch cmd.Operator {
	case OperatorUserVariable:
		return e.resolveUserVariable(cmd.Command, sessionID)
	case OperatorSystemVariable:
		return e.resolveSystemVariable(cmd.Command)
	case OperatorAlias:
		return e.resolveAlias(cmd.Command)
	}
	return cmd.Canonical, nil
}

// resolveUserVariable resolves a user variable reference
func (e *Engine) resolveUserVariable(varName string, sessionID string) (string, *ICMError) {
	// Extract variable name from ${name} or $name
	name := varName
	if len(name) > 2 && name[0] == '$' && name[1] == '{' {
		close := len(name) - 1
		if name[close] == '}' {
			name = name[2:close]
		}
	} else if len(name) > 1 && name[0] == '$' {
		name = name[1:]
	}

	// Check session variables first
	e.mu.RLock()
	session := e.sessions[sessionID]
	if session != nil && session.Variables != nil {
		if val, ok := session.Variables[name]; ok {
			e.mu.RUnlock()
			return val, nil
		}
	}
	e.mu.RUnlock()

	// Check registry
	if val, ok := e.registry.GetUserVariable(name); ok {
		return val, nil
	}

	// Variable not found - pass through literally
	return "${" + name + "}", nil
}

// resolveSystemVariable resolves a system variable reference
func (e *Engine) resolveSystemVariable(varName string) (string, *ICMError) {
	// Check if numeric (session variable)
	isNumeric := true
	for _, c := range varName {
		if c < '0' || c > '9' {
			isNumeric = false
			break
		}
	}

	if isNumeric {
		// Numeric session variable - would be resolved from session context
		// For now, return the reference
		return "%" + varName, nil
	}

	// Named system variable
	if val, ok := e.registry.GetSystemVariable(varName); ok {
		return val, nil
	}

	// Unknown system variable - pass through
	return "%" + varName, nil
}

// resolveAlias resolves an alias reference
func (e *Engine) resolveAlias(aliasName string) (string, *ICMError) {
	if alias, ok := e.registry.GetAlias(aliasName); ok {
		return alias.Expansion, nil
	}

	// Alias not found - return error
	return "", NewICMError(E3001UndefinedAlias, "Alias not found: "+aliasName, map[string]interface{}{
		"alias": aliasName,
	})
}

// Validate performs validation without execution
func (e *Engine) Validate(input string) *ICMError {
	// Recognize
	recognizeResult, err := e.recognizer.Recognize(input)
	if err != nil {
		return err
	}

	// Validate
	return e.validator.Validate(recognizeResult)
}

// Normalize normalizes a command without execution
func (e *Engine) Normalize(input string) *NormalizedCommand {
	recognizeResult, err := e.recognizer.Recognize(input)
	if err != nil {
		return nil
	}

	return e.normalizer.Normalize(recognizeResult)
}

// Classify classifies a command as internal or pass-through
func (e *Engine) Classify(input string) *ClassificationResult {
	return e.passThroughClassifier.ClassifyCommand(input)
}

// RegisterCommandHandler registers a command handler
func (e *Engine) RegisterCommandHandler(family OperatorFamily, command string, handler CommandHandler) error {
	return e.dispatcher.RegisterHandler(family, command, handler)
}

// SetSessionVariable sets a session-scoped variable
func (e *Engine) SetSessionVariable(sessionID, name, value string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.sessions[sessionID] == nil {
		e.sessions[sessionID] = &ExecutionSession{
			SessionID: sessionID,
			Variables: make(map[string]string),
			IsActive:  true,
		}
	}
	e.sessions[sessionID].Variables[name] = value
}

// GetSessionVariable gets a session-scoped variable
func (e *Engine) GetSessionVariable(sessionID, name string) (string, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.sessions[sessionID] == nil || e.sessions[sessionID].Variables == nil {
		return "", false
	}
	val, ok := e.sessions[sessionID].Variables[name]
	return val, ok
}

// ClearSession clears session-specific state
func (e *Engine) ClearSession(sessionID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.sessions, sessionID)
}

// CreateSession creates a new session
func (e *Engine) CreateSession(sessionID, userID, characterName, serverName, host, port string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.sessions[sessionID] = &ExecutionSession{
		SessionID:     sessionID,
		UserID:        userID,
		CharacterName: characterName,
		ServerName:    serverName,
		Host:          host,
		Port:          port,
		ConnectedAt:   time.Now(),
		Variables:     make(map[string]string),
		IsActive:      true,
	}
}

// GetSession returns a session by ID
func (e *Engine) GetSession(sessionID string) (*ExecutionSession, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	session, ok := e.sessions[sessionID]
	return session, ok
}

// UpdateSession updates session information
func (e *Engine) UpdateSession(sessionID string, updateFn func(*ExecutionSession)) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if session, ok := e.sessions[sessionID]; ok {
		updateFn(session)
	}
}

// RegisterEffectHandler registers a handler for state effects
func (e *Engine) RegisterEffectHandler(effectType string, handler StateEffectHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.effectHandlers[effectType] = handler
}

// ProcessEffects processes state effects from command execution
func (e *Engine) ProcessEffects(effects []StateEffect, sessionID string) {
	e.mu.RLock()
	handlers := make(map[string]StateEffectHandler)
	for k, v := range e.effectHandlers {
		handlers[k] = v
	}
	e.mu.RUnlock()

	for _, effect := range effects {
		if handler, ok := handlers[effect.Type]; ok {
			_ = handler.HandleEffect(effect, sessionID)
		}
	}
}

// AddToHistory adds a command to the session history
func (e *Engine) AddToHistory(sessionID, command string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.commandHistory[sessionID] == nil {
		e.commandHistory[sessionID] = []string{}
	}
	// Keep last 100 commands
	history := e.commandHistory[sessionID]
	history = append(history, command)
	if len(history) > 100 {
		history = history[1:]
	}
	e.commandHistory[sessionID] = history
}

// GetHistory returns command history for a session
func (e *Engine) GetHistory(sessionID string) []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.commandHistory[sessionID]
}

// GetRegistry returns the command registry
func (e *Engine) GetRegistry() *Registry {
	return e.registry
}

// GetRecognizer returns the recognizer
func (e *Engine) GetRecognizer() *Recognizer {
	return e.recognizer
}

// GetValidator returns the validator
func (e *Engine) GetValidator() *Validator {
	return e.validator
}

// GetNormalizer returns the normalizer
func (e *Engine) GetNormalizer() *Normalizer {
	return e.normalizer
}

// GetDispatcher returns the dispatcher
func (e *Engine) GetDispatcher() *Dispatcher {
	return e.dispatcher
}

// Global engine instance (for use in handlers)
var (
	defaultEngine     *Engine
	defaultEngineInit sync.Once
)

// GetDefaultEngine returns the default ICM engine
func GetDefaultEngine() *Engine {
	defaultEngineInit.Do(func() {
		defaultEngine = NewEngine()
	})
	return defaultEngine
}
