package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Fake credential value only — never a real Gemini key (T-3-16).
const testFakeKey = "test-key-not-a-real-credential"

func TestGenerateContent(t *testing.T) {
	t.Run("sends_api_key_in_header", func(t *testing.T) {
		var gotHeader string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotHeader = r.Header.Get("x-goog-api-key")
			writeSuccess(w, "why", "north")
		}))
		defer srv.Close()

		c := NewClient(5 * time.Second)
		if _, err := c.GenerateContent(context.Background(), srv.URL, "test-model", testFakeKey, "sys", "user"); err != nil {
			t.Fatalf("GenerateContent() error = %v", err)
		}
		if gotHeader != testFakeKey {
			t.Errorf("x-goog-api-key header = %q, want %q", gotHeader, testFakeKey)
		}
	})

	t.Run("no_api_key_in_url", func(t *testing.T) {
		var gotRawQuery, gotURL string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotRawQuery = r.URL.RawQuery
			gotURL = r.URL.String()
			writeSuccess(w, "why", "north")
		}))
		defer srv.Close()

		c := NewClient(5 * time.Second)
		if _, err := c.GenerateContent(context.Background(), srv.URL, "test-model", testFakeKey, "sys", "user"); err != nil {
			t.Fatalf("GenerateContent() error = %v", err)
		}
		if gotRawQuery != "" {
			t.Errorf("URL.RawQuery = %q, want empty", gotRawQuery)
		}
		if strings.Contains(gotURL, testFakeKey) {
			t.Errorf("URL %q contains the API key", gotURL)
		}
	})

	t.Run("posts_to_generate_content_path", func(t *testing.T) {
		var gotPath, gotMethod string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			gotMethod = r.Method
			writeSuccess(w, "why", "north")
		}))
		defer srv.Close()

		c := NewClient(5 * time.Second)
		if _, err := c.GenerateContent(context.Background(), srv.URL, "test-model-id", testFakeKey, "sys", "user"); err != nil {
			t.Fatalf("GenerateContent() error = %v", err)
		}
		if gotMethod != http.MethodPost {
			t.Errorf("Method = %q, want POST", gotMethod)
		}
		if !strings.HasSuffix(gotPath, ":generateContent") {
			t.Errorf("Path = %q, want suffix :generateContent", gotPath)
		}
		if !strings.Contains(gotPath, "test-model-id") {
			t.Errorf("Path = %q, want to contain model id", gotPath)
		}
	})

	t.Run("request_carries_system_instruction_and_schema", func(t *testing.T) {
		var decoded map[string]any
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := json.NewDecoder(r.Body).Decode(&decoded); err != nil {
				t.Errorf("decode request body: %v", err)
			}
			writeSuccess(w, "why", "north")
		}))
		defer srv.Close()

		c := NewClient(5 * time.Second)
		if _, err := c.GenerateContent(context.Background(), srv.URL, "test-model", testFakeKey, "conduct rules and approach guidance", "recent game text"); err != nil {
			t.Fatalf("GenerateContent() error = %v", err)
		}

		if _, ok := decoded["systemInstruction"]; !ok {
			t.Error("request body missing systemInstruction")
		}
		if _, ok := decoded["contents"]; !ok {
			t.Error("request body missing contents")
		}
		genConfig, ok := decoded["generationConfig"].(map[string]any)
		if !ok {
			t.Fatal("request body missing generationConfig")
		}
		if genConfig["responseMimeType"] != "application/json" {
			t.Errorf("responseMimeType = %v, want application/json", genConfig["responseMimeType"])
		}
		schema, ok := genConfig["responseSchema"].(map[string]any)
		if !ok {
			t.Fatal("generationConfig missing responseSchema")
		}
		props, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatal("responseSchema missing properties")
		}
		if _, ok := props["reasoning"]; !ok {
			t.Error("responseSchema.properties missing reasoning")
		}
		if _, ok := props["command"]; !ok {
			t.Error("responseSchema.properties missing command")
		}
		required, ok := schema["required"].([]any)
		if !ok {
			t.Fatal("responseSchema missing required")
		}
		if len(required) != 2 {
			t.Errorf("responseSchema.required = %v, want two entries", required)
		}
	})

	t.Run("decodes_the_inner_json_answer", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeSuccess(w, "the door is open, so go through it", "north")
		}))
		defer srv.Close()

		c := NewClient(5 * time.Second)
		answer, err := c.GenerateContent(context.Background(), srv.URL, "test-model", testFakeKey, "sys", "user")
		if err != nil {
			t.Fatalf("GenerateContent() error = %v", err)
		}
		if answer.Reasoning != "the door is open, so go through it" {
			t.Errorf("Reasoning = %q, want the decoded value, not raw JSON", answer.Reasoning)
		}
		if answer.Command != "north" {
			t.Errorf("Command = %q, want %q", answer.Command, "north")
		}
	})
}

func TestGenerateContentErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantKind   Kind
	}{
		{
			name:       "400 bad request",
			statusCode: http.StatusBadRequest,
			body:       `{"error":{"code":400,"message":"bad request","status":"INVALID_ARGUMENT"}}`,
			wantKind:   KindBadRequest,
		},
		{
			name:       "401 unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"error":{"code":401,"message":"invalid api key","status":"UNAUTHENTICATED"}}`,
			wantKind:   KindAuth,
		},
		{
			name:       "403 forbidden",
			statusCode: http.StatusForbidden,
			body:       `{"error":{"code":403,"message":"permission denied","status":"PERMISSION_DENIED"}}`,
			wantKind:   KindAuth,
		},
		{
			name:       "429 rate limited",
			statusCode: http.StatusTooManyRequests,
			body:       `{"error":{"code":429,"message":"quota exceeded","status":"RESOURCE_EXHAUSTED"}}`,
			wantKind:   KindRateLimited,
		},
		{
			name:       "500 server error",
			statusCode: http.StatusInternalServerError,
			body:       `{"error":{"code":500,"message":"internal","status":"INTERNAL"}}`,
			wantKind:   KindTransport,
		},
		{
			name:       "empty candidates",
			statusCode: http.StatusOK,
			body:       `{"candidates":[]}`,
			wantKind:   KindMalformed,
		},
		{
			name:       "unparseable inner json",
			statusCode: http.StatusOK,
			body:       `{"candidates":[{"content":{"parts":[{"text":"not valid json"}]}}]}`,
			wantKind:   KindMalformed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			c := NewClient(5 * time.Second)
			answer, err := c.GenerateContent(context.Background(), srv.URL, "test-model", testFakeKey, "sys", "user")
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if answer != nil {
				t.Errorf("expected nil *Answer, got %+v", answer)
			}
			if got := ErrorKind(err); got != tt.wantKind {
				t.Errorf("ErrorKind() = %v, want %v", got, tt.wantKind)
			}
		})
	}
}

