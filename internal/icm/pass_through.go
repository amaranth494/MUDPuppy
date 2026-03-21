package icm

import (
	"strings"
)

// Classifier handles classification of commands as internal vs MUD commands
type Classifier struct {
	reservedOperators map[rune]OperatorFamily
}

// NewClassifier creates a new command classifier
func NewClassifier() *Classifier {
	return &Classifier{
		reservedOperators: map[rune]OperatorFamily{
			'#': OperatorStructured,
			'@': OperatorAlias,
			'$': OperatorUserVariable,
			'%': OperatorSystemVariable,
		},
	}
}

// ClassificationResult contains the result of command classification
type ClassificationResult struct {
	// Whether it's an internal command
	IsInternal bool

	// The operator family (if internal)
	Operator OperatorFamily

	// The command (operator + rest)
	Command string

	// Whether it should pass through to MUD server
	ShouldPassThrough bool

	// The pass-through value
	PassThroughValue string
}

// Classify determines if a command is internal or should pass through
func (c *Classifier) Classify(input string) *ClassificationResult {
	input = strings.TrimSpace(input)
	if input == "" {
		return &ClassificationResult{
			IsInternal:        false,
			ShouldPassThrough: true,
			PassThroughValue:  "",
		}
	}

	// Get first character
	firstChar := rune(input[0])

	// Check if it's a reserved operator
	if op, isReserved := c.reservedOperators[firstChar]; isReserved {
		// It's a reserved operator - check if it's a known command
		return c.classifyReservedOperator(input, op)
	}

	// Not a reserved operator - pass through to MUD
	return &ClassificationResult{
		IsInternal:        false,
		ShouldPassThrough: true,
		PassThroughValue:  input,
	}
}

// classifyReservedOperator classifies input starting with a reserved operator
func (c *Classifier) classifyReservedOperator(input string, op OperatorFamily) *ClassificationResult {
	// It's a reserved operator - it's an internal command
	// Unknown commands within reserved families should return error, not pass through
	return &ClassificationResult{
		IsInternal:        true,
		Operator:          op,
		Command:           input,
		ShouldPassThrough: false,
		PassThroughValue:  "",
	}
}

// UnknownOperatorHandler handles unknown operators
type UnknownOperatorHandler struct {
	classifier    *Classifier
	escapeHandler *EscapeHandler
}

// NewUnknownOperatorHandler creates a new unknown operator handler
func NewUnknownOperatorHandler() *UnknownOperatorHandler {
	return &UnknownOperatorHandler{
		classifier:    NewClassifier(),
		escapeHandler: NewEscapeHandler(),
	}
}

// HandleUnknownOperator handles input with an unknown operator
func (h *UnknownOperatorHandler) HandleUnknownOperator(input string) *ClassificationResult {
	input = strings.TrimSpace(input)
	if input == "" {
		return &ClassificationResult{
			IsInternal:        false,
			ShouldPassThrough: true,
			PassThroughValue:  "",
		}
	}

	// Check if it's a reserved operator with unknown command
	firstChar := rune(input[0])
	if _, isReserved := h.classifier.reservedOperators[firstChar]; isReserved {
		// Reserved operator but unknown command - return error
		// This should be caught by validation, but handle here too
		return &ClassificationResult{
			IsInternal:        true,
			ShouldPassThrough: false,
			PassThroughValue:  "",
		}
	}

	// Unknown operator (not reserved) - pass through to MUD
	return &ClassificationResult{
		IsInternal:        false,
		ShouldPassThrough: true,
		PassThroughValue:  input,
	}
}

// HandleUnknownCommand handles unknown commands within a reserved family
func (h *UnknownOperatorHandler) HandleUnknownCommand(family OperatorFamily, command string) *ICMError {
	// Per spec: Unknown commands in reserved families should return validation error
	// Do NOT pass through to MUD server

	switch family {
	case OperatorStructured:
		return NewICMError(E2004UnknownCommand, "Unknown directive: "+command, map[string]interface{}{
			"command": command,
			"family":  string(family),
		})
	case OperatorAlias:
		return NewICMError(E3001UndefinedAlias, "Unknown alias: "+command, map[string]interface{}{
			"alias": command,
		})
	case OperatorUserVariable:
		// User variables pass through if undefined
		return nil
	case OperatorSystemVariable:
		// System variables - check if known
		return nil
	}

	return nil
}

// ShouldPassToMUD determines if a command should pass through to the MUD server
func (h *UnknownOperatorHandler) ShouldPassToMUD(input string) bool {
	result := h.classifier.Classify(input)
	return result.ShouldPassThrough
}

