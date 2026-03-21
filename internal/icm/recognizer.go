package icm

import (
	"strings"
	"unicode"
)

// Recognizer handles tokenization and operator recognition
type Recognizer struct {
	// Reserved operator characters
	reservedOperators map[rune]OperatorFamily
}

// NewRecognizer creates a new Recognizer
func NewRecognizer() *Recognizer {
	return &Recognizer{
		reservedOperators: map[rune]OperatorFamily{
			'#': OperatorStructured,
			'@': OperatorAlias,
			'$': OperatorUserVariable,
			'%': OperatorSystemVariable,
		},
	}
}

// RecognizeResult contains the result of recognition
type RecognizeResult struct {
	Operator      OperatorFamily
	Command       string
	Args          []string
	IsInternal    bool
	Tokens        []Token
	OriginalInput string
}

// Recognize tokenizes input and identifies the operator family
func (r *Recognizer) Recognize(input string) (*RecognizeResult, *ICMError) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, NewValidationError(E2000SyntaxError, "recognize", input, 0)
	}

	// Find the operator
	operator, opIndex := r.findOperator(input)
	if operator == "" {
		// No recognized operator - not an internal command
		return &RecognizeResult{
			Operator:      "",
			Command:       input,
			Args:          []string{},
			IsInternal:    false,
			Tokens:        r.tokenize(input),
			OriginalInput: input,
		}, nil
	}

	// Extract command and arguments based on operator type
	var command string
	var args []string

	switch operator {
	case OperatorStructured:
		command, args = r.parseStructured(input, opIndex)
	case OperatorAlias:
		command, args = r.parseAlias(input, opIndex)
	case OperatorUserVariable:
		command, args = r.parseUserVariable(input, opIndex)
	case OperatorSystemVariable:
		command, args = r.parseSystemVariable(input, opIndex)
	}

	return &RecognizeResult{
		Operator:      operator,
		Command:       command,
		Args:          args,
		IsInternal:    true,
		Tokens:        r.tokenize(input),
		OriginalInput: input,
	}, nil
}

// findOperator finds the first reserved operator character in the input
func (r *Recognizer) findOperator(input string) (OperatorFamily, int) {
	// Check first character for operator
	if len(input) > 0 {
		if op, ok := r.reservedOperators[rune(input[0])]; ok {
			return op, 0
		}
	}

	// Scan for operator characters
	for i, ch := range input {
		if ch == '\\' {
			// Skip escaped character
			if i+1 < len(input) {
				i++
			}
			continue
		}
		if op, ok := r.reservedOperators[ch]; ok {
			// Only recognize operator at start or after whitespace
			if i == 0 || (i > 0 && unicode.IsSpace(rune(input[i-1]))) {
				return op, i
			}
		}
	}

	return "", -1
}

// parseStructured parses a structured directive (# command)
func (r *Recognizer) parseStructured(input string, opIndex int) (string, []string) {
	// Skip the # character
	rest := strings.TrimSpace(input[opIndex+1:])

	// Split into command and arguments
	parts := r.splitArgs(rest)

	if len(parts) == 0 {
		return "", []string{}
	}

	command := strings.ToUpper(parts[0])
	args := parts[1:]

	return command, args
}

// parseAlias parses an alias reference (@aliasname)
func (r *Recognizer) parseAlias(input string, opIndex int) (string, []string) {
	// Skip the @ character
	rest := strings.TrimSpace(input[opIndex+1:])

	// Alias name is the first "word" (non-whitespace)
	parts := r.splitArgs(rest)

	if len(parts) == 0 {
		return "", []string{}
	}

	aliasName := parts[0]
	args := parts[1:]

	return aliasName, args
}

// parseUserVariable parses a user variable ($var or ${var})
func (r *Recognizer) parseUserVariable(input string, opIndex int) (string, []string) {
	// Skip the $ character
	rest := strings.TrimSpace(input[opIndex+1:])

	// Check for ${} syntax
	if strings.HasPrefix(rest, "{") {
		// Find matching closing brace
		closeIdx := strings.Index(rest, "}")
		if closeIdx == -1 {
			// No closing brace - treat entire rest as variable name
			return rest, []string{}
		}
		varName := rest[1:closeIdx]
		remaining := strings.TrimSpace(rest[closeIdx+1:])
		args := r.splitArgs(remaining)
		return varName, args
	}

	// Simple $name syntax - get first word
	parts := r.splitArgs(rest)
	if len(parts) == 0 {
		return "", []string{}
	}

	return parts[0], parts[1:]
}