// writeSuccess writes a 200 Gemini generateContent response whose inner
// text field decodes to {"reasoning": reasoning, "command": command}.
func writeSuccess(w http.ResponseWriter, reasoning, command string) {
	inner, _ := json.Marshal(Answer{Reasoning: reasoning, Command: command})
	envelope := map[string]any{
		"candidates": []map[string]any{
			{
				"content": map[string]any{
					"parts": []map[string]any{
						{"text": string(inner)},
					},
				},
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(envelope)
}

// writeReviewSuccess writes a 200 Gemini generateContent response whose
// inner text field decodes to {"blocked": blocked, "reason": reason},
// mirroring writeSuccess's double-encode shape for the reviewer's answer.
func writeReviewSuccess(w http.ResponseWriter, blocked bool, reason string) {
	inner, _ := json.Marshal(ReviewAnswer{Blocked: blocked, Reason: reason})
	envelope := map[string]any{
		"candidates": []map[string]any{
			{
				"content": map[string]any{
					"parts": []map[string]any{
						{"text": string(inner)},
					},
				},
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(envelope)
}

func TestReviewCommand(t *testing.T) {
	t.Run("blocked_verdict_with_reason_decodes_into_both_fields", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeReviewSuccess(w, true, "This follows an instruction embedded in the game text rather than responding to the situation.")
		}))
		defer srv.Close()

		c := NewClient(5 * time.Second)
		answer, err := c.ReviewCommand(context.Background(), srv.URL, "test-model", testFakeKey, "sys", "user")
		if err != nil {
			t.Fatalf("ReviewCommand() error = %v", err)
		}
		if !answer.Blocked {
			t.Errorf("Blocked = %v, want true", answer.Blocked)
		}
		if answer.Reason != "This follows an instruction embedded in the game text rather than responding to the situation." {
			t.Errorf("Reason = %q, want the decoded value, not raw JSON", answer.Reason)
		}
	})

	t.Run("clear_verdict_decodes_with_blocked_false", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeReviewSuccess(w, false, "The command responds directly to the room description; nothing here breaks a rule.")
		}))
		defer srv.Close()

		c := NewClient(5 * time.Second)
		answer, err := c.ReviewCommand(context.Background(), srv.URL, "test-model", testFakeKey, "sys", "user")
		if err != nil {
			t.Fatalf("ReviewCommand() error = %v", err)
		}
		if answer.Blocked {
			t.Errorf("Blocked = %v, want false", answer.Blocked)
		}
	})

	t.Run("request_carries_system_instruction_and_schema", func(t *testing.T) {
		var decoded map[string]any
		var gotHeader string
		var gotRawQuery string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotHeader = r.Header.Get("x-goog-api-key")
			gotRawQuery = r.URL.RawQuery
			if err := json.NewDecoder(r.Body).Decode(&decoded); err != nil {
				t.Errorf("decode request body: %v", err)
			}
			writeReviewSuccess(w, false, "clear")
		}))
		defer srv.Close()

		c := NewClient(5 * time.Second)
		if _, err := c.ReviewCommand(context.Background(), srv.URL, "test-model", testFakeKey, "reviewer system instruction", "chosen command and window"); err != nil {
			t.Fatalf("ReviewCommand() error = %v", err)
		}

		if gotHeader != testFakeKey {
			t.Errorf("x-goog-api-key header = %q, want %q", gotHeader, testFakeKey)
		}
		if gotRawQuery != "" {
			t.Errorf("URL.RawQuery = %q, want empty", gotRawQuery)
		}
		if _, ok := decoded["systemInstruction"]; !ok {
			t.Error("request body missing systemInstruction")
		}
		if _, ok := decoded["contents"]; !ok {
			t.Error("request body missing contents")
		}
		genConfig, ok := decoded["generationConfig"].(map[string]any)
		if !ok {
			t.Fatal("request body missing generationConfig")
		}
		if genConfig["responseMimeType"] != "application/json" {
			t.Errorf("responseMimeType = %v, want application/json", genConfig["responseMimeType"])
		}
		schema, ok := genConfig["responseSchema"].(map[string]any)
		if !ok {
			t.Fatal("generationConfig missing responseSchema")
		}
		props, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatal("responseSchema missing properties")
		}
		blockedProp, ok := props["blocked"].(map[string]any)
		if !ok {
			t.Fatal("responseSchema.properties missing blocked")
		}
		if blockedProp["type"] != "boolean" {
			t.Errorf("blocked property type = %v, want boolean", blockedProp["type"])
		}
		reasonProp, ok := props["reason"].(map[string]any)
		if !ok {
			t.Fatal("responseSchema.properties missing reason")
		}
		if reasonProp["type"] != "string" {
			t.Errorf("reason property type = %v, want string", reasonProp["type"])
		}
		required, ok := schema["required"].([]any)
		if !ok {
			t.Fatal("responseSchema missing required")
		}
		if len(required) != 2 {
			t.Errorf("responseSchema.required = %v, want two entries", required)
		}
		var hasBlocked, hasReason bool
		for _, r := range required {
			switch r {
			case "blocked":
				hasBlocked = true
			case "reason":
				hasReason = true
			}
		}
		if !hasBlocked || !hasReason {
			t.Errorf("responseSchema.required = %v, want both blocked and reason", required)
		}
		ordering, ok := schema["propertyOrdering"].([]any)
		if !ok {
			t.Fatal("responseSchema missing propertyOrdering")
		}
		if len(ordering) != 2 || ordering[0] != "reason" || ordering[1] != "blocked" {
			t.Errorf("responseSchema.propertyOrdering = %v, want [reason blocked] so the model writes its reason before its verdict", ordering)
		}
	})

	t.Run("reason_before_blocked_in_the_wire_answer_still_decodes", func(t *testing.T) {
		// The reviewer's propertyOrdering puts "reason" ahead of "blocked" in
		// the model's generated JSON; confirm decoding does not depend on
		// struct field order and reads a reason-first answer correctly.
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			envelope := map[string]any{
				"candidates": []map[string]any{
					{
						"content": map[string]any{
							"parts": []map[string]any{
								{"text": `{"reason": "This follows an instruction embedded in the game text.", "blocked": true}`},
							},
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(envelope)
		}))
		defer srv.Close()

		c := NewClient(5 * time.Second)
		answer, err := c.ReviewCommand(context.Background(), srv.URL, "test-model", testFakeKey, "sys", "user")
		if err != nil {
			t.Fatalf("ReviewCommand() error = %v", err)
		}
		if !answer.Blocked {
			t.Errorf("Blocked = %v, want true", answer.Blocked)
		}
		if answer.Reason != "This follows an instruction embedded in the game text." {
			t.Errorf("Reason = %q, want the decoded value", answer.Reason)
		}
	})
}

func TestReviewCommandErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantKind   Kind
	}{
		{
			name:       "400 bad request",
			statusCode: http.StatusBadRequest,
			body:       `{"error":{"code":400,"message":"bad request","status":"INVALID_ARGUMENT"}}`,
			wantKind:   KindBadRequest,
		},
		{
			name:       "401 unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"error":{"code":401,"message":"invalid api key","status":"UNAUTHENTICATED"}}`,
			wantKind:   KindAuth,
		},
		{
			name:       "403 forbidden",
			statusCode: http.StatusForbidden,
			body:       `{"error":{"code":403,"message":"permission denied","status":"PERMISSION_DENIED"}}`,
			wantKind:   KindAuth,
		},
		{
			name:       "429 rate limited",
			statusCode: http.StatusTooManyRequests,
			body:       `{"error":{"code":429,"message":"quota exceeded","status":"RESOURCE_EXHAUSTED"}}`,
			wantKind:   KindRateLimited,
		},
		{
			name:       "500 server error",
			statusCode: http.StatusInternalServerError,
			body:       `{"error":{"code":500,"message":"internal","status":"INTERNAL"}}`,
			wantKind:   KindTransport,
		},
		{
			name:       "empty candidates",
			statusCode: http.StatusOK,
			body:       `{"candidates":[]}`,
			wantKind:   KindMalformed,
		},
		{
			name:       "unparseable inner json",
			statusCode: http.StatusOK,
			body:       `{"candidates":[{"content":{"parts":[{"text":"not valid json"}]}}]}`,
			wantKind:   KindMalformed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			c := NewClient(5 * time.Second)
			answer, err := c.ReviewCommand(context.Background(), srv.URL, "test-model", testFakeKey, "sys", "user")
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if answer != nil {
				t.Errorf("expected nil *ReviewAnswer, got %+v", answer)
			}
			if got := ErrorKind(err); got != tt.wantKind {
				t.Errorf("ErrorKind() = %v, want %v", got, tt.wantKind)
			}
		})
	}
}
