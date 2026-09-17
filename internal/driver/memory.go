// memory.go assembles the per-decision prompt context D-13 introduces:
// everything both prompts need beyond the game-text window -- the
// profile's standing text, the owner's own session goal, the active
// Quest's bullets, and the current Session Memory -- and the wrapping
// helpers that delimit the model-written blocks (Quest Memory, Session
// Memory) as untrusted data, mirroring driver.go's own wrapWindow exactly.
// The goal is owner-written, not model-written, so goalBlock never wraps
// it in the untrusted-data markers; it sits with the profile's standing
// text as the owner's own instruction (04-CONTEXT.md's Claude's
// Discretion).
package driver

import (
	"strings"

	"github.com/amaranth494/MudPuppy/internal/store"
)

// promptContext is everything a single decision's prompts need beyond the
// window (D-13): the profile, the session goal, the active Quest's
// bullets, and the current Session Memory. Both buildSystemInstruction and
// buildReviewSystemInstruction take one of these in place of the bare
// *store.Profile they took before this plan -- the reviewer sees the
// identical goal/Quest/Session Memory blocks the player model sees, not
// just the game text and the player's chosen command (must_haves truth 1).
type promptContext struct {
	Profile       *store.Profile
	Goal          string
	QuestBullets  []string
	SessionMemory []string
}

// maxQuestBullets, maxSessionMemoryBullets and maxBulletChars are D-12's
// size ceilings: at most 20 Quest Memory bullets and 30 Session Memory
// bullets, each truncated to 200 characters. These are enforced here, in
// Go, on the way into the prompt, regardless of what any answer schema
// asks the model for later (04-RESEARCH's ASVS V5 row and Assumption A7:
// Gemini's structured output has no confirmed maxItems/maxLength
// enforcement of its own). Truncation is silent and defensive, not an
// error -- oversized input is clipped, never rejected.
const (
	maxQuestBullets         = 20
	maxSessionMemoryBullets = 30
	maxBulletChars          = 200
)

// clampBullets enforces D-12's ceilings on a bullet list before it reaches
// either prompt: at most maxCount entries, each truncated to
// maxBulletChars bytes (matching maxCommandBytes' byte-length convention
// elsewhere in this package, not a rune count). A nil or empty input
// returns nil, so an unset memory layer contributes no block at all.
func clampBullets(bullets []string, maxCount int) []string {
	if len(bullets) == 0 {
		return nil
	}
	out := make([]string, 0, maxCount)
	for _, bullet := range bullets {
		if len(out) >= maxCount {
			break
		}
		if len(bullet) > maxBulletChars {
			bullet = bullet[:maxBulletChars]
		}
		out = append(out, bullet)
	}
	return out
}

// renderBullets renders bullets as a plain "- " prefixed list, one per
// line -- the shape wrapQuestMemory and wrapSessionMemory both share.
func renderBullets(bullets []string) string {
	var b strings.Builder
	for i, bullet := range bullets {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("- ")
		b.WriteString(bullet)
	}
	return b.String()
}

// wrapQuestMemory encloses bullets between <QUEST_MEMORY> and
// </QUEST_MEMORY> markers (D-13), mirroring wrapWindow's exact shape, so
// Quest Memory is delimited as untrusted data exactly as the game text is
// -- it was written by the model itself, from game text, and
// untrustedDataParagraph names this marker alongside <GAME_TEXT> and
// <SESSION_MEMORY>. An empty list returns an empty string: no block, no
// markers, nothing to wrap.
func wrapQuestMemory(bullets []string) string {
	if len(bullets) == 0 {
		return ""
	}
	return "<QUEST_MEMORY>\n" + renderBullets(bullets) + "\n</QUEST_MEMORY>"
}

// wrapSessionMemory is wrapQuestMemory's exact structural sibling for
// Session Memory's own markers (D-13).
func wrapSessionMemory(bullets []string) string {
	if len(bullets) == 0 {
		return ""
	}
	return "<SESSION_MEMORY>\n" + renderBullets(bullets) + "\n</SESSION_MEMORY>"
}

// goalBlock returns the session goal's labelled section, or an empty
// string when goal is blank (D-02): a blank goal is not an error state,
// it simply means this block is omitted entirely and the model continues
// playing from the profile's approach guidance as its standing direction
// -- proven by must_haves truth 5's blank case, where the prompt simply
// omits the block rather than saying anything about its absence. The goal
// is the owner's own typed text, never model-written, so unlike Quest
// Memory and Session Memory it is not wrapped in the untrusted-data
// markers; it sits with the profile's standing text, labelled as the
// owner's own instruction.
func goalBlock(goal string) string {
	trimmed := strings.TrimSpace(goal)
	if trimmed == "" {
		return ""
	}
	return "\n\nSession goal (the owner's own instruction for this session; when this is blank you play from the approach guidance above as your standing direction):\n" + trimmed
}
