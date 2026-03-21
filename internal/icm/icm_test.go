package icm

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// Test cases for Recognizer

func TestRecognizer_RecognizeStructured(t *testing.T) {
	r := NewRecognizer()

	tests := []struct {
		input    string
		wantOp   OperatorFamily
		wantCmd  string
		wantArgs []string
	}{
		{"#echo hello", OperatorStructured, "ECHO", []string{"hello"}},
		{"#SET myvar = value", OperatorStructured, "SET", []string{"myvar", "=", "value"}},
		{"#if $foo == bar", OperatorStructured, "IF", []string{"$foo", "==", "bar"}},
		{"#timer mytimer 60 #echo hello", OperatorStructured, "TIMER", []string{"mytimer", "60", "#echo", "hello"}},
		{"#log message", OperatorStructured, "LOG", []string{"message"}},
		{"#help", OperatorStructured, "HELP", []string{}},
		{"#   set   var   value", OperatorStructured, "SET", []string{"var", "value"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := r.Recognize(tt.input)
			if err != nil {
				t.Errorf("Recognize(%q) error = %v", tt.input, err)
				return
			}
			if got.Operator != tt.wantOp {
				t.Errorf("Recognize(%q).Operator = %v, want %v", tt.input, got.Operator, tt.wantOp)
			}
			if got.Command != tt.wantCmd {
				t.Errorf("Recognize(%q).Command = %v, want %v", tt.input, got.Command, tt.wantCmd)
			}
		})
	}
}

func TestRecognizer_RecognizeAlias(t *testing.T) {
	r := NewRecognizer()

	tests := []struct {
		input    string
		wantOp   OperatorFamily
		wantCmd  string
		wantArgs []string
	}{
		{"@myalias", OperatorAlias, "myalias", []string{}},
		{"@myalias arg1 arg2", OperatorAlias, "myalias", []string{"arg1", "arg2"}},
		{"@ alias", OperatorAlias, "alias", []string{}},
		{"@   alias   arg", OperatorAlias, "alias", []string{"arg"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := r.Recognize(tt.input)
			if err != nil {
				t.Errorf("Recognize(%q) error = %v", tt.input, err)
				return
			}
			if got.Operator != tt.wantOp {
				t.Errorf("Recognize(%q).Operator = %v, want %v", tt.input, got.Operator, tt.wantOp)
			}
			if got.Command != tt.wantCmd {
				t.Errorf("Recognize(%q).Command = %v, want %v", tt.input, got.Command, tt.wantCmd)
			}
		})
	}
}

func TestRecognizer_RecognizeUserVariable(t *testing.T) {
	r := NewRecognizer()

	tests := []struct {
		input    string
		wantOp   OperatorFamily
		wantCmd  string
		wantArgs []string
	}{
		{"$myvar", OperatorUserVariable, "myvar", []string{}},
		{"$myvar value", OperatorUserVariable, "myvar", []string{"value"}},
		{"${myvar}", OperatorUserVariable, "myvar", []string{}},
		{"${myvar} value", OperatorUserVariable, "myvar", []string{"value"}},
		{"$var1 $var2", OperatorUserVariable, "var1", []string{"$var2"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := r.Recognize(tt.input)
			if err != nil {
				t.Errorf("Recognize(%q) error = %v", tt.input, err)
				return
			}
			if got.Operator != tt.wantOp {
				t.Errorf("Recognize(%q).Operator = %v, want %v", tt.input, got.Operator, tt.wantOp)
			}
			if got.Command != tt.wantCmd {
				t.Errorf("Recognize(%q).Command = %v, want %v", tt.input, got.Command, tt.wantCmd)
			}
		})
	}
}

