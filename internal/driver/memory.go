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
	"regexp"
	"strings"
	"unicode/utf8"

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

	// maxCoachingBullets is D-08's standing-suggestion list ceiling (plan
	// 05-06): at most 8 coaching lines are ever in effect at once, enforced
	// in Go on the way into either prompt regardless of what the chat
	// answer's own schema asked for, exactly like maxQuestBullets and
	// maxSessionMemoryBullets above.
	maxCoachingBullets = 8
)

// markerNameRe matches the name of any of OUR four delimiting markers, in
// any letter case, wherever it appears in untrusted text.
var markerNameRe = regexp.MustCompile(`(?i)(GAME|QUEST|SESSION|MODEL)_(TEXT|MEMORY|REASONING)`)

// angleBracketReplacer turns every character a model could read as the
// opening or closing angle bracket of a marker -- ASCII '<' and '>' and
// their common Unicode look-alikes -- into a square bracket, which is not
// part of our marker syntax.
var angleBracketReplacer = strings.NewReplacer(
	"<", "[", ">", "]",
	"＜", "[", "＞", "]", // fullwidth
	"﹤", "[", "﹥", "]", // small form
	"‹", "[", "›", "]", // single guillemets
	"〈", "[", "〉", "]", // CJK angle brackets
	"⟨", "[", "⟩", "]", // mathematical angle brackets
	"〈", "[", "〉", "]", // pointing angle brackets
)

// neutraliseUntrusted is the ONE function every untrusted-data wrapper in
// this package passes its content through (code review CR-02 of Phase 4):
// wrapWindow, wrapQuestMemory, wrapSessionMemory and wrapModelReasoning, plus
// goalBlock and the write-side truncateBullets. D-13's whole defence rests on
// the delimiting markers, and until this existed the content between them
// could simply contain "</SESSION_MEMORY>" -- closing its own block and
// leaving whatever followed sitting in the trusted system instruction of the
// player AND the reviewer at once, for as long as the bullet lived.
//
// After this function nothing in s can read as one of our markers: no angle
// bracket of any kind survives (they become square brackets) and no marker
// name survives intact (its underscore becomes a hyphen), so
// "</SESSION_MEMORY>" reaches the model as "[/SESSION-MEMORY]". Line breaks
// are left alone here because the game-text window and a model's reasoning
// legitimately have them; single-line content uses neutraliseLine.
func neutraliseUntrusted(s string) string {
	s = angleBracketReplacer.Replace(s)
	return markerNameRe.ReplaceAllString(s, "$1-$2")
}

// neutraliseLine is neutraliseUntrusted for content that must be exactly one
// line -- a memory bullet, the session goal: every control character
// (CR and LF included) becomes a space and runs of white space collapse to
// one, so a bullet can never start a new line that looks like a heading, a
// new bullet, or a paragraph of the surrounding instruction.
func neutraliseLine(s string) string {
	s = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, s)
	return neutraliseUntrusted(strings.Join(strings.Fields(s), " "))
}

// cutBytes cuts s to at most max bytes without splitting a multi-byte
// character (code review IN-03 of Phase 4: a split rune was stored and
// shown as U+FFFD).
func cutBytes(s string, max int) string {
	if len(s) <= max {
		return s
	}
	for max > 0 && !utf8.RuneStart(s[max]) {
		max--
	}
	return s[:max]
}

// clampBullets enforces D-12's ceilings on a bullet list before it reaches
// either prompt: at most maxCount entries, each neutralised to one
// marker-free line (neutraliseLine -- applied here, at prompt-assembly time,
// so a poisoned bullet stored before code review CR-02's fix is covered too)
// and only then truncated to maxBulletChars bytes (matching maxCommandBytes'
// byte-length convention elsewhere in this package, not a rune count). A nil
// or empty input returns nil, so an unset memory layer contributes no block
// at all.
func clampBullets(bullets []string, maxCount int) []string {
	if len(bullets) == 0 {
		return nil
	}
	out := make([]string, 0, maxCount)
	for _, bullet := range bullets {
		if len(out) >= maxCount {
			break
		}
		out = append(out, cutBytes(neutraliseLine(bullet), maxBulletChars))
	}
	return out
}

// truncateBullets applies D-12's ceilings to a bullet list the model
// proposed on its way out, before anything is stored or shown (T-4-10): at
// most maxItems entries, each neutralised to one marker-free line
// (neutraliseLine, code review CR-02 of Phase 4 -- so neither the store nor
// the panel ever holds a forged marker) and then cut to maxLen bytes, with
// empty and whitespace-only entries dropped entirely. Always returns a
// non-nil slice, even when every entry is dropped or in itself was empty,
// so "the model proposed replacing memory with nothing" stays expressible
// and distinguishable from persistMemory's own "the model proposed no
// change at all" case (a nil answer field), which is checked before this
// function is ever called.
func truncateBullets(in []string, maxItems, maxLen int) []string {
	out := make([]string, 0, maxItems)
	for _, bullet := range in {
		trimmed := neutraliseLine(bullet)
		if trimmed == "" {
			continue
		}
		if len(out) >= maxItems {
			break
		}
		out = append(out, cutBytes(trimmed, maxLen))
	}
	return out
}

// bulletBytes sums the byte length of every bullet in bullets, for the
// memory-update log line (plan 04-08) — the line carries this total, never
// any bullet's own text.
func bulletBytes(bullets []string) int {
	total := 0
	for _, b := range bullets {
		total += len(b)
	}
	return total
}

// renderBullets renders bullets as a plain "- " prefixed list, one per
// line -- the shape wrapQuestMemory and wrapSessionMemory both share. Every
// bullet goes through neutraliseLine here, whatever the caller did or did
// not do first, so neither wrapper can be handed content that closes its own
// block (code review CR-02 of Phase 4).
func renderBullets(bullets []string) string {
	var b strings.Builder
	for i, bullet := range bullets {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("- ")
		b.WriteString(neutraliseLine(bullet))
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

// wrapCoaching encloses bullets between <COACHING> and </COACHING> markers
// (D-08, plan 05-06), a direct sibling of wrapQuestMemory/wrapSessionMemory
// reusing the identical renderBullets/neutraliseLine pipeline -- no new
// clamp, render or neutralise function. Unlike Quest and Session Memory,
// coaching is not the model's own past output; it is the owner's own live
// guidance relayed by AI-chatter, so callers give it its own introducing
// sentence rather than reusing "bullets you wrote yourself" (D-28's Claude's
// Discretion). Used by AI-chatter's own prompt (chat.go's
// buildChatSystemInstruction) today; plan 05-06-02 reuses this same helper,
// unchanged, for AI-player's own prompt (driver.go's
// buildSystemInstruction/buildReviewSystemInstruction), so a withdraw can
// quote a stored line verbatim from the identical rendering both models
// see. An empty list returns the empty string: no block, no markers,
// nothing to wrap.
func wrapCoaching(bullets []string) string {
	if len(bullets) == 0 {
		return ""
	}
	return "<COACHING>\n" + renderBullets(bullets) + "\n</COACHING>"
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
// owner's own instruction. It still goes through neutraliseLine (code review
// CR-02 of Phase 4): the goal is a one-line box, so a pasted line break or a
// literal marker in it is never the owner's intent, and an unwrapped
// "<SESSION_MEMORY>" here would forge the opening of a block in the trusted
// tier of both prompts.
func goalBlock(goal string) string {
	trimmed := neutraliseLine(goal)
	if trimmed == "" {
		return ""
	}
	return "\n\nSession goal (the owner's own instruction for this session; when this is blank you play from the approach guidance above as your standing direction):\n" + trimmed
}
