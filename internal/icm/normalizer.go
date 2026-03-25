package icm

import (
	"regexp"
	"strings"
	"unicode"
)

// optionPattern matches options like (type:string) or (expires:3600)
var optionPattern = regexp.MustCompile(`^\(([^:]+):([^)]+)\)$`)

// Normalizer handles command normalization to canonical form
type Normalizer struct {
	// escape sequences to process
	escapeSequences map[string]string
}

// NewNormalizer creates a new Normalizer
func NewNormalizer() *Normalizer {
	return &Normalizer{
		escapeSequences: map[string]string{
			"\\#":  "#",
			"\\@":  "@",
			"\\$":  "$",
			"\\%":  "%",
			"\\\\": "\\",
		},
	}
}

// Normalize converts a parsed command to canonical form
func (n *Normalizer) Normalize(parsed *RecognizeResult) *NormalizedCommand {
	if parsed == nil {
		return nil
	}

	nc := &NormalizedCommand{
		OriginalInput:   parsed.OriginalInput,
		IsInternal:      parsed.IsInternal,
		Transformations: []string{},
	}

	// If not an internal command, pass through unchanged
	if !parsed.IsInternal || parsed.Operator == "" {
		nc.Canonical = parsed.OriginalInput
		nc.Operator = ""
		nc.Command = parsed.OriginalInput
		nc.Args = []string{}
		nc.RequiresExecution = false
		return nc
	}

	// Normalize based on operator type
	switch parsed.Operator {
	case OperatorStructured:
		nc.Operator = OperatorStructured
		nc.Command = strings.ToUpper(parsed.Command)
		// Extract options and normalize remaining args
		nc.Options, nc.Args = n.extractOptions(parsed.Args)
		nc.Args = n.normalizeArgs(nc.Args)
		nc.Transformations = append(nc.Transformations, "case_fold_command")

	case OperatorAlias:
		nc.Operator = OperatorAlias
		nc.Command = parsed.Command
		nc.Args = n.normalizeArgs(parsed.Args)

	case OperatorUserVariable:
		nc.Operator = OperatorUserVariable
		nc.Command = n.normalizeVariableName(parsed.Command)
		nc.Args = n.normalizeArgs(parsed.Args)
		nc.Transformations = append(nc.Transformations, "expand_shorthand")

	case OperatorSystemVariable:
		nc.Operator = OperatorSystemVariable
		nc.Command = n.normalizeSystemVariable(parsed.Command)
		nc.Args = n.normalizeArgs(parsed.Args)
	}

	// Build canonical form
	nc.Canonical = n.buildCanonical(nc.Operator, nc.Command, nc.Args)

	// Determine if execution is required
	nc.RequiresExecution = n.requiresExecution(nc)

	return nc
}

// extractOptions extracts options like (type:string) from args and returns options map and remaining args
func (n *Normalizer) extractOptions(args []string) (map[string]string, []string) {
	options := make(map[string]string)
	var remaining []string

	for _, arg := range args {
		if match := optionPattern.FindStringSubmatch(arg); match != nil {
			options[match[1]] = match[2]
		} else {
			remaining = append(remaining, arg)
		}
	}

	return options, remaining
}

// normalizeArgs normalizes command arguments
func (n *Normalizer) normalizeArgs(args []string) []string {
	result := make([]string, len(args))
	for i, arg := range args {
		result[i] = n.normalizeArgument(arg)
	}
	return result
}

// normalizeArgument normalizes a single argument
func (n *Normalizer) normalizeArgument(arg string) string {
	// Process escape sequences
	result := n.resolveEscapes(arg)

	// Collapse whitespace (but preserve quoted strings)
	result = n.collapseWhitespace(result)

	// Trim each argument
	result = strings.TrimSpace(result)

	return result
}

// resolveEscapes converts escape sequences to their literal form
func (n *Normalizer) resolveEscapes(input string) string {
	result := input

	for escape, literal := range n.escapeSequences {
		result = strings.ReplaceAll(result, escape, literal)
	}

	return result
}

