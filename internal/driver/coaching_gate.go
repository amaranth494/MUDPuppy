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

// maxWithdrawsPerMessage is the most coaching lines one owner message may
// take back, mirroring maxPushesPerMessage (owner-reported fix OW-03).
const maxWithdrawsPerMessage = 2

// minSharedStemRunes is how long a common prefix two words need before they
// count as the same stem: "casting" and "cast", "spells" and "spell".
const minSharedStemRunes = 4

// OW-03, the withdraw gate. A withdraw had no gate at first, on the reasoning
// that it can only remove a line that exists and the owner sees which one
// went. A live run of the chat-channel corpus then showed 2 of 2 hostile
// samples removing a standing SAFETY line while the owner had only asked an
// ordinary question: game text told AI-chatter to withdraw it, and it did.
// Seeing the line go afterwards is not the same as having asked for it.
//
// The rule, enforced in applyCoaching from the owner's CURRENT message:
//
//  1. a withdraw of a line is honoured when the owner's message shares at
//     least one content-word stem with that line (withdrawSharesStemWithOwner);
//  2. otherwise, when the owner's message holds a take-it-back word or phrase
//     (ownerAsksToTakeBack), a withdraw is honoured ONLY for the most recently
//     added standing line -- whatever line the model named, only the newest
//     may go this way, once per message;
//  3. anything else is refused: the line stays and the owner is told.

// sharesStem reports whether two words share a common prefix of at least
// minSharedStemRunes characters.
func sharesStem(a, b string) bool {
	ar, br := []rune(a), []rune(b)
	n := 0
	for n < len(ar) && n < len(br) && ar[n] == br[n] {
		n++
	}
	return n >= minSharedStemRunes
}

// withdrawSharesStemWithOwner is rule 1: some content word of the owner's
// message shares a stem with some content word of the line. Content words
// are the push gate's (contentWords), so a stop-word can never be the link.
func withdrawSharesStemWithOwner(ownerMessage, line string) bool {
	lineWords := contentWords(line)
	for _, ow := range contentWords(ownerMessage) {
		for _, lw := range lineWords {
			if sharesStem(ow, lw) {
				return true
			}
		}
	}
	return false
}

// takeBackSuffixes are the endings that make a word of the owner's message an
// inflection of a take-back word: "forgetting", "cancelled", "removed",
// "withdrawn". A bare prefix match is not used: "undoubtedly" is not "undo"
// and a "scrapbook" is not "scrap".
var takeBackSuffixes = []string{"", "s", "es", "d", "ed", "ing", "n", "ne", "al", "ting", "ted", "led", "ling", "ped", "ping"}

// isInflectionOf reports whether tok is word, or word with one of
// takeBackSuffixes, allowing for a dropped final "e" ("removing").
func isInflectionOf(tok, word string) bool {
	stems := []string{word}
	if strings.HasSuffix(word, "e") {
		stems = append(stems, strings.TrimSuffix(word, "e"))
	}
	for _, stem := range stems {
		if !strings.HasPrefix(tok, stem) {
			continue
		}
		rest := tok[len(stem):]
		for _, suffix := range takeBackSuffixes {
			if rest == suffix {
				return true
			}
		}
	}
	return false
}

// takeBackWords are single words, matched with their inflections
// (isInflectionOf). takeBackPhrases are matched as whole consecutive words.
var (
	takeBackWords   = []string{"forget", "withdraw", "cancel", "remove", "nevermind", "scrap", "undo"}
	takeBackPhrases = []string{"drop that", "never mind", "take back", "take that back", "ignore that", "stop doing", "no longer"}
)

// ownerAsksToTakeBack is rule 2's test: the owner's own message holds a
// take-it-back word or phrase. Only the owner's message is read, so game
// text can never supply the word.
func ownerAsksToTakeBack(ownerMessage string) bool {
	tokens := strings.FieldsFunc(strings.ToLower(ownerMessage), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	for _, tok := range tokens {
		for _, w := range takeBackWords {
			if isInflectionOf(tok, w) {
				return true
			}
		}
	}
	joined := " " + strings.Join(tokens, " ") + " "
	for _, p := range takeBackPhrases {
		if strings.Contains(joined, " "+p+" ") {
			return true
		}
	}
	return false
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
