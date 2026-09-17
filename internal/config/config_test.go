package config

import (
	"bytes"
	"encoding/base64"
	"log"
	"os"
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

// TestLoadRequiresEncryptionKeyEverywhereUnlessLocalDev proves D-26
// (DR-4-04): the credential-vault key is required on every host, not only
// on Railway, instead of letting internal/crypto.DefaultKeyStore quietly
// generate a fresh random key on every process start on a non-Railway
// host -- the exact failure that already hit staging once (DR-3.1-04,
// when RAILWAY_ENVIRONMENT was the only signal checked). The only way past
// the requirement is the explicitly named MUDPUPPY_LOCAL_DEV flag; a
// present-but-malformed key is always a startup failure, flag on or off,
// because the opt-out skips the requirement, never the validation. Every
// sub-test asserts on the returned error, never on logged text.
func TestLoadRequiresEncryptionKeyEverywhereUnlessLocalDev(t *testing.T) {
	valid := testFakeVaultKey()

	t.Run("no key, no flag, no RAILWAY_ENVIRONMENT fails", func(t *testing.T) {
		t.Setenv("SESSION_SECRET", "test-secret")
		t.Setenv("MUDPUPPY_LOCAL_DEV", "")
		t.Setenv("RAILWAY_ENVIRONMENT", "")
		t.Setenv("ENCRYPTION_KEY_V1", "")

		cfg, err := Load()
		if err == nil {
			t.Fatal("expected Load() to return an error when ENCRYPTION_KEY_V1 is unset and MUDPUPPY_LOCAL_DEV is not on")
		}
		if cfg != nil {
			t.Errorf("expected nil config on error, got %+v", cfg)
		}
		if !strings.Contains(err.Error(), "ENCRYPTION_KEY_V1") {
			t.Errorf("error %q does not mention ENCRYPTION_KEY_V1", err.Error())
		}
	})

	t.Run("no key, RAILWAY_ENVIRONMENT set still fails (the old escape route is gone)", func(t *testing.T) {
		t.Setenv("SESSION_SECRET", "test-secret")
		t.Setenv("MUDPUPPY_LOCAL_DEV", "")
		t.Setenv("RAILWAY_ENVIRONMENT", "staging")
		t.Setenv("ENCRYPTION_KEY_V1", "")

		if _, err := Load(); err == nil {
			t.Fatal("expected Load() to fail even with RAILWAY_ENVIRONMENT set: it is no longer part of this decision")
		}
	})

	for _, spelling := range []string{"true", "TRUE", "1"} {
		t.Run("no key, MUDPUPPY_LOCAL_DEV="+spelling+" succeeds", func(t *testing.T) {
			t.Setenv("SESSION_SECRET", "test-secret")
			t.Setenv("MUDPUPPY_LOCAL_DEV", spelling)
			t.Setenv("RAILWAY_ENVIRONMENT", "")
			t.Setenv("ENCRYPTION_KEY_V1", "")

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() returned error with MUDPUPPY_LOCAL_DEV=%s: %v", spelling, err)
			}
			if cfg == nil {
				t.Fatal("Load() returned nil *Config")
			}
		})
	}

	for _, spelling := range []string{"no", "0", ""} {
		name := spelling
		if name == "" {
			name = "empty"
		}
		t.Run("no key, MUDPUPPY_LOCAL_DEV="+name+" fails", func(t *testing.T) {
			t.Setenv("SESSION_SECRET", "test-secret")
			t.Setenv("MUDPUPPY_LOCAL_DEV", spelling)
			t.Setenv("RAILWAY_ENVIRONMENT", "")
			t.Setenv("ENCRYPTION_KEY_V1", "")

			if _, err := Load(); err == nil {
				t.Fatalf("expected Load() to fail with MUDPUPPY_LOCAL_DEV=%q: anything other than 1/true is off", spelling)
			}
		})
	}

	t.Run("a valid key present with the flag off succeeds", func(t *testing.T) {
		t.Setenv("SESSION_SECRET", "test-secret")
		t.Setenv("MUDPUPPY_LOCAL_DEV", "")
		t.Setenv("RAILWAY_ENVIRONMENT", "")
		t.Setenv("ENCRYPTION_KEY_V1", valid)
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

	// The opt-out skips the requirement, never the validation: a malformed
	// key present with the flag on is still a startup failure.
	unusable := []struct {
		name       string
		v1, v2     string
		namesError string
	}{
		{"a key that is not base64", testFakeKey, "", "ENCRYPTION_KEY_V1"},
		{"a key of the wrong length", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x01}, 16)), "", "ENCRYPTION_KEY_V1"},
		{"a key pasted with a trailing space", valid + " ", "", "ENCRYPTION_KEY_V1"},
		{"a key pasted in quotes", `"` + valid + `"`, "", "ENCRYPTION_KEY_V1"},
		{"a URL-safe base64 key", base64.URLEncoding.EncodeToString(bytes.Repeat([]byte{0xfb}, 32)), "", "ENCRYPTION_KEY_V1"},
		{"an unusable rotation key beside a good V1", valid, testFakeKey2, "ENCRYPTION_KEY_V2"},
	}
	for _, tc := range unusable {
		t.Run("a malformed key ("+tc.name+") present with the flag on still fails", func(t *testing.T) {
			t.Setenv("SESSION_SECRET", "test-secret")
			t.Setenv("MUDPUPPY_LOCAL_DEV", "true")
			t.Setenv("RAILWAY_ENVIRONMENT", "")
			t.Setenv("ENCRYPTION_KEY_V1", tc.v1)
			t.Setenv("ENCRYPTION_KEY_V2", tc.v2)
			t.Setenv("ENCRYPTION_KEY_V3", "")

			cfg, err := Load()
			if err == nil {
				t.Fatal("expected Load() to refuse a key the key store cannot use even with the local-development flag on: the opt-out skips the requirement, never the validation")
			}
			if cfg != nil {
				t.Errorf("expected nil config on error, got a config")
			}
			if !strings.Contains(err.Error(), tc.namesError) {
				t.Errorf("error %q does not name %s", err.Error(), tc.namesError)
			}
			for _, secret := range []string{tc.v1, tc.v2, strings.TrimSpace(tc.v1), strings.Trim(tc.v1, `"`)} {
				if secret != "" && strings.Contains(err.Error(), secret) {
					t.Error("error text must never contain key material")
				}
			}
		})
	}

	// The gate and the key store share one parser, so they cannot disagree.
	// Go's base64 decoder ignores line breaks, which means a key pasted
	// with a trailing newline IS usable by the key store, so the gate must
	// accept it rather than refuse a server that would have started fine.
	t.Run("a trailing newline is accepted because the key store accepts it", func(t *testing.T) {
		t.Setenv("SESSION_SECRET", "test-secret")
		t.Setenv("MUDPUPPY_LOCAL_DEV", "")
		t.Setenv("RAILWAY_ENVIRONMENT", "")
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
}

// TestLocalDevOptOutIsAnnounced proves D-26's second half: turning on the
// local-development escape hatch is a deliberate act that is announced --
// the server logs that it is running without a vault key because
// MUDPUPPY_LOCAL_DEV is set, and never prints the key or any other
// environment value. No sub-test in this file ever asserts on, prints, or
// constructs a real key value; testFakeVaultKey() is a well-formed but
// obviously fake key (32 bytes of 0x01), and testFakeKey/testFakeKey2 are
// plain placeholder strings.
func TestLocalDevOptOutIsAnnounced(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	origFlags := log.Flags()
	log.SetFlags(0)
	defer log.SetFlags(origFlags)

	t.Setenv("SESSION_SECRET", "test-secret")
	t.Setenv("MUDPUPPY_LOCAL_DEV", "true")
	t.Setenv("RAILWAY_ENVIRONMENT", "")
	t.Setenv("ENCRYPTION_KEY_V1", "")
	t.Setenv("ENCRYPTION_KEY_V2", "")
	t.Setenv("ENCRYPTION_KEY_V3", "")

	if _, err := Load(); err != nil {
		t.Fatalf("Load() returned error with the local-development flag on: %v", err)
	}

	out := buf.String()
	if out == "" {
		t.Fatal("expected an announcement line to be logged, got none")
	}
	if !strings.Contains(out, "MUDPUPPY_LOCAL_DEV") {
		t.Errorf("expected the announcement to name MUDPUPPY_LOCAL_DEV, got: %q", out)
	}
}

func TestLoadAIRegistry(t *testing.T) {
	t.Setenv("SESSION_SECRET", "test-secret")
	t.Setenv("MUDPUPPY_LOCAL_DEV", "true") // this test is not about the vault key
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
	t.Setenv("MUDPUPPY_LOCAL_DEV", "true") // this test is not about the vault key
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
			t.Setenv("MUDPUPPY_LOCAL_DEV", "true") // this test is not about the vault key
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
	t.Setenv("MUDPUPPY_LOCAL_DEV", "true") // this test is not about the vault key
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
