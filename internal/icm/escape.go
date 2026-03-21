package icm

import (
	"strings"
)

// EscapeHandler manages escape sequences and literal pass-through
type EscapeHandler struct {
	// Escape sequences map escape char to literal
	escapeSequences map[string]string

	// Reserved operator characters
	reservedOperators map[rune]bool
}

// NewEscapeHandler creates a new escape handler
func NewEscapeHandler() *EscapeHandler {
	return &EscapeHandler{
		escapeSequences: map[string]string{
			"\\#":  "#",
			"\\@":  "@",
			"\\$":  "$",
			"\\%":  "%",
			"\\\\": "\\",
			"\\n":  "\n",
			"\\t":  "\t",
			"\\r":  "\r",
		},
		reservedOperators: map[rune]bool{
			'#': true,
			'@': true,
			'$': true,
			'%': true,
		},
	}
}

// ResolveEscapes converts escape sequences to their literal values
func (e *EscapeHandler) ResolveEscapes(input string) string {
	result := input

	// Process escape sequences (longest first to avoid partial matches)
	escapes := []string{"\\n", "\\t", "\\r", "\\#", "\\@", "\\$", "\\%", "\\\\"}
	for _, escape := range escapes {
		literal := e.escapeSequences[escape]
		result = strings.ReplaceAll(result, escape, literal)
	}

	return result
}

// PreserveEscapes preserves escape sequences in normalized form
func (e *EscapeHandler) PreserveEscapes(input string) string {
	// For storage, we want to keep escape sequences as-is
	// Just do basic whitespace normalization
	return strings.TrimSpace(input)
}

// IsEscapeSequence checks if a string is an escape sequence
func (e *EscapeHandler) IsEscapeSequence(s string) bool {
	_, ok := e.escapeSequences[s]
	return ok
}

// GetEscapeSequence returns the literal for an escape sequence
func (e *EscapeHandler) GetEscapeSequence(escape string) (string, bool) {
	literal, ok := e.escapeSequences[escape]
	return literal, ok
}

// IsReservedOperator checks if a character is a reserved operator
func (e *EscapeHandler) IsReservedOperator(ch rune) bool {
	return e.reservedOperators[ch]
}

// ProcessEscapeAtPosition processes an escape sequence at a specific position
func (e *EscapeHandler) ProcessEscapeAtPosition(input string, pos int) (string, int) {
	if pos >= len(input)-1 {
		return string(input[pos]), pos + 1
	}

	// Check for two-character escape
	twoChar := input[pos : pos+2]
	if literal, ok := e.escapeSequences[twoChar]; ok {
		return literal, pos + 2
	}

	// Not a recognized escape, return as-is
	return string(input[pos]), pos + 1
}

// FindEscapeSequences finds all escape sequences in a string
func (e *EscapeHandler) FindEscapeSequences(input string) []string {
	var found []string

	for escape := range e.escapeSequences {
		if strings.Contains(input, escape) {
			found = append(found, escape)
		}
	}

	return found
}

// CountEscapeSequences counts escape sequences in a string
func (e *EscapeHandler) CountEscapeSequences(input string) int {
	count := 0

	for escape := range e.escapeSequences {
		count += strings.Count(input, escape)
	}

	return count
}

// HasEscapeSequences checks if input contains any escape sequences
func (e *EscapeHandler) HasEscapeSequences(input string) bool {
	for escape := range e.escapeSequences {
		if strings.Contains(input, escape) {
			return true
		}
	}
	return false
}

// EscapeLiteral escapes a literal string to include escape sequences
// This is the inverse of ResolveEscapes
func (e *EscapeHandler) EscapeLiteral(input string) string {
	result := input

	// Escape in reverse order to avoid double-escaping
	// First escape backslash
	result = strings.ReplaceAll(result, "\\", "\\\\")
	// Then escape operators
	result = strings.ReplaceAll(result, "#", "\\#")
	result = strings.ReplaceAll(result, "@", "\\@")
	result = strings.ReplaceAll(result, "$", "\\$")
	result = strings.ReplaceAll(result, "%", "\\%")
	// Then escape special chars
	result = strings.ReplaceAll(result, "\n", "\\n")
	result = strings.ReplaceAll(result, "\t", "\\t")
	result = strings.ReplaceAll(result, "\r", "\\r")

	return result
}

// LiteralPassThrough determines if input should pass through literally
type LiteralPassThrough struct {
	EscapeHandler *EscapeHandler
}

// NewLiteralPassThrough creates a new literal pass-through handler
func NewLiteralPassThrough() *LiteralPassThrough {
	return &LiteralPassThrough{
		EscapeHandler: NewEscapeHandler(),
	}
}

// ShouldPassThrough checks if unrecognized input should pass through
func (l *LiteralPassThrough) ShouldPassThrough(input string) bool {
	input = strings.TrimSpace(input)

	// Empty input doesn't pass through
	if input == "" {
		return false
	}

	// Get first character
	firstChar := rune(input[0])

	// If first char is a reserved operator, we should NOT pass through
	// (it would be an unknown command in a known family)
	if l.EscapeHandler.IsReservedOperator(firstChar) {
		return false
	}

	// Unknown operator (like &) should pass through
	return true
}

// HandleUnknownVariable handles undefined variable references
func (l *LiteralPassThrough) HandleUnknownVariable(input string) string {
	// Per spec: undefined variables pass through literally
	// e.g., ${undefined} -> ${undefined}
	return input
}

// HandleUnknownSystemVariable handles undefined system variable references
func (l *LiteralPassThrough) HandleUnknownSystemVariable(input string) string {
	// Per spec: unknown system variables pass through literally
	// e.g., %UNKNOWN -> %UNKNOWN
	return input
}

// HandleMalformedReference handles malformed variable references
func (l *LiteralPassThrough) HandleMalformedReference(input string) string {
	// Per spec: malformed references (like % alone) pass through literally
	return input
}

// ValidateLiteral checks if a literal value is valid
func (l *LiteralPassThrough) ValidateLiteral(input string) bool {
	// Empty is valid
	if input == "" {
		return true
	}

	// Check for balanced braces
	openBraces := 0
	closeBraces := 0

	for _, ch := range input {
		switch ch {
		case '{':
			openBraces++
		case '}':
			closeBraces++
		}
	}

	// If braces are unbalanced, it might be malformed
	// But per spec, we still pass through
	return true
}
