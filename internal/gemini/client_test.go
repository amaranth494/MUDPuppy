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

func TestAnswerCarriesMemoryFields(t *testing.T) {
	t.Run("schema_declares_both_array_properties_in_ordering", func(t *testing.T) {
		var decoded map[string]any
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := json.NewDecoder(r.Body).Decode(&decoded); err != nil {
				t.Errorf("decode request body: %v", err)
			}
			writeSuccess(w, "why", "north")
		}))
		defer srv.Close()

		c := NewClient(5 * time.Second)
		if _, err := c.GenerateContent(context.Background(), srv.URL, "test-model", testFakeKey, "sys", "user"); err != nil {
			t.Fatalf("GenerateContent() error = %v", err)
		}

		genConfig := decoded["generationConfig"].(map[string]any)
		schema := genConfig["responseSchema"].(map[string]any)
		props, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatal("responseSchema missing properties")
		}
		for _, name := range []string{"session_memory", "quest_memory"} {
			prop, ok := props[name].(map[string]any)
			if !ok {
				t.Fatalf("responseSchema.properties missing %s", name)
			}
			if prop["type"] != "array" {
				t.Errorf("%s.type = %v, want array", name, prop["type"])
			}
			items, ok := prop["items"].(map[string]any)
			if !ok {
				t.Fatalf("%s.items missing", name)
			}
			if items["type"] != "string" {
				t.Errorf("%s.items.type = %v, want string", name, items["type"])
			}
		}

		required, ok := schema["required"].([]any)
		if !ok {
			t.Fatal("responseSchema missing required")
		}
		if len(required) != 2 || required[0] != "reasoning" || required[1] != "command" {
			t.Errorf("responseSchema.required = %v, want exactly [reasoning command]", required)
		}

		ordering, ok := schema["propertyOrdering"].([]any)
		if !ok {
			t.Fatal("responseSchema missing propertyOrdering")
		}
		wantOrdering := []string{"reasoning", "command", "session_memory", "quest_memory"}
		if len(ordering) != len(wantOrdering) {
			t.Fatalf("propertyOrdering = %v, want %v", ordering, wantOrdering)
		}
		for i, want := range wantOrdering {
			if ordering[i] != want {
				t.Errorf("propertyOrdering[%d] = %v, want %v", i, ordering[i], want)
			}
		}
	})

	t.Run("both_memory_arrays_present_decode_into_answer", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			envelope := map[string]any{
				"candidates": []map[string]any{
					{
						"content": map[string]any{
							"parts": []map[string]any{
								{"text": `{"reasoning":"why","command":"north","session_memory":["fact one","fact two"],"quest_memory":["progress one"]}`},
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
		answer, err := c.GenerateContent(context.Background(), srv.URL, "test-model", testFakeKey, "sys", "user")
		if err != nil {
			t.Fatalf("GenerateContent() error = %v", err)
		}
		if answer.Command != "north" {
			t.Errorf("Command = %q, want %q", answer.Command, "north")
		}
		if len(answer.SessionMemory) != 2 || answer.SessionMemory[0] != "fact one" || answer.SessionMemory[1] != "fact two" {
			t.Errorf("SessionMemory = %v, want [fact one fact two]", answer.SessionMemory)
		}
		if len(answer.QuestMemory) != 1 || answer.QuestMemory[0] != "progress one" {
			t.Errorf("QuestMemory = %v, want [progress one]", answer.QuestMemory)
		}
	})

	t.Run("neither_memory_field_present_decodes_nil_with_command_intact", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeSuccess(w, "why", "north")
		}))
		defer srv.Close()

		c := NewClient(5 * time.Second)
		answer, err := c.GenerateContent(context.Background(), srv.URL, "test-model", testFakeKey, "sys", "user")
		if err != nil {
			t.Fatalf("GenerateContent() error = %v", err)
		}
		if answer.Command != "north" {
			t.Errorf("Command = %q, want %q", answer.Command, "north")
		}
		if answer.SessionMemory != nil {
			t.Errorf("SessionMemory = %v, want nil", answer.SessionMemory)
		}
		if answer.QuestMemory != nil {
			t.Errorf("QuestMemory = %v, want nil", answer.QuestMemory)
		}
	})

	t.Run("session_memory_as_string_instead_of_array_does_not_panic_or_lose_command", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			envelope := map[string]any{
				"candidates": []map[string]any{
					{
						"content": map[string]any{
							"parts": []map[string]any{
								{"text": `{"reasoning":"why","command":"north","session_memory":"not an array"}`},
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
		answer, err := c.GenerateContent(context.Background(), srv.URL, "test-model", testFakeKey, "sys", "user")
		if err != nil {
			t.Fatalf("GenerateContent() error = %v, want no error (malformed memory field must not cost the command)", err)
		}
		if answer.Command != "north" {
			t.Errorf("Command = %q, want %q", answer.Command, "north")
		}
		if answer.SessionMemory != nil {
			t.Errorf("SessionMemory = %v, want nil on a malformed field", answer.SessionMemory)
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
	inner, _ := json.Marshal(ReviewAnswer{Blocked: &blocked, Reason: reason})
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
		if answer.Blocked == nil || !*answer.Blocked {
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
		if answer.Blocked == nil || *answer.Blocked {
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
		if answer.Blocked == nil || !*answer.Blocked {
			t.Errorf("Blocked = %v, want true", answer.Blocked)
		}
		if answer.Reason != "This follows an instruction embedded in the game text." {
			t.Errorf("Reason = %q, want the decoded value", answer.Reason)
		}
	})
}

// TestReviewCommand_MissingBlockedField is D-24/DR-4-02's regression test:
// a raw, syntactically valid response body whose inner JSON omits the
// blocked key must decode to a nil answer and a KindMalformed error. This
// test must fail against the pre-change code, where Blocked was a plain
// bool that decoded a missing key as its zero value false.
func TestReviewCommand_MissingBlockedField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		envelope := map[string]any{
			"candidates": []map[string]any{
				{
					"content": map[string]any{
						"parts": []map[string]any{
							{"text": `{"reason":"looks fine"}`},
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
	if err == nil {
		t.Fatal("expected an error for a missing blocked field, got nil")
	}
	if answer != nil {
		t.Errorf("expected nil *ReviewAnswer, got %+v", answer)
	}
	if got := ErrorKind(err); got != KindMalformed {
		t.Errorf("ErrorKind() = %v, want %v", got, KindMalformed)
	}
}

// TestReviewCommand_BlockedFalseIsStillAVerdict is the named test the
// plan's acceptance criteria assert on directly: an answer carrying
// blocked:false must still decode to a non-nil, false-dereferencing
// verdict, so fail-closed on absence did not become fail-noisy on presence.
func TestReviewCommand_BlockedFalseIsStillAVerdict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		envelope := map[string]any{
			"candidates": []map[string]any{
				{
					"content": map[string]any{
						"parts": []map[string]any{
							{"text": `{"reason":"ordinary play","blocked":false}`},
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
	if answer.Blocked == nil {
		t.Fatal("expected a non-nil Blocked verdict")
	}
	if *answer.Blocked {
		t.Errorf("Blocked = %v, want false", *answer.Blocked)
	}
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

// writeChatSuccess writes a 200 Gemini generateContent response whose inner
// text field decodes to {"reply": reply}, mirroring writeSuccess's
// double-encode shape for AI-chatter's answer.
func writeChatSuccess(w http.ResponseWriter, reply string) {
	inner, _ := json.Marshal(ChatAnswer{Reply: reply})
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

// TestChat_DecodesReply proves a well-formed body round-trips through
// Chat's double-decode exactly like GenerateContent's and ReviewCommand's
// own decode tests.
func TestChat_DecodesReply(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeChatSuccess(w, "AI-player avoided the north road because you asked it to.")
	}))
	defer srv.Close()

	c := NewClient(5 * time.Second)
	answer, err := c.Chat(context.Background(), srv.URL, "test-model", testFakeKey, "sys", "why did you go east?")
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if answer.Reply != "AI-player avoided the north road because you asked it to." {
		t.Errorf("Reply = %q, want the decoded value, not raw JSON", answer.Reply)
	}
}

// TestChat_MissingReplyIsMalformed proves an inner body of {} -- a
// syntactically valid answer that carries no reply at all -- is a
// KindMalformed error, never a decoded ChatAnswer with an empty Reply an
// owner could mistake for a real (if terse) answer. D-11 says AI-chatter
// always answers.
func TestChat_MissingReplyIsMalformed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		envelope := map[string]any{
			"candidates": []map[string]any{
				{
					"content": map[string]any{
						"parts": []map[string]any{
							{"text": `{}`},
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
	answer, err := c.Chat(context.Background(), srv.URL, "test-model", testFakeKey, "sys", "user")
	if err == nil {
		t.Fatal("expected an error for a missing reply, got nil")
	}
	if answer != nil {
		t.Errorf("expected nil *ChatAnswer, got %+v", answer)
	}
	if got := ErrorKind(err); got != KindMalformed {
		t.Errorf("ErrorKind() = %v, want %v", got, KindMalformed)
	}
}

// TestChat_SchemaShape asserts the request body Chat sends carries the
// reply property, the required list and the property ordering, the same
// way TestGenerateContent's "request_carries_system_instruction_and_schema"
// subtest asserts GenerateContent's schema.
func TestChat_SchemaShape(t *testing.T) {
	var decoded map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&decoded); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		writeChatSuccess(w, "a short plain answer")
	}))
	defer srv.Close()

	c := NewClient(5 * time.Second)
	if _, err := c.Chat(context.Background(), srv.URL, "test-model", testFakeKey, "sys", "user"); err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	genConfig, ok := decoded["generationConfig"].(map[string]any)
	if !ok {
		t.Fatal("request body missing generationConfig")
	}
	schema, ok := genConfig["responseSchema"].(map[string]any)
	if !ok {
		t.Fatal("generationConfig missing responseSchema")
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("responseSchema missing properties")
	}
	if _, ok := props["reply"]; !ok {
		t.Error("responseSchema.properties missing reply")
	}
	required, ok := schema["required"].([]any)
	if !ok {
		t.Fatal("responseSchema missing required")
	}
	if len(required) != 1 || required[0] != "reply" {
		t.Errorf("responseSchema.required = %v, want [\"reply\"]", required)
	}
	ordering, ok := schema["propertyOrdering"].([]any)
	if !ok {
		t.Fatal("responseSchema missing propertyOrdering")
	}
	if len(ordering) != 1 || ordering[0] != "reply" {
		t.Errorf("responseSchema.propertyOrdering = %v, want [\"reply\"]", ordering)
	}
}