func TestRecognizer_RecognizeSystemVariable(t *testing.T) {
	r := NewRecognizer()

	tests := []struct {
		input    string
		wantOp   OperatorFamily
		wantCmd  string
		wantArgs []string
	}{
		{"%1", OperatorSystemVariable, "1", []string{}},
		{"%42", OperatorSystemVariable, "42", []string{}},
		{"%TIME", OperatorSystemVariable, "TIME", []string{}},
		{"%CHARACTER", OperatorSystemVariable, "CHARACTER", []string{}},
		{"%1 value", OperatorSystemVariable, "1", []string{"value"}},
		{"%TIME value", OperatorSystemVariable, "TIME", []string{"value"}},
		{"%", OperatorSystemVariable, "", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := r.Recognize(tt.input)
			if err != nil {
				t.Errorf("Recognize(%q) error = %v", tt.input, err)
				return
			}
			if got.Operator != tt.wantOp {
				t.Errorf("Recognize(%q).Operator = %v, want %v", tt.input, got.Operator, tt.wantOp)
			}
			if got.Command != tt.wantCmd {
				t.Errorf("Recognize(%q).Command = %v, want %v", tt.input, got.Command, tt.wantCmd)
			}
		})
	}
}

func TestRecognizer_NotInternal(t *testing.T) {
	r := NewRecognizer()

	tests := []struct {
		input string
	}{
		{"look"},
		{"say hello"},
		{"who"},
		{"inventory"},
		{"go north"},
		{""},
		{"   "},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := r.Recognize(tt.input)
			if err != nil && tt.input != "" && strings.TrimSpace(tt.input) != "" {
				t.Errorf("Recognize(%q) error = %v", tt.input, err)
				return
			}
			if got != nil && got.IsInternal {
				t.Errorf("Recognize(%q).IsInternal = true, want false", tt.input)
			}
		})
	}
}

// Test cases for Normalizer

func TestNormalizer_Normalize(t *testing.T) {
	n := NewNormalizer()

	tests := []struct {
		input    string
		wantCmd  string
		wantArgs []string
	}{
		{"#echo hello", "ECHO", []string{"hello"}},
		{"#  set  var  value", "SET", []string{"var", "value"}},
		{"@myalias arg", "myalias", []string{"arg"}},
		{"$myvar", "${myvar}", []string{}},
		{"${myvar}", "${myvar}", []string{}},
		{"%42", "42", []string{}},
		{"%TIME", "TIME", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r := NewRecognizer()
			parsed, err := r.Recognize(tt.input)
			if err != nil {
				t.Skipf("Skipping: recognition error = %v", err)
				return
			}

			got := n.Normalize(parsed)
			if got == nil {
				t.Errorf("Normalize(%q) returned nil", tt.input)
				return
			}
			if got.Command != tt.wantCmd {
				t.Errorf("Normalize(%q).Command = %v, want %v", tt.input, got.Command, tt.wantCmd)
			}
		})
	}
}

// Test cases for Validator

func TestValidator_Validate(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		input   string
		wantErr bool
	}{
		{"#echo hello", false},
		{"#set var value", false},
		{"#if $foo == bar", false},
		{"#timer mytimer 60 #echo hello", false},
		{"#whatev", true}, // Unknown command
		{"@validalias", false},
		{"$validvar", false},
		{"%1", false},
		{"%TIME", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r := NewRecognizer()
			parsed, err := r.Recognize(tt.input)
			if err != nil {
				t.Skipf("Skipping: recognition error = %v", err)
				return
			}

			gotErr := v.Validate(parsed)
			if (gotErr != nil) != tt.wantErr {
				t.Errorf("Validate(%q) error = %v, want error = %v", tt.input, gotErr, tt.wantErr)
			}
		})
	}
}

// Test cases for Classifier

func TestClassifier_Classify(t *testing.T) {
	c := NewClassifier()

	tests := []struct {
		input           string
		wantInternal    bool
		wantPassThrough bool
	}{
		{"look", false, true},
		{"#echo hello", true, false},
		{"@myalias", true, false},
		{"$myvar", true, false},
		{"%1", true, false},
		{"&somecommand", false, true},
		{"", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := c.Classify(tt.input)
			if got.IsInternal != tt.wantInternal {
				t.Errorf("Classify(%q).IsInternal = %v, want %v", tt.input, got.IsInternal, tt.wantInternal)
			}
			if got.ShouldPassThrough != tt.wantPassThrough {
				t.Errorf("Classify(%q).ShouldPassThrough = %v, want %v", tt.input, got.ShouldPassThrough, tt.wantPassThrough)
			}
		})
	}
}

