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

	t.Run("session_read_is_scoped_to_the_connection_as_defence_in_depth", func(t *testing.T) {
		if !strings.Contains(conversationForSessionSQL, "connection_id") {
			t.Errorf("conversationForSessionSQL missing connection_id scoping: %q", conversationForSessionSQL)
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

	// Code review WR-03 of Phase 5: seq restarts at 1 in every game session,
	// so the one read that spans game sessions must not order by it.
	t.Run("the_read_that_spans_game_sessions_orders_by_a_key_that_never_restarts", func(t *testing.T) {
		stmt := strings.Join(strings.Fields(conversationForConnectionSQL), " ")
		if !strings.HasSuffix(stmt, "ORDER BY cl.created_at ASC, cl.id ASC") {
			t.Errorf("conversationForConnectionSQL must order by created_at then id: %q", stmt)
		}
		orderBy := stmt[strings.Index(stmt, "ORDER BY"):]
		if strings.Contains(orderBy, "seq") {
			t.Errorf("conversationForConnectionSQL orders by seq, which restarts in every game session: %q", orderBy)
		}
		// The sequence number really is per game session -- the reason above.
		if !strings.Contains(nextConversationSeqSQL, "WHERE game_session_id = $1") {
			t.Errorf("nextConversationSeqSQL is no longer scoped to one game session; revisit the ordering: %q", nextConversationSeqSQL)
		}
	})

	t.Run("the_reads_inside_one_game_session_keep_ordering_by_seq", func(t *testing.T) {
		if !strings.Contains(conversationForSessionSQL, "WHERE cl.game_session_id = $1") || !strings.Contains(conversationForSessionSQL, "ORDER BY cl.seq ASC") {
			t.Errorf("conversationForSessionSQL should read one game session ordered by seq: %q", conversationForSessionSQL)
		}
		if !strings.Contains(recentConversationSQL, "WHERE game_session_id = $1") || !strings.Contains(recentConversationSQL, "ORDER BY seq DESC") {
			t.Errorf("recentConversationSQL should read one game session ordered by seq: %q", recentConversationSQL)
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
