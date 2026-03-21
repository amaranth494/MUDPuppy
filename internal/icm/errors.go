package icm

// ErrorCode represents an ICM error code
type ErrorCode string

const (
	// Recognition errors (1xxx)
	E1000UnrecognizedOperator ErrorCode = "E1000"
	E1001AmbiguousInput       ErrorCode = "E1001"

	// Validation errors (2xxx)
	E2000SyntaxError        ErrorCode = "E2000"
	E2001MissingArgument    ErrorCode = "E2001"
	E2002InvalidArgument    ErrorCode = "E2002"
	E2003ArgumentOutOfRange ErrorCode = "E2003"
	E2004UnknownCommand     ErrorCode = "E2004"

	// Resolution errors (3xxx)
	E3000UndefinedVariable ErrorCode = "E3000"
	E3001UndefinedAlias    ErrorCode = "E3001"
	E3002CircularReference ErrorCode = "E3002"
	E3003ResolutionTimeout ErrorCode = "E3003"

	// Execution errors (4xxx)
	E4000ExecutionFailed  ErrorCode = "E4000"
	E4001PermissionDenied ErrorCode = "E4001"
	E4002RateLimited      ErrorCode = "E4002"
	E4003CircuitBreaker   ErrorCode = "E4003"

	// System errors (5xxx)
	E5000InternalError      ErrorCode = "E5000"
	E5001ServiceUnavailable ErrorCode = "E5001"
)

// ICMError represents an error in ICM processing
type ICMError struct {
	Code        ErrorCode              `json:"code"`
	Message     string                 `json:"message"`
	UserMessage string                 `json:"userMessage"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Context     *ErrorContext          `json:"context,omitempty"`
}

// ErrorContext provides context about where an error occurred
type ErrorContext struct {
	Phase    string `json:"phase"`
	Position int    `json:"position,omitempty"`
	Input    string `json:"input,omitempty"`
}

// UserErrorMessages maps error codes to user-facing messages
var UserErrorMessages = map[ErrorCode]string{
	E1000UnrecognizedOperator: "Unknown operator in command",
	E1001AmbiguousInput:       "Ambiguous input - please clarify",
	E2000SyntaxError:          "Syntax error in command",
	E2001MissingArgument:      "Missing required argument",
	E2002InvalidArgument:      "Invalid argument provided",
	E2003ArgumentOutOfRange:   "Argument out of valid range",
	E2004UnknownCommand:       "Unknown command: %s",
	E3000UndefinedVariable:    "Variable not found: %s",
	E3001UndefinedAlias:       "Alias not found: %s",
	E3002CircularReference:    "Circular reference detected",
	E3003ResolutionTimeout:    "Resolution timed out",
	E4000ExecutionFailed:      "Command execution failed",
	E4001PermissionDenied:     "You don't have permission to use this command",
	E4002RateLimited:          "Too many commands. Please wait before trying again.",
	E4003CircuitBreaker:       "Command execution paused due to repeated patterns",
	E5000InternalError:        "An internal error occurred",
	E5001ServiceUnavailable:   "Service temporarily unavailable",
}

// NewICMError creates a new ICM error
func NewICMError(code ErrorCode, message string, details map[string]interface{}) *ICMError {
	userMsg, ok := UserErrorMessages[code]
	if !ok {
		userMsg = "An error occurred"
	}

	return &ICMError{
		Code:        code,
		Message:     message,
		UserMessage: userMsg,
		Details:     details,
	}
}

// NewValidationError creates a new validation error
func NewValidationError(code ErrorCode, phase string, input string, position int) *ICMError {
	return &ICMError{
		Code:        code,
		Message:     string(code),
		UserMessage: UserErrorMessages[code],
		Context: &ErrorContext{
			Phase:    phase,
			Position: position,
			Input:    input,
		},
	}
}

// Error implements the error interface
func (e *ICMError) Error() string {
	return string(e.Code) + ": " + e.Message
}
