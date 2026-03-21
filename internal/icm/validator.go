package icm

import (
	"regexp"
	"strings"
)

// Validator handles syntax validation and argument checking
type Validator struct {
	// Structured command syntax rules
	structuredRules map[string]*StructuredRule

	// Variable name pattern
	variablePattern *regexp.Regexp

	// Alias name pattern
	aliasPattern *regexp.Regexp
}

// StructuredRule defines syntax rules for a structured command
type StructuredRule struct {
	Command       string
	MinArgs       int
	MaxArgs       int
	RequiredFlags []string
	OptionalFlags []string
	ArgPatterns   []string // Regex patterns for each argument
}

// NewValidator creates a new Validator
func NewValidator() *Validator {
	v := &Validator{
		variablePattern: regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`),
		aliasPattern:    regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`),
	}

	// Initialize structured command rules
	v.structuredRules = map[string]*StructuredRule{
		// Conditional commands
		"IF": {
			Command:     "IF",
			MinArgs:     1,
			MaxArgs:     3,
			ArgPatterns: []string{`^.+$`, `^.+$`, `^.+$`}, // condition, true-val, false-val
		},
		"ELSE": {
			Command: "ELSE",
			MinArgs: 0,
			MaxArgs: 0,
		},
		"ENDIF": {
			Command: "ENDIF",
			MinArgs: 0,
			MaxArgs: 0,
		},

		// Variable commands
		"SET": {
			Command:     "SET",
			MinArgs:     2,
			MaxArgs:     2,
			ArgPatterns: []string{`^[a-zA-Z_][a-zA-Z0-9_]*$`, `.*`}, // name, value
		},
		"ADD": {
			Command:     "ADD",
			MinArgs:     2,
			MaxArgs:     2,
			ArgPatterns: []string{`^[a-zA-Z_][a-zA-Z0-9_]*$`, `^-?[\d.]+$`}, // name, value
		},
		"SUB": {
			Command:     "SUB",
			MinArgs:     2,
			MaxArgs:     2,
			ArgPatterns: []string{`^[a-zA-Z_][a-zA-Z0-9_]*$`, `^-?[\d.]+$`}, // name, value
		},

		// Timer commands
		"TIMER": {
			Command:     "TIMER",
			MinArgs:     3,
			MaxArgs:     4,
			ArgPatterns: []string{`^[a-zA-Z_][a-zA-Z0-9_]*$`, `^\d+$`, `^.+$`, `.*`}, // name, interval, command
		},
		"START": {
			Command:     "START",
			MinArgs:     1,
			MaxArgs:     1,
			ArgPatterns: []string{`^[a-zA-Z_][a-zA-Z0-9_]*$`}, // timer name
		},
		"STOP": {
			Command:     "STOP",
			MinArgs:     1,
			MaxArgs:     1,
			ArgPatterns: []string{`^[a-zA-Z_][a-zA-Z0-9_]*$`}, // timer name
		},
		"CHECK": {
			Command:     "CHECK",
			MinArgs:     1,
			MaxArgs:     1,
			ArgPatterns: []string{`^.+$`}, // condition
		},
		"CANCEL": {
			Command:     "CANCEL",
			MinArgs:     1,
			MaxArgs:     1,
			ArgPatterns: []string{`^[a-zA-Z_][a-zA-Z0-9_]*$`}, // timer name
		},

		// Logging command
		"LOG": {
			Command:     "LOG",
			MinArgs:     1,
			MaxArgs:     10,
			ArgPatterns: []string{`.*`}, // message (can be any string)
		},

		// Echo command
		"ECHO": {
			Command:     "ECHO",
			MinArgs:     0,
			MaxArgs:     10,
			ArgPatterns: []string{`.*`},
		},

		// Help command
		"HELP": {
			Command:     "HELP",
			MinArgs:     0,
			MaxArgs:     1,
			ArgPatterns: []string{`^[a-zA-Z_][a-zA-Z0-9_]*$`},
		},

		// Gag command (suppress output)
		"GAG": {
			Command:     "GAG",
			MinArgs:     0,
			MaxArgs:     1,
			ArgPatterns: []string{`^\d+$`}, // line count
		},

		// Delay command
		"DELAY": {
			Command:     "DELAY",
			MinArgs:     1,
			MaxArgs:     1,
			ArgPatterns: []string{`^\d+$`}, // milliseconds
		},

		// Variable expansion (internal)
		"EXPAND": {
			Command:     "EXPAND",
			MinArgs:     1,
			MaxArgs:     1,
			ArgPatterns: []string{`.*`},
		},
	}

	return v
}

