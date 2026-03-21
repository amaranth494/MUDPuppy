// Package icm provides the Internal Command Module (ICM) for MUDPuppy.
// The ICM handles all internal commands including directives (#), aliases (@),
// user variables ($), and system variables (%).
package icm

import (
	"time"
)

// OperatorFamily represents the type of internal command operator
type OperatorFamily string

const (
	// OperatorStructured represents # (directives like #IF, #SET, #TIMER)
	OperatorStructured OperatorFamily = "#"
	// OperatorAlias represents @ (alias references)
	OperatorAlias OperatorFamily = "@"
	// OperatorUserVariable represents $ (user variables)
	OperatorUserVariable OperatorFamily = "$"
	// OperatorSystemVariable represents % (system/session variables)
	OperatorSystemVariable OperatorFamily = "%"
)

// ExecutionContext defines where a command is being executed from
type ExecutionContext string

const (
	// ContextPreview - Local validation, help display, syntax check (frontend-only)
	ContextPreview ExecutionContext = "preview"
	// ContextSubmission - User command entry, keybinding, history (Frontend → Backend)
	ContextSubmission ExecutionContext = "submission"
	// ContextAutomation - Alias expansion, trigger action, timer fire (Backend-authoritative)
	ContextAutomation ExecutionContext = "automation"
	// ContextOperational - Service tooling, admin commands (Backend-only)
	ContextOperational ExecutionContext = "operational"
)

// Token represents a parsed command token
type Token struct {
	Type     TokenType
	Value    string
	Position int
}

// TokenType represents the type of token
type TokenType int

const (
	TokenOperator TokenType = iota
	TokenCommand
	TokenArgument
	TokenEquals
	TokenWhitespace
	TokenEOF
)

// ParsedCommand represents a command that has been tokenized and recognized
type ParsedCommand struct {
	// Input
	RawInput string

	// Recognized
	Operator   OperatorFamily
	Command    string
	Args       []string
	IsInternal bool

	// Normalized
	Normalized string

	// Metadata
	Tokens          []Token
	Transformations []string

	// Position tracking
	Pos int
}

// NormalizedCommand represents the canonical form of a command
type NormalizedCommand struct {
	// Canonical form
	Canonical string

	// Decomposed parts
	Operator OperatorFamily
	Command  string
	Args     []string

	// Metadata
	OriginalInput   string
	Transformations []string

	// Classification
	IsInternal        bool
	RequiresExecution bool
}

// ICMRequest represents a request to the ICM
type ICMRequest struct {
	// Input
	Raw string `json:"raw"`

	// Execution context
	Context ExecutionContext `json:"context"`

	// Session context
	SessionID string `json:"sessionId"`
	UserID    string `json:"userId,omitempty"`

	// Optional: pre-resolved variables
	Variables map[string]string `json:"variables,omitempty"`

	// Options
	Options *ICMOptions `json:"options,omitempty"`
}

// ICMOptions provides optional parameters for ICM processing
type ICMOptions struct {
	SkipSafety bool `json:"skipSafety,omitempty"`
	DryRun     bool `json:"dryRun,omitempty"`
	Trace      bool `json:"trace,omitempty"`
}

// ICMResponse represents the response from the ICM
type ICMResponse struct {
	// Command classification
	IsInternal bool           `json:"isInternal"`
	Operator   OperatorFamily `json:"operator,omitempty"`
	Command    string         `json:"command,omitempty"`

	// Processed form
	Normalized string `json:"normalized,omitempty"`
	Resolved   string `json:"resolved,omitempty"`

	// Result
	Result *CommandResult `json:"result,omitempty"`
	Error  *ICMError      `json:"error,omitempty"`

	// Metadata
	ProcessingTime int64            `json:"processingTime"`
	Trace          *ProcessingTrace `json:"trace,omitempty"`
}

// CommandResult represents the result of command execution
type CommandResult struct {
	Output   string         `json:"output,omitempty"`
	Effects  []StateEffect  `json:"effects,omitempty"`
	Metadata ResultMetadata `json:"metadata"`
}

// ResultMetadata contains metadata about command execution
type ResultMetadata struct {
	ExecutionTime time.Duration `json:"executionTime"`
	Handler       string        `json:"handler"`
}

