package store

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TestConversationStoreSQL asserts the package-level SQL constants' exact
// shape and the two Go-side guard checks, without a database connection --
// none is started or required by this test.
func TestConversationStoreSQL(t *testing.T) {
	t.Run("statements_name_the_conversation_lines_table", func(t *testing.T) {
		for name, stmt := range map[string]string{
			"nextConversationSeqSQL":      nextConversationSeqSQL,
			"insertConversationLineSQL":   insertConversationLineSQL,
			"conversationForConnectionSQL": conversationForConnectionSQL,
			"conversationForSessionSQL":   conversationForSessionSQL,
			"recentConversationSQL":       recentConversationSQL,
		} {
			if !strings.Contains(stmt, "conversation_lines") {
				t.Errorf("%s does not name conversation_lines: %q", name, stmt)
			}
		}
	})

	t.Run("login_scoped_read_bounds_on_login_started_at", func(t *testing.T) {
		if !strings.Contains(conversationForConnectionSQL, "login_started_at") {
			t.Errorf("conversationForConnectionSQL missing login_started_at bound: %q", conversationForConnectionSQL)
		}
	})

	t.Run("every_full_read_orders_ascending", func(t *testing.T) {
		for name, stmt := range map[string]string{
			"conversationForConnectionSQL": conversationForConnectionSQL,
			"conversationForSessionSQL":   conversationForSessionSQL,
		} {
			if !strings.Contains(stmt, "ASC") {
				t.Errorf("%s does not order ascending: %q", name, stmt)
			}
		}
	})

	t.Run("recent_conversation_orders_descending_with_a_limit", func(t *testing.T) {
		if !strings.Contains(recentConversationSQL, "DESC") {
			t.Errorf("recentConversationSQL does not order descending: %q", recentConversationSQL)
		}
		if !strings.Contains(recentConversationSQL, "LIMIT") {
			t.Errorf("recentConversationSQL does not carry a LIMIT: %q", recentConversationSQL)
		}
	})

	t.Run("append_chat_line_rejects_unknown_speaker_before_touching_the_database", func(t *testing.T) {
		s := &ConversationStore{}
		_, err := s.AppendChatLine(uuid.New(), "attacker", "hello")
		if err == nil {
			t.Fatal("expected an error for an unknown speaker, got nil")
		}
	})

	t.Run("recent_conversation_rejects_a_non_positive_limit_before_touching_the_database", func(t *testing.T) {
		s := &ConversationStore{}
		lines, err := s.RecentConversation(uuid.New(), 0)
		if err != nil {
			t.Fatalf("RecentConversation(0) error = %v, want nil", err)
		}
		if len(lines) != 0 {
			t.Errorf("RecentConversation(0) = %v, want empty slice", lines)
		}

		lines, err = s.RecentConversation(uuid.New(), -5)
		if err != nil {
			t.Fatalf("RecentConversation(-5) error = %v, want nil", err)
		}
		if len(lines) != 0 {
			t.Errorf("RecentConversation(-5) = %v, want empty slice", lines)
		}
	})
}
