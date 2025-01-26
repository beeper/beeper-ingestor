package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/rs/zerolog/hlog"
	"go.mau.fi/gomuks/pkg/hicli/database"
	"maunium.net/go/mautrix/id"
)

type PaginationArg struct {
	Cursor    string // rowid of the event
	Direction string // "after" or "after"
}

type SearchMessagesQueryParams struct {
	Sender     string
	Before     int64
	After      int64
	Limit      int
	RoomID     string
	Pagination *PaginationArg
}

type PaginatedMessagesWithCursors struct {
	Items        []Message `json:"items"`
	HasMore      bool      `json:"has_more"`
	OldestCursor string    `json:"oldest_cursor"`
	NewestCursor string    `json:"newest_cursor"`
}

// SearchMessagesQuery represents search parameters for message queries

type SearchMessagesQuery struct {
	RoomID    id.RoomID
	Sender    id.UserID
	Before    int64
	After     int64
	Limit     int
	Cursor    database.EventRowID
	Direction string // "before" or "after"
}

// SearchMessages searches for messages with the given parameters
func (ab *BeeperIngestor) SearchMessagesDatabaseQuery(ctx context.Context, params SearchMessagesQuery) ([]*database.Event, error) {
	conditions := []string{"(event.type = 'm.room.message' OR event.decrypted_type = 'm.room.message')"}
	args := make([]any, 0)

	if params.RoomID != "" {
		conditions = append(conditions, "event.room_id = $"+strconv.Itoa(len(args)+1))
		args = append(args, params.RoomID)
	}

	if params.Sender != "" {
		conditions = append(conditions, "event.sender = $"+strconv.Itoa(len(args)+1))
		args = append(args, params.Sender)
	}

	if params.Before != 0 {
		conditions = append(conditions, "event.timestamp < $"+strconv.Itoa(len(args)+1))
		args = append(args, params.Before)
	}

	if params.After != 0 {
		conditions = append(conditions, "event.timestamp > $"+strconv.Itoa(len(args)+1))
		args = append(args, params.After)
	}

	if params.Cursor != 0 {
		if params.Direction == "before" {
			conditions = append(conditions, "event.rowid < $"+strconv.Itoa(len(args)+1))
		} else {
			conditions = append(conditions, "event.rowid > $"+strconv.Itoa(len(args)+1))
		}
		args = append(args, params.Cursor)
	}

	query := `
		SELECT event.rowid, COALESCE(timeline.rowid, 0) as timeline_rowid,
		       event.room_id, event_id, sender, type, state_key, timestamp, content, decrypted, decrypted_type,
		       unsigned, local_content, transaction_id, redacted_by, relates_to, relation_type,
		       megolm_session_id, decryption_error, send_error, reactions, last_edit_rowid, unread_type
		FROM event
		LEFT JOIN timeline ON event.rowid = timeline.event_rowid
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY event.timestamp DESC, event.rowid DESC
		LIMIT $` + strconv.Itoa(len(args)+1)

	args = append(args, params.Limit+1) // +1 to check for hasMore

	return ab.gmx.Client.DB.Event.QueryHelper.QueryMany(ctx, query, args...)
}


