package icm

import (
	"fmt"
	"sync"
)

// Registry stores command definitions and metadata
type Registry struct {
	// Structured commands (prefix #)
	structured map[string]CommandHandler

	// Aliases (prefix @) - stored as name -> alias data
	aliases map[string]*AliasDefinition

	// User variables (prefix $) - stored as name -> variable data
	userVariables map[string]*VariableDefinition

	// System variables (prefix %)
	systemVariables map[string]*SystemVariableDefinition

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// AliasDefinition defines an alias
type AliasDefinition struct {
	Name        string
	Expansion   string
	Description string
	CreatedAt   int64
}

// VariableDefinition defines a user variable
type VariableDefinition struct {
	Name       string
	Value      string
	IsReadOnly bool
}

// SystemVariableDefinition defines a system variable
type SystemVariableDefinition struct {
	Name        string
	Description string
	IsReadOnly  bool
	Getter      func() string
}

// NewRegistry creates a new command registry
func NewRegistry() *Registry {
	r := &Registry{
		structured:      make(map[string]CommandHandler),
		aliases:         make(map[string]*AliasDefinition),
		userVariables:   make(map[string]*VariableDefinition),
		systemVariables: make(map[string]*SystemVariableDefinition),
	}

	// Register default system variables
	r.registerDefaultSystemVariables()

	// Register built-in commands
	r.registerBuiltInCommands()

	return r
}

// registerDefaultSystemVariables registers the default system variables
func (r *Registry) registerDefaultSystemVariables() {
	defaults := []struct {
		name        string
		description string
		getter      func() string
	}{
		{"TIME", "Current time", func() string { return "00:00" }},
		{"DATE", "Current date", func() string { return "2026-01-01" }},
		{"CHARACTER", "Current character name", func() string { return "" }},
		{"SERVER", "Current server name", func() string { return "" }},
		{"SESSIONID", "Current session ID", func() string { return "" }},
		{"HOST", "Current host", func() string { return "" }},
		{"PORT", "Current port", func() string { return "" }},
	}

	for _, sv := range defaults {
		r.systemVariables[sv.name] = &SystemVariableDefinition{
			Name:        sv.name,
			Description: sv.description,
			IsReadOnly:  true,
			Getter:      sv.getter,
		}
	}
}

// registerBuiltInCommands registers the built-in ICM commands
func (r *Registry) registerBuiltInCommands() {
	// These will be populated when we implement the dispatcher
	// For now, the registry just holds the structure
}

// RegisterStructuredCommand registers a structured directive handler
func (r *Registry) RegisterStructuredCommand(name string, handler CommandHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	upperName := name
	if upperName == "" {
		return fmt.Errorf("command name cannot be empty")
	}

	if _, exists := r.structured[upperName]; exists {
		return fmt.Errorf("command %s already registered", name)
	}

	r.structured[upperName] = handler
	return nil
}

// GetStructuredCommand returns a registered structured command handler
func (r *Registry) GetStructuredCommand(name string) (CommandHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, exists := r.structured[name]
	return handler, exists
}

// GetAllStructuredCommands returns all registered structured commands
func (r *Registry) GetAllStructuredCommands() map[string]CommandHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]CommandHandler)
	for k, v := range r.structured {
		result[k] = v
	}
	return result
}

// RegisterAlias registers an alias
func (r *Registry) RegisterAlias(name string, expansion string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if name == "" {
		return fmt.Errorf("alias name cannot be empty")
	}

	r.aliases[name] = &AliasDefinition{
		Name:      name,
		Expansion: expansion,
		CreatedAt: 0, // Would be set to current timestamp
	}

	return nil
}

// GetAlias returns an alias by name
func (r *Registry) GetAlias(name string) (*AliasDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	alias, exists := r.aliases[name]
	return alias, exists
}

// GetAllAliases returns all registered aliases
func (r *Registry) GetAllAliases() map[string]*AliasDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*AliasDefinition)
	for k, v := range r.aliases {
		result[k] = v
	}
	return result
}

// DeleteAlias removes an alias
func (r *Registry) DeleteAlias(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.aliases[name]; exists {
		delete(r.aliases, name)
		return true
	}
	return false
}

// SetUserVariable sets a user variable value
func (r *Registry) SetUserVariable(name string, value string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.userVariables[name] = &VariableDefinition{
		Name:  name,
		Value: value,
	}
}

// GetUserVariable returns a user variable value
func (r *Registry) GetUserVariable(name string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	variable, exists := r.userVariables[name]
	if !exists {
		return "", false
	}
	return variable.Value, true
}

// GetAllUserVariables returns all user variables
func (r *Registry) GetAllUserVariables() map[string]*VariableDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*VariableDefinition)
	for k, v := range r.userVariables {
		result[k] = v
	}
	return result
}

// DeleteUserVariable removes a user variable
func (r *Registry) DeleteUserVariable(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.userVariables[name]; exists {
		delete(r.userVariables, name)
		return true
	}
	return false
}

// RegisterSystemVariable registers a system variable
func (r *Registry) RegisterSystemVariable(name string, definition *SystemVariableDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if name == "" {
		return fmt.Errorf("system variable name cannot be empty")
	}

	r.systemVariables[name] = definition
	return nil
}

// GetSystemVariable returns a system variable value
func (r *Registry) GetSystemVariable(name string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	variable, exists := r.systemVariables[name]
	if !exists {
		return "", false
	}

	if variable.Getter != nil {
		return variable.Getter(), true
	}
	return "", true
}

// GetAllSystemVariables returns all system variables
func (r *Registry) GetAllSystemVariables() map[string]*SystemVariableDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*SystemVariableDefinition)
	for k, v := range r.systemVariables {
		result[k] = v
	}
	return result
}

// ClearUserVariables clears all user variables (session reset)
func (r *Registry) ClearUserVariables() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.userVariables = make(map[string]*VariableDefinition)
}

// ClearSessionVariables clears session-scoped variables (%n numeric)
func (r *Registry) ClearSessionVariables() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Only keep system variables (named %variables)
	// Numeric session variables would be stored separately
}

// HasCommand checks if a command exists
func (r *Registry) HasCommand(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.structured[name]
	return exists
}

// HasAlias checks if an alias exists
func (r *Registry) HasAlias(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.aliases[name]
	return exists
}

// HasSystemVariable checks if a system variable exists
func (r *Registry) HasSystemVariable(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.systemVariables[name]
	return exists
}

// CommandCount returns the number of registered commands
func (r *Registry) CommandCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.structured)
}

// AliasCount returns the number of registered aliases
func (r *Registry) AliasCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.aliases)
}
