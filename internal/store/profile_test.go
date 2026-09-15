package store

import "testing"

func intPtr(i int) *int {
	return &i
}

func strPtr(s string) *string {
	return &s
}

func TestResolveAISettings(t *testing.T) {
	tests := []struct {
		name               string
		settings           AISettings
		serverDefaultModel string
		wantModelName      string
		wantCallCapSet     bool
		wantCallCap        int
		wantThreshold      int
	}{
		{
			name:               "all blank resolves to server default model, no cap, threshold 3",
			settings:           AISettings{},
			serverDefaultModel: "gemini-server-default",
			wantModelName:      "gemini-server-default",
			wantCallCapSet:     false,
			wantCallCap:        0,
			wantThreshold:      DefaultDisengageThreshold,
		},
		{
			name:               "explicit model name passes through unchanged",
			settings:           AISettings{ModelName: "gemini-2.5-pro"},
			serverDefaultModel: "gemini-server-default",
			wantModelName:      "gemini-2.5-pro",
			wantCallCapSet:     false,
			wantCallCap:        0,
			wantThreshold:      DefaultDisengageThreshold,
		},
		{
			name:               "explicit call cap 25 sets CallCapSet and CallCap",
			settings:           AISettings{CallCap: intPtr(25)},
			serverDefaultModel: "gemini-server-default",
			wantModelName:      "gemini-server-default",
			wantCallCapSet:     true,
			wantCallCap:        25,
			wantThreshold:      DefaultDisengageThreshold,
		},
		{
			name:               "explicit disengage threshold 1 passes through unchanged",
			settings:           AISettings{DisengageThreshold: intPtr(1)},
			serverDefaultModel: "gemini-server-default",
			wantModelName:      "gemini-server-default",
			wantCallCapSet:     false,
			wantCallCap:        0,
			wantThreshold:      1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveAISettings(tt.settings, tt.serverDefaultModel)
			if got.ModelName != tt.wantModelName {
				t.Errorf("ResolveAISettings(...).ModelName = %q, want %q", got.ModelName, tt.wantModelName)
			}
			if got.CallCapSet != tt.wantCallCapSet {
				t.Errorf("ResolveAISettings(...).CallCapSet = %v, want %v", got.CallCapSet, tt.wantCallCapSet)
			}
			if got.CallCap != tt.wantCallCap {
				t.Errorf("ResolveAISettings(...).CallCap = %d, want %d", got.CallCap, tt.wantCallCap)
			}
			if got.DisengageThreshold != tt.wantThreshold {
				t.Errorf("ResolveAISettings(...).DisengageThreshold = %d, want %d", got.DisengageThreshold, tt.wantThreshold)
			}
		})
	}
}

func TestEngageGateAllowed(t *testing.T) {
	tests := []struct {
		name                  string
		policyVersionAccepted *string
		policyAcceptedAt      *string
		want                  bool
	}{
		{
			name:                  "both nil is refused",
			policyVersionAccepted: nil,
			policyAcceptedAt:      nil,
			want:                  false,
		},
		{
			name:                  "version set, timestamp nil is refused",
			policyVersionAccepted: strPtr("1.0"),
			policyAcceptedAt:      nil,
			want:                  false,
		},
		{
			name:                  "version nil, timestamp set is refused",
			policyVersionAccepted: nil,
			policyAcceptedAt:      strPtr("2026-09-15T00:00:00Z"),
			want:                  false,
		},
		{
			name:                  "version is empty string with timestamp set is refused",
			policyVersionAccepted: strPtr(""),
			policyAcceptedAt:      strPtr("2026-09-15T00:00:00Z"),
			want:                  false,
		},
		{
			name:                  "both set is allowed",
			policyVersionAccepted: strPtr("1.0"),
			policyAcceptedAt:      strPtr("2026-09-15T00:00:00Z"),
			want:                  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EngageGateAllowed(tt.policyVersionAccepted, tt.policyAcceptedAt)
			if got != tt.want {
				t.Errorf("EngageGateAllowed(%v, %v) = %v, want %v", tt.policyVersionAccepted, tt.policyAcceptedAt, got, tt.want)
			}
		})
	}
}

func TestEngageGateRefusalMessageIsPhase2Contract(t *testing.T) {
	want := "AI Player has not been configured for this connection. Accept the Safety and Abuse policy in AI Player settings before engaging autopilot."
	if EngageGateRefusalMessage != want {
		t.Errorf("EngageGateRefusalMessage = %q, want %q", EngageGateRefusalMessage, want)
	}
}