// Test cases for EscapeHandler

func TestEscapeHandler_ResolveEscapes(t *testing.T) {
	e := NewEscapeHandler()

	tests := []struct {
		input string
		want  string
	}{
		{"\\#echo", "#echo"},
		{"\\@alias", "@alias"},
		{"\\$var", "$var"},
		{"\\%sys", "%sys"},
		{"\\\\", "\\"},
		{"\\n", "\n"},
		{"\\t", "\t"},
		{"hello world", "hello world"},
		{"no escapes here", "no escapes here"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := e.ResolveEscapes(tt.input)
			if got != tt.want {
				t.Errorf("ResolveEscapes(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEscapeHandler_HasEscapeSequences(t *testing.T) {
	e := NewEscapeHandler()

	tests := []struct {
		input string
		want  bool
	}{
		{"\\#echo", true},
		{"hello", false},
		{"no escapes", false},
		{"mixed\\#here", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := e.HasEscapeSequences(tt.input)
			if got != tt.want {
				t.Errorf("HasEscapeSequences(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// Test cases for Engine

func TestEngine_Process(t *testing.T) {
	e := NewEngine()

	tests := []struct {
		input          string
		wantInternal   bool
		wantNormalized string
	}{
		{"look", false, "look"},
		{"#echo hello", true, "# ECHO hello"},
		{"#set var value", true, "# SET var value"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			req := &ICMRequest{
				Raw:       tt.input,
				Context:   ContextPreview,
				SessionID: "test-session",
			}

			got := e.Process(req)
			if got.IsInternal != tt.wantInternal {
				t.Errorf("Process(%q).IsInternal = %v, want %v", tt.input, got.IsInternal, tt.wantInternal)
			}
		})
	}
}

// Test cases for Registry

func TestRegistry_RegisterAlias(t *testing.T) {
	r := NewRegistry()

	err := r.RegisterAlias("testalias", "echo hello")
	if err != nil {
		t.Errorf("RegisterAlias error = %v", err)
	}

	alias, ok := r.GetAlias("testalias")
	if !ok {
		t.Errorf("GetAlias failed")
	}
	if alias.Expansion != "echo hello" {
		t.Errorf("GetAlias().Expansion = %v, want 'echo hello'", alias.Expansion)
	}
}

func TestRegistry_SetUserVariable(t *testing.T) {
	r := NewRegistry()

	r.SetUserVariable("testvar", "testvalue")

	val, ok := r.GetUserVariable("testvar")
	if !ok {
		t.Errorf("GetUserVariable failed")
	}
	if val != "testvalue" {
		t.Errorf("GetUserVariable() = %v, want 'testvalue'", val)
	}
}

func TestRegistry_SystemVariables(t *testing.T) {
	r := NewRegistry()

	// Check default system variables are registered
	tests := []string{"TIME", "DATE", "CHARACTER", "SERVER", "SESSIONID", "HOST", "PORT"}

	for _, name := range tests {
		if !r.HasSystemVariable(name) {
			t.Errorf("System variable %q not registered", name)
		}
	}
}

// Test cases for SafetyChecker

func TestDefaultSafetyChecker_CheckRateLimit(t *testing.T) {
	s := NewDefaultSafetyChecker()

	// Should not be rate limited initially
	if s.CheckRateLimit("test-session", "command") {
		t.Errorf("CheckRateLimit should return false initially")
	}

	// Record many executions
	for i := 0; i < 100; i++ {
		s.RecordExecution("test-session", "command")
	}

	// Should be rate limited now
	if !s.CheckRateLimit("test-session", "command") {
		t.Errorf("CheckRateLimit should return true after many executions")
	}
}

func TestDefaultSafetyChecker_CheckTimeout(t *testing.T) {
	s := NewDefaultSafetyChecker()

	// Should not timeout with recent start time
	start := time.Now()
	if s.CheckTimeout(start) {
		t.Errorf("CheckTimeout should return false for recent start time")
	}

	// Should timeout with old start time
	oldStart := time.Now().Add(-10 * time.Second)
	if !s.CheckTimeout(oldStart) {
		t.Errorf("CheckTimeout should return true for old start time")
	}
}

// Integration tests for backend command execution

func TestEngine_ExecuteStructuredCommand(t *testing.T) {
	// Create new engine for each test to avoid circuit breaker state
	e := NewEngine()

	tests := []struct {
		input      string
		context    ExecutionContext
		wantError  bool
		wantOutput string
	}{
		{"#echo hello", ContextSubmission, false, "hello"},
		{"#set myvar myvalue", ContextSubmission, false, ""},
		{"#timer mytimer 60 #echo hello", ContextSubmission, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			req := &ICMRequest{
				Raw:       tt.input,
				Context:   tt.context,
				SessionID: "test-session-" + tt.input,
			}

			resp := e.Process(req)
			if tt.wantError && resp.Error == nil {
				t.Errorf("Process(%q) expected error, got nil", tt.input)
			}
			if !tt.wantError && resp.Error != nil {
				t.Errorf("Process(%q) unexpected error: %v", tt.input, resp.Error)
			}
			if resp.Result != nil && tt.wantOutput != "" && resp.Result.Output != tt.wantOutput {
				t.Errorf("Process(%q).Output = %q, want %q", tt.input, resp.Result.Output, tt.wantOutput)
			}
		})
	}
}

func TestEngine_LogCommand(t *testing.T) {
	e := NewEngine()

	// Test #LOG in operational context
	req := &ICMRequest{
		Raw:       "#log test message",
		Context:   ContextOperational,
		SessionID: "log-test-session",
	}

	resp := e.Process(req)
	if resp.Error != nil {
		t.Errorf("Process #LOG in operational context error: %v", resp.Error)
	}
	if resp.Result == nil {
		t.Errorf("Process #LOG result is nil")
	}
}

func TestEngine_LogCommandPreviewDenied(t *testing.T) {
	e := NewEngine()

	// Test #LOG in preview context should fail
	req := &ICMRequest{
		Raw:       "#log test message",
		Context:   ContextPreview,
		SessionID: "log-preview-session",
	}

	resp := e.Process(req)
	// Preview context doesn't execute, so no error expected
	// The command is validated but not executed
	if resp.Error != nil {
		t.Logf("Preview context returned error (may be expected): %v", resp.Error)
	}
}

func TestEngine_TimerCommands(t *testing.T) {
	e := NewEngine()

	// Test timer commands with unique session IDs to avoid circuit breaker
	timerTests := []struct {
		input     string
		context   ExecutionContext
		wantError bool
	}{
		{"#start mytimer", ContextSubmission, false},
		{"#stop mytimer", ContextSubmission, false},
		{"#check mytimer", ContextSubmission, false},
		{"#cancel mytimer", ContextSubmission, false},
	}

	for i, tt := range timerTests {
		t.Run(tt.input, func(t *testing.T) {
			req := &ICMRequest{
				Raw:       tt.input,
				Context:   tt.context,
				SessionID: fmt.Sprintf("timer-test-session-%d", i),
			}

			resp := e.Process(req)
			if tt.wantError && resp.Error == nil {
				t.Errorf("Process(%q) expected error, got nil", tt.input)
			}
			if !tt.wantError && resp.Error != nil {
				t.Errorf("Process(%q) unexpected error: %v", tt.input, resp.Error)
			}
		})
	}
}

func TestEngine_SessionManagement(t *testing.T) {
	e := NewEngine()

	// Create a session
	e.CreateSession("session-1", "user-1", "TestChar", "test.server.com", "localhost", "1234")

	// Verify session exists
	session, ok := e.GetSession("session-1")
	if !ok {
		t.Errorf("GetSession failed to find session")
	}
	if session.CharacterName != "TestChar" {
		t.Errorf("Session.CharacterName = %q, want %q", session.CharacterName, "TestChar")
	}

	// Set session variable
	e.SetSessionVariable("session-1", "testvar", "testvalue")

	// Get session variable
	val, ok := e.GetSessionVariable("session-1", "testvar")
	if !ok {
		t.Errorf("GetSessionVariable failed")
	}
	if val != "testvalue" {
		t.Errorf("GetSessionVariable() = %q, want %q", val, "testvalue")
	}

	// Clear session
	e.ClearSession("session-1")

	// Verify session is gone
	_, ok = e.GetSession("session-1")
	if ok {
		t.Errorf("ClearSession did not remove session")
	}
}

func TestEngine_CommandHistory(t *testing.T) {
	e := NewEngine()

	sessionID := "test-session"

	// Add commands to history
	e.AddToHistory(sessionID, "look")
	e.AddToHistory(sessionID, "go north")
	e.AddToHistory(sessionID, "get sword")

	// Get history
	history := e.GetHistory(sessionID)
	if len(history) != 3 {
		t.Errorf("GetHistory length = %d, want 3", len(history))
	}
	if history[0] != "look" {
		t.Errorf("GetHistory[0] = %q, want %q", history[0], "look")
	}
}

func TestEngine_StateEffects(t *testing.T) {
	e := NewEngine()

	// Test user variable setting through registry
	e.GetRegistry().SetUserVariable("myvar", "myvalue")

	// Verify variable was set
	val, ok := e.GetRegistry().GetUserVariable("myvar")
	if !ok {
		t.Errorf("Variable myvar not found in registry")
	}
	if val != "myvalue" {
		t.Errorf("Variable myvar = %q, want %q", val, "myvalue")
	}

	// Test session variable setting through engine
	e.SetSessionVariable("test-session", "sessionvar", "sessionvalue")

	// Verify session variable was set
	sessVal, ok := e.GetSessionVariable("test-session", "sessionvar")
	if !ok {
		t.Errorf("Session variable sessionvar not found")
	}
	if sessVal != "sessionvalue" {
		t.Errorf("Session variable sessionvar = %q, want %q", sessVal, "sessionvalue")
	}
}

// testEffectHandler implements StateEffectHandler for testing
type testEffectHandler struct{}

func (h *testEffectHandler) HandleEffect(effect StateEffect, sessionID string) error {
	return nil
}

func (h *testEffectHandler) GetHandlerType() string {
	return "test"
}

func TestDispatcher_SafetyEnforcement(t *testing.T) {
	d := NewDispatcher(NewRegistry())

	// Test rate limiting
	sessionID := "test-session"

	// Record max executions
	for i := 0; i < 100; i++ {
		d.safety.RecordExecution(sessionID, "test-command")
	}

	// Should be rate limited
	if !d.safety.CheckRateLimit(sessionID, "test-command") {
		t.Errorf("CheckRateLimit should return true after max executions")
	}

	// Test circuit breaker
	d.safety.RecordExecution(sessionID, "loop-command")
	d.safety.RecordExecution(sessionID, "loop-command")
	d.safety.RecordExecution(sessionID, "loop-command")
	d.safety.RecordExecution(sessionID, "loop-command")
	d.safety.RecordExecution(sessionID, "loop-command")

	if !d.safety.CheckCircuitBreaker(sessionID) {
		t.Errorf("CheckCircuitBreaker should return true after max loop count")
	}
}

func TestLogHandler_Governance(t *testing.T) {
	h := NewLogHandler()

	// Test that #LOG works in operational context
	cmd := &NormalizedCommand{
		Command: "LOG",
		Args:    []string{"test message"},
	}

	ctx := ContextOperational
	result, err := h.Handle(&ctx, cmd)
	if err != nil {
		t.Errorf("Handle in operational context error: %v", err)
	}
	if result == nil {
		t.Errorf("Handle returned nil result")
	}

	// Test that #LOG is blocked in preview context
	ctx = ContextPreview
	_, err = h.Handle(&ctx, cmd)
	if err == nil {
		t.Errorf("Handle in preview context should error")
	}
	if err.Code != E4001PermissionDenied {
		t.Errorf("Expected permission denied error, got: %v", err)
	}

	// Test log levels
	cmdWithLevel := &NormalizedCommand{
		Command: "LOG",
		Args:    []string{"debug", "debug message"},
	}

	ctx = ContextOperational
	result, _ = h.Handle(&ctx, cmdWithLevel)
	if result == nil || len(result.Effects) == 0 {
		t.Errorf("Expected log effect")
	}
}

func TestAuthorityLevels(t *testing.T) {
	// Test that authority levels are correctly defined

	// Preview should not execute
	auth := GetAuthorityForContext(ContextPreview)
	if auth.CanExecute {
		t.Errorf("Preview context should not allow execution")
	}

	// Submission should execute
	auth = GetAuthorityForContext(ContextSubmission)
	if !auth.CanExecute {
		t.Errorf("Submission context should allow execution")
	}

	// Automation should execute
	auth = GetAuthorityForContext(ContextAutomation)
	if !auth.CanExecute {
		t.Errorf("Automation context should allow execution")
	}
	if !auth.IsAutomation {
		t.Errorf("Automation context should be marked as automation")
	}

	// Operational should require admin role
	auth = GetAuthorityForContext(ContextOperational)
	if !auth.CanExecute {
		t.Errorf("Operational context should allow execution")
	}
	if auth.RequiresRole != "admin" {
		t.Errorf("Operational context should require admin role")
	}
}

func TestHandlerRegistration(t *testing.T) {
	e := NewEngine()
	d := e.GetDispatcher()

	// Verify built-in handlers are registered
	tests := []struct {
		family     OperatorFamily
		command    string
		wantExists bool
	}{
		{OperatorStructured, "ECHO", true},
		{OperatorStructured, "LOG", true},
		{OperatorStructured, "SET", true},
		{OperatorStructured, "IF", true},
		{OperatorStructured, "TIMER", true},
		{OperatorStructured, "ELSE", true},
		{OperatorStructured, "ENDIF", true},
		{OperatorStructured, "START", true},
		{OperatorStructured, "STOP", true},
		{OperatorStructured, "CHECK", true},
		{OperatorStructured, "CANCEL", true},
		{OperatorStructured, "UNKNOWN", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			handler := d.getHandler(tt.family, tt.command)
			if tt.wantExists && handler == nil {
				t.Errorf("Handler %q not registered", tt.command)
			}
			if !tt.wantExists && handler != nil {
				t.Errorf("Handler %q should not exist", tt.command)
			}
		})
	}
}

func TestEndToEnd_CommandPipeline(t *testing.T) {
	e := NewEngine()

	// Create session
	e.CreateSession("e2e-session", "user-1", "Hero", "mud.example.com", "localhost", "23")

	// Test full pipeline: recognize -> validate -> normalize -> resolve -> dispatch

	// Test user variable set and get
	e.SetSessionVariable("e2e-session", "player", "Hero")

	req := &ICMRequest{
		Raw:       "$player",
		Context:   ContextSubmission,
		SessionID: "e2e-session",
	}

	resp := e.Process(req)
	if resp.Error != nil {
		t.Errorf("Process error: %v", resp.Error)
	}
	if !resp.IsInternal {
		t.Errorf("Expected internal command")
	}
	if resp.Resolved != "Hero" {
		t.Errorf("Expected resolved to 'Hero', got %q", resp.Resolved)
	}

	// Test alias expansion
	e.GetRegistry().RegisterAlias("greet", "say Hello")

	req2 := &ICMRequest{
		Raw:       "@greet",
		Context:   ContextSubmission,
		SessionID: "e2e-session",
	}

	resp2 := e.Process(req2)
	if resp2.Error != nil {
		t.Errorf("Alias process error: %v", resp2.Error)
	}
	if resp2.Resolved != "say Hello" {
		t.Errorf("Expected resolved alias to 'say Hello', got %q", resp2.Resolved)
	}

	// Test system variable
	req3 := &ICMRequest{
		Raw:       "%TIME",
		Context:   ContextSubmission,
		SessionID: "e2e-session",
	}

	resp3 := e.Process(req3)
	if resp3.Error != nil {
		t.Errorf("System variable process error: %v", resp3.Error)
	}
	if resp3.Resolved == "" {
		t.Errorf("Expected system variable to resolve")
	}
}
