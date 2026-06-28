package management

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/logging/conversationlog"
)

// ListConversationLogs returns paginated full-conversation log summaries.
func (h *Handler) ListConversationLogs(c *gin.Context) {
	h.listConversationLogs(c, false)
}

// TailConversationLogs returns the newest full-conversation log summaries.
func (h *Handler) TailConversationLogs(c *gin.Context) {
	h.listConversationLogs(c, true)
}

func (h *Handler) listConversationLogs(c *gin.Context, tail bool) {
	if h == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "handler unavailable"})
		return
	}

	store := h.conversationLogStore()
	if store == nil || !store.Enabled() {
		c.JSON(http.StatusOK, gin.H{
			"enabled":     false,
			"entries":     []conversationlog.EntrySummary{},
			"next_cursor": "",
			"malformed":   0,
			"tail":        tail,
		})
		return
	}

	query, err := parseConversationLogListQuery(c, !tail)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := store.List(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to list conversation logs: %v", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"enabled":     true,
		"entries":     result.Entries,
		"next_cursor": result.NextCursor,
		"malformed":   result.Malformed,
		"tail":        tail,
	})
}

// GetConversationLog returns one full-conversation log entry by opaque ID.
func (h *Handler) GetConversationLog(c *gin.Context) {
	if h == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "handler unavailable"})
		return
	}
	store := h.conversationLogStore()
	if store == nil || !store.Enabled() {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation log entry not found"})
		return
	}

	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing conversation log id"})
		return
	}
	entry, err := store.Read(id)
	if err != nil {
		if errors.Is(err, conversationlog.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "conversation log entry not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to read conversation log entry: %v", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"enabled": true,
		"entry":   entry,
	})
}

func parseConversationLogListQuery(c *gin.Context, allowCursor bool) (conversationlog.ListQuery, error) {
	query := conversationlog.ListQuery{
		RequestID: strings.TrimSpace(c.Query("request_id")),
		Provider:  strings.TrimSpace(c.Query("provider")),
		Model:     strings.TrimSpace(c.Query("model")),
		Path:      strings.TrimSpace(c.Query("path")),
	}
	if allowCursor {
		cursor := strings.TrimSpace(c.Query("cursor"))
		if cursor != "" {
			parsed, err := strconv.Atoi(cursor)
			if err != nil || parsed < 0 {
				return query, errors.New("invalid conversation log cursor")
			}
			query.Cursor = cursor
		}
	}

	limitRaw := strings.TrimSpace(c.Query("limit"))
	if limitRaw != "" {
		limit, err := strconv.Atoi(limitRaw)
		if err != nil || limit <= 0 {
			return query, errors.New("invalid conversation log limit")
		}
		query.Limit = limit
	}

	statusCodeRaw := strings.TrimSpace(c.Query("status_code"))
	if statusCodeRaw != "" {
		statusCode, err := strconv.Atoi(statusCodeRaw)
		if err != nil || statusCode < 0 || statusCode > 999 {
			return query, errors.New("invalid conversation log status_code")
		}
		query.StatusCode = &statusCode
	}

	hasErrorRaw := strings.TrimSpace(c.Query("has_error"))
	if hasErrorRaw != "" {
		hasError, err := strconv.ParseBool(hasErrorRaw)
		if err != nil {
			return query, errors.New("invalid conversation log has_error")
		}
		query.HasError = &hasError
	}

	from, err := parseConversationLogTimeFilter("from", c.Query("from"))
	if err != nil {
		return query, err
	}
	to, err := parseConversationLogTimeFilter("to", c.Query("to"))
	if err != nil {
		return query, err
	}
	if !from.IsZero() && !to.IsZero() && from.After(to) {
		return query, errors.New("invalid conversation log time range")
	}
	query.From = from
	query.To = to

	return query, nil
}

func parseConversationLogTimeFilter(name string, raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid conversation log %s", name)
	}
	return parsed, nil
}
