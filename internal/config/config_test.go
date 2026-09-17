package config

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/amaranth494/MudPuppy/internal/crypto"
)

// Fake credential values only — never a real Gemini key (T-3-16).
const testFakeKey = "test-key-not-a-real-credential"
const testFakeKey2 = "test-key-not-a-real-credential-2"

// testFakeVaultKey is a well-formed vault key that is obviously not a real
// one: the standard base64 of thirty-two 0x01 bytes.
func testFakeVaultKey() string {
	return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x01}, 32))
}

// TestLoadRequiresEncryptionKeyOutsideDevelopment proves D-25 (DR-3.1-04):
// outside local development the server refuses to start when the
// credential-vault key is absent, instead of quietly starting with a
// freshly generated key that makes every stored credential unreadable
// after the next restart. The owner's own machine (RAILWAY_ENVIRONMENT
// unset) is unaffected.
func TestLoadRequiresEncryptionKeyOutsideDevelopment(t *testing.T) {
	t.Run("staging without the key fails", func(t *testing.T) {
		t.Setenv("SESSION_SECRET", "test-secret")
		t.Setenv("RAILWAY_ENVIRONMENT", "staging")
		t.Setenv("ENCRYPTION_KEY_V1", "")

		cfg, err := Load()
		if err == nil {
			t.Fatal("expected Load() to return an error when ENCRYPTION_KEY_V1 is unset outside local development")
		}
		if cfg != nil {
			t.Errorf("expected nil config on error, got %+v", cfg)
		}
		if !strings.Contains(err.Error(), "ENCRYPTION_KEY_V1") {
			t.Errorf("error %q does not mention ENCRYPTION_KEY_V1", err.Error())
		}
		if strings.Contains(err.Error(), testFakeKey) {
			t.Error("error text must never contain key material")
		}
	})

	// Code review WR-07 of Phase 4 changed this sub-test's key. It used to
	// set ENCRYPTION_KEY_V1 to testFakeKey ("test-key-not-a-real-...") and
	// expect success -- a value the key store cannot use, so the test was
	// pinning the hole: staging started, on a random key. It now uses a
	// well-formed fake key; the unusable value moved to the sub-tests below.
	t.Run("staging with a usable key set succeeds", func(t *testing.T) {
		t.Setenv("SESSION_SECRET", "test-secret")
		t.Setenv("RAILWAY_ENVIRONMENT", "staging")
		t.Setenv("ENCRYPTION_KEY_V1", testFakeVaultKey())
		t.Setenv("ENCRYPTION_KEY_V2", "")
		t.Setenv("ENCRYPTION_KEY_V3", "")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() returned error: %v", err)
		}
		if cfg == nil {
			t.Fatal("Load() returned nil *Config")
		}
	})

	// The three ways a key can be present and still unusable (WR-07), plus
	// an unusable rotation key beside a good V1.
	valid := testFakeVaultKey()
	unusable := []struct {
		name   string
		v1, v2 string
		names  string
	}{
		{"staging with a key that is not base64 fails", testFakeKey, "", "ENCRYPTION_KEY_V1"},
		{"staging with a key of the wrong length fails", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x01}, 16)), "", "ENCRYPTION_KEY_V1"},
		{"staging with a key pasted with a trailing space fails", valid + " ", "", "ENCRYPTION_KEY_V1"},
		{"staging with a key pasted in quotes fails", `"` + valid + `"`, "", "ENCRYPTION_KEY_V1"},
		{"staging with a URL-safe base64 key fails", base64.URLEncoding.EncodeToString(bytes.Repeat([]byte{0xfb}, 32)), "", "ENCRYPTION_KEY_V1"},
		{"staging with an unusable rotation key fails", valid, testFakeKey2, "ENCRYPTION_KEY_V2"},
	}
	for _, tc := range unusable {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SESSION_SECRET", "test-secret")
			t.Setenv("RAILWAY_ENVIRONMENT", "staging")
			t.Setenv("ENCRYPTION_KEY_V1", tc.v1)
			t.Setenv("ENCRYPTION_KEY_V2", tc.v2)
			t.Setenv("ENCRYPTION_KEY_V3", "")

			cfg, err := Load()
			if err == nil {
				t.Fatal("expected Load() to refuse a key the key store cannot use: the server would start on a random key")
			}
			if cfg != nil {
				t.Errorf("expected nil config on error, got a config")
			}
			if !strings.Contains(err.Error(), tc.names) {
				t.Errorf("error %q does not name %s", err.Error(), tc.names)
			}
			for _, secret := range []string{tc.v1, tc.v2, strings.TrimSpace(tc.v1), strings.Trim(tc.v1, `"`)} {
				if secret != "" && strings.Contains(err.Error(), secret) {
					t.Error("error text must never contain key material")
				}
			}
		})
	}

	// The gate and the key store share one parser, so they cannot disagree.
	// Go's base64 decoder ignores line breaks, which means a key pasted with
	// a trailing newline IS usable by the key store (the review's example of
	// an unusable key was wrong on this one detail) -- and so the gate must
	// accept it rather than refuse a server that would have started fine.
	t.Run("staging with a trailing newline is accepted because the key store accepts it", func(t *testing.T) {
		t.Setenv("SESSION_SECRET", "test-secret")
		t.Setenv("RAILWAY_ENVIRONMENT", "staging")
		t.Setenv("ENCRYPTION_KEY_V1", valid+"\n")
		t.Setenv("ENCRYPTION_KEY_V2", "")
		t.Setenv("ENCRYPTION_KEY_V3", "")

		if _, err := crypto.ParseKey(valid + "\n"); err != nil {
			t.Fatalf("test premise: the key store's parser should accept a trailing newline, got %v", err)
		}
		if _, err := Load(); err != nil {
			t.Fatalf("Load() refused a key the key store can use: %v", err)
		}
	})

	t.Run("local development with an unusable key still succeeds", func(t *testing.T) {
		t.Setenv("SESSION_SECRET", "test-secret")
		t.Setenv("RAILWAY_ENVIRONMENT", "")
		t.Setenv("ENCRYPTION_KEY_V1", testFakeKey)

		if _, err := Load(); err != nil {
			t.Fatalf("Load() returned error on the owner's own machine: %v", err)
		}
	})

	t.Run("local development without the key succeeds", func(t *testing.T) {
		t.Setenv("SESSION_SECRET", "test-secret")
		t.Setenv("RAILWAY_ENVIRONMENT", "")
		t.Setenv("ENCRYPTION_KEY_V1", "")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() returned error on the owner's own machine: %v", err)
		}
		if cfg == nil {
			t.Fatal("Load() returned nil *Config")
		}
	})
}

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