// Validate validates a parsed command
func (v *Validator) Validate(parsed *RecognizeResult) *ICMError {
	if parsed == nil {
		return NewValidationError(E2000SyntaxError, "validate", "", 0)
	}

	// If not an internal command, no validation needed
	if !parsed.IsInternal || parsed.Operator == "" {
		return nil
	}

	switch parsed.Operator {
	case OperatorStructured:
		return v.validateStructured(parsed.Command, parsed.Args)
	case OperatorAlias:
		return v.validateAlias(parsed.Command)
	case OperatorUserVariable:
		return v.validateUserVariable(parsed.Command)
	case OperatorSystemVariable:
		return v.validateSystemVariable(parsed.Command)
	}

	return nil
}

// validateStructured validates a structured directive
func (v *Validator) validateStructured(command string, args []string) *ICMError {
	rule, exists := v.structuredRules[strings.ToUpper(command)]
	if !exists {
		return NewICMError(E2004UnknownCommand, "Unknown structured directive: "+command, map[string]interface{}{
			"command": command,
		})
	}

	// Check minimum arguments
	if len(args) < rule.MinArgs {
		return NewICMError(E2001MissingArgument, "Missing required argument for "+command, map[string]interface{}{
			"command":    command,
			"minArgs":    rule.MinArgs,
			"actualArgs": len(args),
		})
	}

	// Check maximum arguments
	if len(args) > rule.MaxArgs {
		return NewICMError(E2002InvalidArgument, "Too many arguments for "+command, map[string]interface{}{
			"command":    command,
			"maxArgs":    rule.MaxArgs,
			"actualArgs": len(args),
		})
	}

	// Validate argument patterns
	for i, arg := range args {
		if i < len(rule.ArgPatterns) {
			pattern := rule.ArgPatterns[i]
			if pattern != "" && pattern != ".*" {
				matched, err := regexp.MatchString(pattern, arg)
				if err != nil || !matched {
					return NewICMError(E2002InvalidArgument, "Invalid argument at position "+string(rune('1'+i)), map[string]interface{}{
						"command":  command,
						"position": i,
						"value":    arg,
						"expected": pattern,
					})
				}
			}
		}
	}

	return nil
}

// validateAlias validates an alias name
func (v *Validator) validateAlias(aliasName string) *ICMError {
	if aliasName == "" {
		return NewICMError(E2001MissingArgument, "Alias name is required", nil)
	}

	if !v.aliasPattern.MatchString(aliasName) {
		return NewICMError(E2002InvalidArgument, "Invalid alias name: "+aliasName, map[string]interface{}{
			"alias": aliasName,
		})
	}

	return nil
}

// validateUserVariable validates a user variable name
func (v *Validator) validateUserVariable(varName string) *ICMError {
	if varName == "" {
		return NewICMError(E2001MissingArgument, "Variable name is required", nil)
	}

	if !v.variablePattern.MatchString(varName) {
		return NewICMError(E2002InvalidArgument, "Invalid variable name: "+varName, map[string]interface{}{
			"variable": varName,
		})
	}

	return nil
}

// validateSystemVariable validates a system variable reference
func (v *Validator) validateSystemVariable(varName string) *ICMError {
	if varName == "" {
		// Malformed reference like % - pass through literally
		return nil
	}

	// Check if it's a numeric variable (all digits)
	allDigits := true
	for _, c := range varName {
		if c < '0' || c > '9' {
			allDigits = false
			break
		}
	}

	if allDigits {
		// Numeric session variable - validate range
		return nil
	}

	// Named system variable - will be resolved against registry
	// For now, just validate it's a valid identifier
	if !v.variablePattern.MatchString(varName) {
		// Invalid format - pass through literally per spec
		return nil
	}

	return nil
}

// GetStructuredCommands returns all registered structured commands
func (v *Validator) GetStructuredCommands() []string {
	commands := make([]string, 0, len(v.structuredRules))
	for cmd := range v.structuredRules {
		commands = append(commands, cmd)
	}
	return commands
}

// GetStructuredRule returns the syntax rule for a command
func (v *Validator) GetStructuredRule(command string) (*StructuredRule, bool) {
	rule, ok := v.structuredRules[strings.ToUpper(command)]
	return rule, ok
}

// ValidateArgumentCount validates argument count against a rule
func (v *Validator) ValidateArgumentCount(args []string, min int, max int) *ICMError {
	if len(args) < min {
		return NewICMError(E2001MissingArgument, "Missing required arguments", map[string]interface{}{
			"min":    min,
			"actual": len(args),
		})
	}

	if len(args) > max {
		return NewICMError(E2002InvalidArgument, "Too many arguments", map[string]interface{}{
			"max":    max,
			"actual": len(args),
		})
	}

	return nil
}