// collapseWhitespace collapses multiple whitespace to single space
// but preserves content inside quotes
func (n *Normalizer) collapseWhitespace(input string) string {
	var result strings.Builder
	inQuotes := false
	quoteChar := rune(0)
	prevWasSpace := false

	for _, ch := range input {
		if !inQuotes && (ch == '"' || ch == '\'') {
			inQuotes = true
			quoteChar = ch
			result.WriteRune(ch)
			prevWasSpace = false
			continue
		}

		if inQuotes && ch == quoteChar {
			inQuotes = false
			quoteChar = 0
		}

		if !inQuotes && unicode.IsSpace(ch) {
			if !prevWasSpace && result.Len() > 0 {
				result.WriteRune(' ')
				prevWasSpace = true
			}
			continue
		}

		result.WriteRune(ch)
		prevWasSpace = false
	}

	return strings.TrimSpace(result.String())
}

// normalizeVariableName normalizes a variable name
func (n *Normalizer) normalizeVariableName(name string) string {
	// Expand $name to ${name} for canonical form
	if !strings.HasPrefix(name, "{") {
		return "${" + name + "}"
	}
	return name
}

// normalizeSystemVariable normalizes a system variable reference
func (n *Normalizer) normalizeSystemVariable(name string) string {
	// Check if it's a numeric form - keep as-is
	isNumeric := true
	for _, c := range name {
		if c < '0' || c > '9' {
			isNumeric = false
			break
		}
	}

	if isNumeric {
		return name
	}

	// Named form - uppercase for consistency
	return strings.ToUpper(name)
}

// buildCanonical builds the canonical command string
func (n *Normalizer) buildCanonical(op OperatorFamily, cmd string, args []string) string {
	var parts []string

	if op != "" {
		parts = append(parts, string(op))
	}

	parts = append(parts, cmd)
	parts = append(parts, args...)

	return strings.Join(parts, " ")
}

// requiresExecution determines if a command requires backend execution
func (n *Normalizer) requiresExecution(nc *NormalizedCommand) bool {
	if !nc.IsInternal {
		return false
	}

	// Structured directives typically require execution
	if nc.Operator == OperatorStructured {
		// Some commands like ELSE, ENDIF don't need execution
		noExecCommands := map[string]bool{
			"ELSE":   true,
			"ENDIF":  true,
			"EXPAND": false, // This is internal only
		}
		if noExecCommands[nc.Command] {
			return false
		}
		return true
	}

	// Variables and aliases require resolution
	if nc.Operator == OperatorUserVariable || nc.Operator == OperatorAlias {
		return true
	}

	// System variables may need resolution
	if nc.Operator == OperatorSystemVariable {
		return true
	}

	return false
}

// NormalizeInput normalizes raw input string (used for preview)
func (n *Normalizer) NormalizeInput(input string) string {
	input = strings.TrimSpace(input)
	input = n.resolveEscapes(input)
	input = n.collapseWhitespace(input)
	return input
}

// GetEscapeSequences returns the escape sequence map
func (n *Normalizer) GetEscapeSequences() map[string]string {
	seq := make(map[string]string)
	for k, v := range n.escapeSequences {
		seq[k] = v
	}
	return seq
}

// AddEscapeSequence adds a custom escape sequence
func (n *Normalizer) AddEscapeSequence(escape string, literal string) {
	n.escapeSequences[escape] = literal
}

// IsEscapeSequence checks if a string is an escape sequence
func (n *Normalizer) IsEscapeSequence(s string) bool {
	_, ok := n.escapeSequences[s]
	return ok
}

// NormalizeForStorage normalizes a command for storage (preserves escapes)
func (n *Normalizer) NormalizeForStorage(input string) string {
	// For storage, we want to preserve escape sequences
	// Just trim and collapse whitespace
	input = strings.TrimSpace(input)
	input = n.collapseWhitespace(input)
	return input
}

// DetectOperatorCaseMismatch detects if operator case differs from canonical
func (n *Normalizer) DetectOperatorCaseMismatch(input string) bool {
	// Check for # commands with lowercase
	lowerMatch := regexp.MustCompile(`^#[a-z]`)
	return lowerMatch.MatchString(input)
}
