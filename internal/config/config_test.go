package config

import "testing"

// Fake credential values only — never a real Gemini key (T-3-16).
const testFakeKey = "test-key-not-a-real-credential"
const testFakeKey2 = "test-key-not-a-real-credential-2"

func TestLoadAIRegistry(t *testing.T) {
	t.Setenv("SESSION_SECRET", "test-secret")
	t.Setenv("AI_MODEL_DEFAULT", "gemini")
	t.Setenv("AI_MODEL_GEMINI_NAME", "test-model-name")
	t.Setenv("AI_MODEL_GEMINI_ENDPOINT", "https://example.invalid/gemini")
	t.Setenv("AI_MODEL_GEMINI_KEY", testFakeKey)
	t.Setenv("AI_MODEL_SECOND_NAME", "test-second-model")
	t.Setenv("AI_MODEL_SECOND_ENDPOINT", "https://example.invalid/second")
	t.Setenv("AI_MODEL_SECOND_KEY", testFakeKey2)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	gemini, ok := cfg.AIModels["GEMINI"]
	if !ok {
		t.Fatal("expected GEMINI entry in registry")
	}
	if gemini.ModelName != "test-model-name" {
		t.Errorf("ModelName = %q, want %q", gemini.ModelName, "test-model-name")
	}
	if gemini.Endpoint != "https://example.invalid/gemini" {
		t.Errorf("Endpoint = %q", gemini.Endpoint)
	}
	if gemini.APIKey != testFakeKey {
		t.Errorf("APIKey = %q", gemini.APIKey)
	}
	if gemini.Provider != "gemini" {
		t.Errorf("Provider = %q, want default %q", gemini.Provider, "gemini")
	}

	second, ok := cfg.AIModels["SECOND"]
	if !ok {
		t.Fatal("expected SECOND entry in registry")
	}
	if second.ModelName != "test-second-model" {
		t.Errorf("ModelName = %q, want %q", second.ModelName, "test-second-model")
	}

	if cfg.AIDefaultModelSlug != "GEMINI" {
		t.Errorf("AIDefaultModelSlug = %q, want GEMINI", cfg.AIDefaultModelSlug)
	}
	if _, ok := cfg.AIModels[cfg.AIDefaultModelSlug]; !ok {
		t.Error("default slug does not point at a registered entry")
	}
}

func TestLoadAIRegistryIgnoresIncompleteEntries(t *testing.T) {
	t.Setenv("SESSION_SECRET", "test-secret")
	t.Setenv("AI_MODEL_INCOMPLETE_NAME", "test-model-name")
	// No endpoint, no key set for INCOMPLETE.

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if _, ok := cfg.AIModels["INCOMPLETE"]; ok {
		t.Error("expected entry missing an endpoint and key to be skipped")
	}
}

func TestAIConfigured(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T)
		want  bool
	}{
		{
			name:  "nothing set",
			setup: func(t *testing.T) {},
			want:  false,
		},
		{
			name: "default slug points at a missing entry",
			setup: func(t *testing.T) {
				t.Setenv("AI_MODEL_DEFAULT", "GHOST")
			},
			want: false,
		},
		{
			name: "an entry with a blank key",
			setup: func(t *testing.T) {
				t.Setenv("AI_MODEL_DEFAULT", "GEMINI")
				t.Setenv("AI_MODEL_GEMINI_NAME", "test-model-name")
				t.Setenv("AI_MODEL_GEMINI_ENDPOINT", "https://example.invalid")
				// No key: Load() skips registering this entry at all, so
				// AIConfigured is false because the default slug is unmatched.
			},
			want: false,
		},
		{
			name: "a complete entry",
			setup: func(t *testing.T) {
				t.Setenv("AI_MODEL_DEFAULT", "GEMINI")
				t.Setenv("AI_MODEL_GEMINI_NAME", "test-model-name")
				t.Setenv("AI_MODEL_GEMINI_ENDPOINT", "https://example.invalid")
				t.Setenv("AI_MODEL_GEMINI_KEY", testFakeKey)
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SESSION_SECRET", "test-secret")
			tt.setup(t)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() returned error: %v", err)
			}
			if cfg == nil {
				t.Fatal("Load() returned nil *Config")
			}
			if got := cfg.AIConfigured(); got != tt.want {
				t.Errorf("AIConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveModelEntry(t *testing.T) {
	t.Setenv("SESSION_SECRET", "test-secret")
	t.Setenv("AI_MODEL_DEFAULT", "GEMINI")
	t.Setenv("AI_MODEL_GEMINI_NAME", "test-model-name")
	t.Setenv("AI_MODEL_GEMINI_ENDPOINT", "https://example.invalid/gemini")
	t.Setenv("AI_MODEL_GEMINI_KEY", testFakeKey)
	t.Setenv("AI_MODEL_ALT_NAME", "test-alt-model")
	t.Setenv("AI_MODEL_ALT_ENDPOINT", "https://example.invalid/alt")
	t.Setenv("AI_MODEL_ALT_KEY", testFakeKey2)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	t.Run("blank requested resolves to the default entry", func(t *testing.T) {
		entry, ok := cfg.ResolveModelEntry("")
		if !ok {
			t.Fatal("expected ok")
		}
		if entry.Slug != "GEMINI" {
			t.Errorf("Slug = %q, want GEMINI", entry.Slug)
		}
	})

	t.Run("requested matches an entry slug case-insensitively", func(t *testing.T) {
		entry, ok := cfg.ResolveModelEntry("alt")
		if !ok {
			t.Fatal("expected ok")
		}
		if entry.Slug != "ALT" {
			t.Errorf("Slug = %q, want ALT", entry.Slug)
		}
	})

	t.Run("requested matches an entry's model name exactly", func(t *testing.T) {
		entry, ok := cfg.ResolveModelEntry("test-alt-model")
		if !ok {
			t.Fatal("expected ok")
		}
		if entry.Slug != "ALT" {
			t.Errorf("Slug = %q, want ALT", entry.Slug)
		}
	})

	t.Run("unknown requested name falls back to the default endpoint and key", func(t *testing.T) {
		entry, ok := cfg.ResolveModelEntry("some-retired-model")
		if !ok {
			t.Fatal("expected ok")
		}
		if entry.ModelName != "some-retired-model" {
			t.Errorf("ModelName = %q, want the requested name", entry.ModelName)
		}
		if entry.Endpoint != "https://example.invalid/gemini" {
			t.Errorf("Endpoint = %q, want the default entry's endpoint", entry.Endpoint)
		}
		if entry.APIKey != testFakeKey {
			t.Errorf("APIKey = %q, want the default entry's key", entry.APIKey)
		}
	})

	t.Run("no usable default returns the zero value and false", func(t *testing.T) {
		empty := &Config{}
		entry, ok := empty.ResolveModelEntry("anything")
		if ok {
			t.Fatal("expected not ok")
		}
		if entry != (AIModelEntry{}) {
			t.Errorf("entry = %+v, want the zero value", entry)
		}
	})
}
