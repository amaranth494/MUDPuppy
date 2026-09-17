// coaching_gate.go is the mechanical provenance gate in front of the
// coaching store (code review CR-02 of Phase 5, D-28).
//
// AI-chatter reads untrusted game text and model-written memory and then
// names lines to push down to AI-player. Until this gate existed the only
// thing tying a pushed line to what the owner typed was a sentence in the
// chat prompt. The rule here is enforced in Go, after the model has answered
// and before anything is stored: a pushed line is accepted only when at
// least half of its content words also appear in the owner's CURRENT chat
// message. A line lifted from a sign, a memory bullet or AI-player's
// reasoning shares no such words with "what are you doing?" and is refused,
// whatever the model was talked into.
//
// The gate is deliberately simple and predictable. It is not a judgement of
// meaning; it is a check that the words came from the owner. The chat prompt
// tells the model to reuse the owner's own wording for exactly this reason.
package driver

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// maxPushesPerMessage is the most coaching lines one owner message may add
// (code review CR-02 of Phase 5). With the standing list capped at
// maxCoachingBullets, it also bounds how much one answer can evict: a push
// never evicts more than it adds, so a single answer cannot wipe the list.
const maxPushesPerMessage = 2

// minContentWordRunes is the shortest word the gate counts. Shorter words
// ("the", "to", "go") carry too little to tie a line to a message.
const minContentWordRunes = 4

// coachingStopWords are words of minContentWordRunes or more that carry no
// content of their own. They are removed from BOTH sides before comparing,
// so a short hostile line cannot pass on a generic word it happens to share
// with the owner's message ("never flee" against "you never listen").
var coachingStopWords = map[string]bool{
	"about": true, "after": true, "again": true, "also": true, "always": true,
	"another": true, "anyone": true, "anything": true, "because": true,
	"been": true, "before": true, "being": true, "could": true, "does": true,
	"doing": true, "done": true, "each": true, "else": true, "ever": true,
	"every": true, "everyone": true, "everything": true, "from": true,
	"have": true, "having": true, "here": true, "into": true, "just": true,
	"like": true, "make": true, "many": true, "more": true, "most": true,
	"much": true, "must": true, "need": true, "never": true, "nothing": true,
	"once": true, "only": true, "onto": true, "other": true, "over": true,
	"please": true, "shall": true, "should": true, "some": true,
	"someone": true, "something": true, "such": true, "sure": true,
	"tell": true, "than": true, "that": true, "their": true, "theirs": true,
	"them": true, "then": true, "there": true, "these": true, "they": true,
	"this": true, "those": true, "told": true, "under": true, "unless": true,
	"until": true, "very": true, "want": true, "were": true, "what": true,
	"when": true, "whenever": true, "where": true, "which": true,
	"while": true, "will": true, "with": true, "would": true, "your": true,
	"yours": true,
}

// contentWords returns the distinct content words of s, in first-seen
// order: lower-cased, split on anything that is not a letter or a digit, at
// least minContentWordRunes long, stop-words removed.
func contentWords(s string) []string {
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	seen := make(map[string]bool, len(fields))
	out := make([]string, 0, len(fields))
	for _, w := range fields {
		if utf8.RuneCountInString(w) < minContentWordRunes || coachingStopWords[w] || seen[w] {
			continue
		}
		seen[w] = true
		out = append(out, w)
	}
	return out
}

// pushedLineComesFromOwner is the provenance rule: at least half of line's
// content words appear in ownerMessage. Both arguments are the RAW texts,
// before any marker neutralising, so the comparison is on the words the
// model and the owner actually wrote. A line with no content words at all
// is never accepted.
func pushedLineComesFromOwner(ownerMessage, line string) bool {
	owner := make(map[string]bool)
	for _, w := range contentWords(ownerMessage) {
		owner[w] = true
	}
	words := contentWords(line)
	if len(words) == 0 {
		return false
	}
	hits := 0
	for _, w := range words {
		if owner[w] {
			hits++
		}
	}
	return hits*2 >= len(words)
}