// parseSystemVariable parses a system variable (%name or %n)
func (r *Recognizer) parseSystemVariable(input string, opIndex int) (string, []string) {
	// Skip the % character
	rest := strings.TrimSpace(input[opIndex+1:])

	// Check for numeric form (%1, %42)
	if len(rest) > 0 && unicode.IsDigit(rune(rest[0])) {
		// Read consecutive digits
		i := 0
		for ; i < len(rest) && unicode.IsDigit(rune(rest[i])); i++ {
		}
		numStr := rest[:i]
		remaining := strings.TrimSpace(rest[i:])
		args := r.splitArgs(remaining)
		return numStr, args
	}

	// Named form (%TIME, %CHARACTER)
	parts := r.splitArgs(rest)
	if len(parts) == 0 {
		return "", []string{}
	}

	varName := parts[0]
	args := parts[1:]

	return varName, args
}

// splitArgs splits a string into arguments, respecting quoted strings
func (r *Recognizer) splitArgs(input string) []string {
	var args []string
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)

	for _, ch := range input {
		if !inQuotes && (ch == '"' || ch == '\'') {
			inQuotes = true
			quoteChar = ch
		} else if inQuotes && ch == quoteChar {
			inQuotes = false
			quoteChar = 0
		} else if !inQuotes && unicode.IsSpace(ch) {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

// tokenize creates tokens from input string
func (r *Recognizer) tokenize(input string) []Token {
	var tokens []Token
	pos := 0

	for pos < len(input) {
		ch := rune(input[pos])

		if ch == '\\' {
			// Escape sequence
			if pos+1 < len(input) {
				tokens = append(tokens, Token{
					Type:     TokenArgument,
					Value:    string(input[pos : pos+2]),
					Position: pos,
				})
				pos += 2
			} else {
				tokens = append(tokens, Token{
					Type:     TokenArgument,
					Value:    string(ch),
					Position: pos,
				})
				pos++
			}
			continue
		}

		if op, ok := r.reservedOperators[ch]; ok {
			tokens = append(tokens, Token{
				Type:     TokenOperator,
				Value:    string(op),
				Position: pos,
			})
			pos++
			continue
		}

		if unicode.IsSpace(ch) {
			start := pos
			for pos < len(input) && unicode.IsSpace(rune(input[pos])) {
				pos++
			}
			tokens = append(tokens, Token{
				Type:     TokenWhitespace,
				Value:    input[start:pos],
				Position: start,
			})
			continue
		}

		if ch == '=' {
			tokens = append(tokens, Token{
				Type:     TokenEquals,
				Value:    "=",
				Position: pos,
			})
			pos++
			continue
		}

		// Regular argument
		start := pos
		for pos < len(input) {
			nextCh := rune(input[pos])
			if _, isOp := r.reservedOperators[nextCh]; isOp {
				break
			}
			if unicode.IsSpace(nextCh) {
				break
			}
			if nextCh == '=' {
				break
			}
			pos++
		}

		if start < pos {
			tokens = append(tokens, Token{
				Type:     TokenArgument,
				Value:    input[start:pos],
				Position: start,
			})
		}
	}

	tokens = append(tokens, Token{
		Type:     TokenEOF,
		Value:    "",
		Position: len(input),
	})

	return tokens
}

// IsReservedOperator checks if a character is a reserved operator
func (r *Recognizer) IsReservedOperator(ch rune) bool {
	_, ok := r.reservedOperators[ch]
	return ok
}

// GetOperatorFamilies returns all registered operator families
func (r *Recognizer) GetOperatorFamilies() map[rune]OperatorFamily {
	result := make(map[rune]OperatorFamily)
	for k, v := range r.reservedOperators {
		result[k] = v
	}
	return result
}