// StateEffect represents a state change resulting from command execution
type StateEffect struct {
	Type  string      `json:"type"`
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// ProcessingTrace contains trace information for debugging
type ProcessingTrace struct {
	Phases   []PhaseTrace `json:"phases"`
	Warnings []string     `json:"warnings,omitempty"`
}

// PhaseTrace represents the execution of a single phase
type PhaseTrace struct {
	Phase    string        `json:"phase"`
	Duration time.Duration `json:"duration"`
	Input    string        `json:"input"`
	Output   string        `json:"output"`
	Error    string        `json:"error,omitempty"`
}

// CommandHandler defines the interface for command handlers
type CommandHandler interface {
	// Handle executes the command and returns the result
	Handle(ctx *ExecutionContext, cmd *NormalizedCommand) (*CommandResult, *ICMError)

	// Validate validates the command arguments
	Validate(cmd *NormalizedCommand) *ICMError

	// Name returns the command name
	Name() string

	// Description returns the command description
	Description() string
}

// AuthorityLevel defines what operations are allowed in a context
type AuthorityLevel struct {
	CanExecute      bool
	CanModifyState  bool
	RequiresAuth    bool
	IsAutomation    bool
	RequiresRole    string
	AllowedContexts []ExecutionContext
}

// ExecutionSession holds session-specific execution context
type ExecutionSession struct {
	SessionID     string
	UserID        string
	CharacterID   string
	CharacterName string
	ServerName    string
	Host          string
	Port          string
	ConnectedAt   time.Time
	Variables     map[string]string
	IsActive      bool
}

// LogEntry represents a #LOG command entry
type LogEntry struct {
	ID            string                 `json:"id"`
	Timestamp     time.Time              `json:"timestamp"`
	Level         string                 `json:"level"`
	Message       string                 `json:"message"`
	SessionID     string                 `json:"sessionId"`
	UserID        string                 `json:"userId"`
	CorrelationID string                 `json:"correlationId"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// SafetyLimits defines the default safety limits for ICM
var SafetyLimits = struct {
	MaxAliasDepth           int
	MaxCommandsPerDispatch  int
	MinTriggerInterval      time.Duration
	MaxLoopCount            int
	MaxExecutionTime        time.Duration
	MaxQueueDepth           int
	MaxVariableDepth        int
	MaxExpressionComplexity int
}{
	MaxAliasDepth:           3,
	MaxCommandsPerDispatch:  100,
	MinTriggerInterval:      100 * time.Millisecond,
	MaxLoopCount:            5,
	MaxExecutionTime:        5 * time.Second,
	MaxQueueDepth:           100,
	MaxVariableDepth:        10,
	MaxExpressionComplexity: 50,
}

// AuthorityLevels maps execution contexts to their authority levels
var AuthorityLevels = map[ExecutionContext]AuthorityLevel{
	ContextPreview: {
		CanExecute:      false,
		CanModifyState:  false,
		RequiresAuth:    false,
		AllowedContexts: []ExecutionContext{ContextPreview},
	},
	ContextSubmission: {
		CanExecute:      true,
		CanModifyState:  true,
		RequiresAuth:    true,
		AllowedContexts: []ExecutionContext{ContextSubmission, ContextPreview},
	},
	ContextAutomation: {
		CanExecute:      true,
		CanModifyState:  true,
		RequiresAuth:    true,
		IsAutomation:    true,
		AllowedContexts: []ExecutionContext{ContextAutomation, ContextPreview},
	},
	ContextOperational: {
		CanExecute:      true,
		CanModifyState:  true,
		RequiresAuth:    true,
		RequiresRole:    "admin",
		AllowedContexts: []ExecutionContext{ContextOperational, ContextPreview},
	},
}

// GetAuthorityForContext returns the authority level for a given context
func GetAuthorityForContext(ctx ExecutionContext) AuthorityLevel {
	if auth, ok := AuthorityLevels[ctx]; ok {
		return auth
	}
	return AuthorityLevels[ContextPreview]
}

// CanExecuteInContext checks if a command can execute in the given context
func CanExecuteInContext(ctx ExecutionContext, cmd *NormalizedCommand) bool {
	auth := GetAuthorityForContext(ctx)
	if !auth.CanExecute {
		return false
	}

	// Check if command requires specific context
	return true
}