// GetPassThroughValue returns the value to pass through to MUD
func (h *UnknownOperatorHandler) GetPassThroughValue(input string) string {
	result := h.classifier.Classify(input)
	if result.ShouldPassThrough {
		return result.PassThroughValue
	}
	return ""
}

// PassThroughClassifier implements the pass-through classification algorithm
type PassThroughClassifier struct {
	reservedChars map[rune]bool
	escapeHandler *EscapeHandler
	registry      *Registry
}

// NewPassThroughClassifier creates a new pass-through classifier
func NewPassThroughClassifier(registry *Registry) *PassThroughClassifier {
	return &PassThroughClassifier{
		reservedChars: map[rune]bool{
			'#': true,
			'@': true,
			'$': true,
			'%': true,
		},
		escapeHandler: NewEscapeHandler(),
		registry:      registry,
	}
}

// ClassifyCommand implements the full classification algorithm
// Per PR02PH01-design.md Section 10
func (c *PassThroughClassifier) ClassifyCommand(input string) *ClassificationResult {
	input = strings.TrimSpace(input)

	// Step 1: Check for empty input
	if input == "" {
		return &ClassificationResult{
			IsInternal:        false,
			ShouldPassThrough: false,
			PassThroughValue:  "",
		}
	}

	// Step 2: Check if first char is a reserved operator
	firstChar := rune(input[0])
	if c.reservedChars[firstChar] {
		return c.classifyReserved(input, firstChar)
	}

	// Step 3: Not a reserved operator - pass through to MUD
	return &ClassificationResult{
		IsInternal:        false,
		ShouldPassThrough: true,
		PassThroughValue:  input,
	}
}

// classifyReserved classifies input starting with a reserved operator
func (c *PassThroughClassifier) classifyReserved(input string, firstChar rune) *ClassificationResult {
	op := c.reservedChars[firstChar]
	if !op {
		return &ClassificationResult{
			IsInternal:        false,
			ShouldPassThrough: true,
			PassThroughValue:  input,
		}
	}

	// Extract the command part
	var command string
	switch firstChar {
	case '#':
		rest := strings.TrimSpace(input[1:])
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			command = strings.ToUpper(parts[0])
		}
	case '@':
		rest := strings.TrimSpace(input[1:])
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			command = parts[0]
		}
	case '$':
		rest := strings.TrimSpace(input[1:])
		if strings.HasPrefix(rest, "{") {
			close := strings.Index(rest, "}")
			if close > 0 {
				command = rest[1:close]
			}
		} else {
			parts := strings.Fields(rest)
			if len(parts) > 0 {
				command = parts[0]
			}
		}
	case '%':
		rest := strings.TrimSpace(input[1:])
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			command = parts[0]
		}
	}

	// Check if the command is known
	isKnown := c.isKnownCommand(firstChar, command)

	if !isKnown {
		// Unknown command in reserved family - error, not pass through
		return &ClassificationResult{
			IsInternal:        true,
			Operator:          OperatorFamily(string(firstChar)),
			ShouldPassThrough: false,
			PassThroughValue:  "",
		}
	}

	// Known command - internal
	return &ClassificationResult{
		IsInternal:        true,
		Operator:          OperatorFamily(string(firstChar)),
		Command:           input,
		ShouldPassThrough: false,
		PassThroughValue:  "",
	}
}

// isKnownCommand checks if a command is known within its family
func (c *PassThroughClassifier) isKnownCommand(family rune, command string) bool {
	switch family {
	case '#':
		// Check structured commands
		upperCmd := strings.ToUpper(command)
		knownCommands := map[string]bool{
			"IF": true, "ELSE": true, "ENDIF": true,
			"SET": true, "ADD": true, "SUB": true,
			"TIMER": true, "START": true, "STOP": true,
			"CHECK": true, "CANCEL": true,
			"LOG": true, "ECHO": true, "HELP": true,
			"GAG": true, "DELAY": true, "EXPAND": true,
		}
		return knownCommands[upperCmd]

	case '@':
		// Check aliases in registry
		if c.registry != nil {
			return c.registry.HasAlias(command)
		}
		return false

	case '$':
		// User variables don't need to be known - undefined = pass through
		return true

	case '%':
		// Check system variables
		upperCmd := strings.ToUpper(command)
		if c.registry != nil {
			return c.registry.HasSystemVariable(upperCmd)
		}
		// Default system variables
		knownSysVars := map[string]bool{
			"TIME": true, "DATE": true, "CHARACTER": true,
			"SERVER": true, "SESSIONID": true, "HOST": true, "PORT": true,
		}
		// Numeric variables are always valid session variables
		if isNumeric(upperCmd) {
			return true
		}
		return knownSysVars[upperCmd]
	}

	return false
}

// isNumeric checks if a string contains only digits
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