func (ab *BeeperIngestor) SearchMessages(w http.ResponseWriter, r *http.Request) {
	log := hlog.FromRequest(r)
	query := &SearchMessagesQueryParams{
		RoomID: r.URL.Query().Get("room_id"),
		Limit:  100, // Default limit
	}

	// Handle sender with proper Matrix UserID parsing
	if senderStr := r.URL.Query().Get("sender"); senderStr != "" {
		// Ensure the @ prefix is present
		if !strings.HasPrefix(senderStr, "@") {
			senderStr = "@" + senderStr
		}
		// Basic Matrix ID validation: @user:domain
		if !strings.Contains(senderStr, ":") {
			http.Error(w, "Invalid sender user ID format", http.StatusBadRequest)
			return
		}
		query.Sender = senderStr
	}

	// Parse limit if provided
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			http.Error(w, "Invalid limit parameter", http.StatusBadRequest)
			return
		}
		query.Limit = limit
	}

	// Parse before/after timestamps if provided
	if beforeStr := r.URL.Query().Get("before"); beforeStr != "" {
		before, err := strconv.ParseInt(beforeStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid before timestamp", http.StatusBadRequest)
			return
		}
		query.Before = before
	}

	if afterStr := r.URL.Query().Get("after"); afterStr != "" {
		after, err := strconv.ParseInt(afterStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid after timestamp", http.StatusBadRequest)
			return
		}
		query.After = after
	}

	// Parse pagination cursor if provided
	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		direction := r.URL.Query().Get("direction")
		if direction != "before" && direction != "after" {
			http.Error(w, "Invalid pagination direction, must be 'before' or 'after'", http.StatusBadRequest)
			return
		}
		query.Pagination = &PaginationArg{
			Cursor:    cursor,
			Direction: direction,
		}
	}

	searchParams := SearchMessagesQuery{
		RoomID:    id.RoomID(query.RoomID),
		Sender:    id.UserID(query.Sender),
		Before:    query.Before,
		After:     query.After,
		Limit:     max(1, min(query.Limit, 1000)),
		Direction: "before",
	}

	if query.Pagination != nil {
		cursor, err := strconv.ParseInt(query.Pagination.Cursor, 10, 64)
		if err != nil {
			http.Error(w, "Invalid cursor", http.StatusBadRequest)
			return
		}
		searchParams.Cursor = database.EventRowID(cursor)
		searchParams.Direction = query.Pagination.Direction
	}

	events, err := ab.SearchMessagesDatabaseQuery(r.Context(), searchParams)
	if err != nil {
		log.Err(err).Msg("Failed to query timeline")
		http.Error(w, "Failed to query timeline", http.StatusInternalServerError)
		return
	}

	// If no events found, return empty response instead of error
	if len(events) == 0 {
		response := &PaginatedMessagesWithCursors{
			Items:        []Message{},
			HasMore:      false,
			OldestCursor: "",
			NewestCursor: "",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get room info for all unique room IDs
	roomInfoMap := make(map[id.RoomID]*database.Room)
	for _, event := range events {
		if _, exists := roomInfoMap[event.RoomID]; !exists {
			room, err := ab.gmx.Client.DB.Room.Get(r.Context(), event.RoomID)
			if err != nil {
				log.Warn().Err(err).Str("room_id", string(event.RoomID)).Msg("Failed to get room info")
				continue
			}
			roomInfoMap[event.RoomID] = room
		}
	}

	var messages []Message
	var oldestRowID, newestRowID string

	for i, event := range events {
		if i < query.Limit {
			message := Message{
				URL:       fmt.Sprintf("https://matrix.to/#/%s/%s", event.RoomID, event.ID),
				Timestamp: event.Timestamp,
				SenderID:  event.Sender.String(),
				ID:        string(event.ID),
				RoomInfo: &RoomInfo{
					ID:  string(event.RoomID),
					URL: fmt.Sprintf("https://matrix.to/#/%s", event.RoomID),
				},
			}

			// Set room name from room info if available
			if room := roomInfoMap[event.RoomID]; room != nil && room.Name != nil {
				message.RoomInfo.Name = *room.Name
			} else {
				message.RoomInfo.Name = string(event.RoomID)
			}

			if event.LocalContent != nil && event.LocalContent.SanitizedHTML != "" {
				message.Text = event.LocalContent.SanitizedHTML
			} else {
				var content struct {
					Body string `json:"body"`
				}
				var rawContent []byte
				if event.LocalContent != nil && event.LocalContent.WasPlaintext || event.DecryptedType == "m.room.message" {
					rawContent = event.Decrypted
				} else {
					rawContent = event.Content
				}
				if err := json.Unmarshal(rawContent, &content); err == nil {
					message.Text = content.Body
				} else {
					message.Text = string(rawContent)
				}
			}
			var unsigned struct {
				Age     int `json:"age"`
				HSOrder int `json:"com.beeper.hs.order"`
			}
			if err := json.Unmarshal(event.Unsigned, &unsigned); err == nil {
				message.SortKey = unsigned.HSOrder
			}
			messages = append(messages, message)

			// Track cursors
			if oldestRowID == "" {
				oldestRowID = fmt.Sprint(event.RowID)
			}
			newestRowID = fmt.Sprint(event.RowID)
		}
	}

	hasMore := len(events) > query.Limit

	response := &PaginatedMessagesWithCursors{
		Items:        messages,
		HasMore:      hasMore,
		OldestCursor: oldestRowID,
		NewestCursor: newestRowID,
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Err(err).Msg("Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
