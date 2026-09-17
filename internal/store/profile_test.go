package store

import (
	"strings"
	"testing"
)

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

// TestResolveAISettings_RateLimitBlankMeansServerDefault proves D-25/
// DR-4-03's resolution rule: a nil RateLimitPerSecond resolves to
// RateLimitSet == false and RateLimitPerSecond == DefaultAIRateLimitPerSecond
// (never "unlimited" — unlike CallCap's own blank-means-no-cap rule), a set
// value resolves to RateLimitSet == true and that value, and the three
// existing settings resolve exactly as TestResolveAISettings above already
// proves they do, unaffected by this fourth field.
func TestResolveAISettings_RateLimitBlankMeansServerDefault(t *testing.T) {
	tests := []struct {
		name               string
		settings           AISettings
		serverDefaultModel string
		wantRateLimitSet   bool
		wantRateLimit      int
	}{
		{
			name:               "nil rate limit resolves to server default, not unlimited",
			settings:           AISettings{},
			serverDefaultModel: "gemini-server-default",
			wantRateLimitSet:   false,
			wantRateLimit:      DefaultAIRateLimitPerSecond,
		},
		{
			name:               "explicit rate limit 7 sets RateLimitSet and RateLimitPerSecond",
			settings:           AISettings{RateLimitPerSecond: intPtr(7)},
			serverDefaultModel: "gemini-server-default",
			wantRateLimitSet:   true,
			wantRateLimit:      7,
		},
		{
			name: "rate limit resolves independently of the other three settings",
			settings: AISettings{
				ModelName:          "gemini-2.5-pro",
				CallCap:            intPtr(25),
				DisengageThreshold: intPtr(1),
				RateLimitPerSecond: intPtr(10),
			},
			serverDefaultModel: "gemini-server-default",
			wantRateLimitSet:   true,
			wantRateLimit:      10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveAISettings(tt.settings, tt.serverDefaultModel)
			if got.RateLimitSet != tt.wantRateLimitSet {
				t.Errorf("ResolveAISettings(...).RateLimitSet = %v, want %v", got.RateLimitSet, tt.wantRateLimitSet)
			}
			if got.RateLimitPerSecond != tt.wantRateLimit {
				t.Errorf("ResolveAISettings(...).RateLimitPerSecond = %d, want %d", got.RateLimitPerSecond, tt.wantRateLimit)
			}

			// The three existing settings resolve exactly as
			// TestResolveAISettings already proves, unaffected by this
			// fourth field.
			wantModelName := tt.settings.ModelName
			if wantModelName == "" {
				wantModelName = tt.serverDefaultModel
			}
			if got.ModelName != wantModelName {
				t.Errorf("ResolveAISettings(...).ModelName = %q, want %q", got.ModelName, wantModelName)
			}
			wantCallCapSet := tt.settings.CallCap != nil
			if got.CallCapSet != wantCallCapSet {
				t.Errorf("ResolveAISettings(...).CallCapSet = %v, want %v", got.CallCapSet, wantCallCapSet)
			}
			wantThreshold := DefaultDisengageThreshold
			if tt.settings.DisengageThreshold != nil {
				wantThreshold = *tt.settings.DisengageThreshold
			}
			if got.DisengageThreshold != wantThreshold {
				t.Errorf("ResolveAISettings(...).DisengageThreshold = %d, want %d", got.DisengageThreshold, wantThreshold)
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

// profileColumnsOtherThanTheGoal is every profiles column UpdateProfile's
// whole-row write names. None of them may appear in the goal's statement.
var profileColumnsOtherThanTheGoal = []string{
	"keybindings", "settings", "aliases", "triggers", "variables", "timers",
	"conduct_rules", "approach_guidance", "never_issue_list", "ai_settings",
	"policy_version_accepted", "policy_accepted_at",
}

// TestUpdateSessionGoalTouchesOnlyTheGoal proves code review WR-08 of Phase
// 4 without a database connection, in this package's pinned-SQL style: the
// goal box's statement sets the goal column (and updated_at) and nothing
// else, and is scoped to the owner's own profile row.
func TestUpdateSessionGoalTouchesOnlyTheGoal(t *testing.T) {
	stmt := strings.Join(strings.Fields(sqlWithoutComments(updateSessionGoalSQL)), " ")

	const want = "UPDATE profiles SET session_goal = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3"
	if stmt != want {
		t.Fatalf("goal statement =\n  %s\nwant\n  %s", stmt, want)
	}
	for _, column := range profileColumnsOtherThanTheGoal {
		if strings.Contains(stmt, column) {
			t.Errorf("the goal statement names %q: a goal save must never write another column", column)
		}
	}
}

// TestUpdateProfileLeavesTheGoalAlone is the other direction of WR-08: the
// whole-row write, which saves every column it names from a row read
// moments earlier, must not name session_goal, or a settings/alias/trigger
// save that interleaves with a goal edit writes the old goal back.
func TestUpdateProfileLeavesTheGoalAlone(t *testing.T) {
	stmt := sqlWithoutComments(updateProfileSQL)
	if strings.Contains(stmt, "session_goal") {
		t.Fatalf("UpdateProfile's whole-row write names session_goal:\n%s", stmt)
	}
	for _, column := range profileColumnsOtherThanTheGoal[:10] {
		if !strings.Contains(stmt, column+" = $") {
			t.Errorf("UpdateProfile's statement no longer sets %q:\n%s", column, stmt)
		}
	}
	if !strings.Contains(stmt, "WHERE id = $11 AND user_id = $12") {
		t.Errorf("UpdateProfile's statement is not scoped to the owner's row as expected:\n%s", stmt)
	}
}

func TestEngageGateRefusalMessageIsPhase2Contract(t *testing.T) {
	want := "AI Player has not been configured for this connection. Accept the Safety and Abuse policy in AI Player settings before engaging autopilot."
	if EngageGateRefusalMessage != want {
		t.Errorf("EngageGateRefusalMessage = %q, want %q", EngageGateRefusalMessage, want)
	}
}
