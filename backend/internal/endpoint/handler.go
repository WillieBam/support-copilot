package endpoint

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	"github.com/WillieBam/support_copilot/backend/types"
	"github.com/WillieBam/support_copilot/backend/types/models"
	"github.com/WillieBam/support_copilot/backend/types/requests"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type Handler struct {
	apps        interfaces.IAppService
	authService interfaces.IAuthService
	teamService interfaces.ITeamService
	userService interfaces.IUserService
}

func NewHandler(a interfaces.IAppService, authService interfaces.IAuthService, opts ...interface{}) *Handler {
	h := &Handler{
		apps:        a,
		authService: authService,
	}
	for _, opt := range opts {
		if ts, ok := opt.(interfaces.ITeamService); ok {
			h.teamService = ts
		}
		if us, ok := opt.(interfaces.IUserService); ok {
			h.userService = us
		}
	}
	return h
}

// TokenExchangeHandler converts a validated Firebase token into a JWT session token
func (h *Handler) TokenExchangeHandler(c *echo.Context) error {
	var req requests.TokenExchangeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing request payload"})
	}

	if req.FirebaseToken == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing firebase request"})
	}

	verified, claims, err := h.authService.ExchangeToken(c.Request().Context(), req.FirebaseToken)
	if err != nil {
		if err.Error() == "mfa_required" {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error":   "mfa_required",
				"message": "TOTP verification required",
			})
		}
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	var expires time.Time
	if claims != nil && claims.ExpiresAt != nil {
		expires = claims.ExpiresAt.Time
	} else {
		expires = time.Now().Add(1 * time.Hour)
	}

	cookie := &http.Cookie{
		Name:     "support_copilot_session",
		Value:    verified,
		Expires:  expires,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(cookie)
	slog.Info("Successfully created and attached HttpOnly session cookie",
		"user_uid", claims.FirebaseUID,
		"expires_at", expires.Format(time.RFC3339),
	)
	return c.JSON(http.StatusOK, map[string]string{"status": "authenticated"})
}

func (h *Handler) Me(c *echo.Context) error {
	uidVal := c.Get("user_uid")
	appUID, ok := uidVal.(string)
	if !ok || appUID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized session"})
	}

	emailVal := c.Get("user_email")
	email, _ := emailVal.(string)

	// return info about who is authenticated to revive React UI client state
	return c.JSON(http.StatusOK, map[string]interface{}{
		"authenticated": true,
		"user_uid":      appUID,
		"user_email":    email,
	})
}

// Query handles POST /query/chat
func (h *Handler) Query(c *echo.Context) error {
	uidVal := c.Get("user_uid")
	appUID, ok := uidVal.(string)
	if !ok || appUID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required: missing support copilot token context",
		})
	}

	log.Printf("[LOG] Successfully authenticated user UID: %s processing query stream.", appUID)
	var req requests.ChatQueryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Input == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "input is required"})
	}

	var convID uuid.UUID
	var isFirstMessage bool
	ctx := c.Request().Context()

	// get database user record safely if userService is configured
	if h.userService != nil {
		dbUser, err := h.userService.GetUserByFirebaseUID(ctx, appUID)
		if err == nil && dbUser != nil {
			if req.ConversationID != nil && *req.ConversationID != uuid.Nil {
				convID = *req.ConversationID
			} else if req.TeamID != nil && *req.TeamID != uuid.Nil {
				conv, err := h.apps.CreateConversation(ctx, *req.TeamID, dbUser.ID)
				if err == nil && conv != nil {
					convID = conv.ID
					isFirstMessage = true
				}
			}
		}
	}

	// save user message if conversation exists
	if convID != uuid.Nil {
		_, _ = h.apps.SaveMessage(ctx, convID, "user", req.Input, "")
	}

	resp := c.Response()
	resp.Header().Set("Content-Type", "text/event-stream")
	resp.Header().Set("Cache-Control", "no-cache")
	resp.Header().Set("Connection", "keep-alive")
	resp.WriteHeader(http.StatusOK)

	flusher, ok := resp.(http.Flusher)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Streaming unsupported"})
	}
	flusher.Flush()

	// notify client of assigned conversation id
	if convID != uuid.Nil && isFirstMessage {
		metaEvent := types.StreamEvent{
			Type:    "meta",
			Content: convID.String(),
		}
		eventJSON, _ := json.Marshal(metaEvent)
		fmt.Fprintf(resp, "data: %s\n\n", eventJSON)
		flusher.Flush()
	}

	var teamID uuid.UUID
	if req.TeamID != nil && *req.TeamID != uuid.Nil {
		teamID = *req.TeamID
	} else if convID != uuid.Nil {
		if conv, err := h.apps.GetConversationByID(ctx, convID); err == nil && conv != nil {
			teamID = conv.TeamID
		}
	}

	streamChan := make(chan types.StreamEvent)
	errorChan := make(chan error, 1)

	var opts []interface{}
	if teamID != uuid.Nil {
		opts = append(opts, teamID)
	}

	go func() {
		// pass the channel into the service so it can push events
		err := h.apps.QueryStreamWithTools(c.Request().Context(), req.Input, req.History, streamChan, opts...)
		if err != nil {
			errorChan <- err
		}
		// always close the channel when the service is done
		close(streamChan)
	}()

	var fullAssistantText strings.Builder
	var fullReasoningText strings.Builder

	for {
		select {
		case event, ok := <-streamChan:
			if !ok {
				// stream finished cleanly, persist assistant message and generate title
				if convID != uuid.Nil {
					asstText := fullAssistantText.String()
					reasText := fullReasoningText.String()
					if asstText != "" || reasText != "" {
						_, _ = h.apps.SaveMessage(context.Background(), convID, "assistant", asstText, reasText)
					}
					if isFirstMessage {
						title, err := h.apps.GenerateAndSaveTitle(context.Background(), convID, req.Input, asstText)
						if err == nil && title != "" {
							titleEvent := types.StreamEvent{
								Type:    "title",
								Content: title,
							}
							eventJSON, _ := json.Marshal(titleEvent)
							fmt.Fprintf(resp, "data: %s\n\n", eventJSON)
							flusher.Flush()
						}
					}
				}
				return nil
			}

			if event.Type == "text" {
				fullAssistantText.WriteString(event.Content)
			} else if event.Type == "reasoning" {
				fullReasoningText.WriteString(event.Content)
			} else if event.Type == "drain" {
				fullAssistantText.Reset()
				fullReasoningText.Reset()
			}

			eventJSON, _ := json.Marshal(event)
			fmt.Fprintf(resp, "data: %s\n\n", eventJSON)
			flusher.Flush()

		case err := <-errorChan:
			slog.Error("[STREAM ERROR]: query stream failed", "err", err)
			errEvent := types.StreamEvent{
				Type: "text",
				// always use fmt.Sprintf to build json
				Content: fmt.Sprintf("\n\n**Error** %s", err.Error()),
			}
			// always marshal with json.Marshal
			eventJSON, _ := json.Marshal(errEvent)

			fmt.Fprintf(resp, "data: %s\n\n", eventJSON)
			flusher.Flush()
			return nil

		case <-c.Request().Context().Done():
			log.Println("[STREAM]: Client Disconnected (prompt edited or stopped). Aborting stream gracefully.")
			return nil

		}
	}

}

func (h *Handler) IngestAlert(c *echo.Context) error {
	var req requests.AlertIngestRequest
	if err := c.Bind(&req); err != nil {
		slog.Error("[ALERT] Failed to bind alert request payload", "err", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid alert payload"})
	}

	slog.Info("[ALERT] Alert ingestion request received", "service", req.ServiceName, "severity", req.Severity, "incident_id", req.IncidentID)

	if req.ServiceName == "" {
		slog.Warn("[ALERT] Missing required service name in alert payload")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "service name is required"})
	}

	err := h.apps.IngestAlert(c.Request().Context(), req.IncidentID, req.ServiceName, req.Severity, string(req.Metrics))
	if err != nil {
		slog.Error("[ALERT] Failed to ingest alert via app service", "service", req.ServiceName, "err", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	slog.Info("[ALERT] Alert successfully ingested", "service", req.ServiceName)
	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

func (h *Handler) RetrieveAlert(c *echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid alert ID"})
	}

	a, err := h.apps.RetrieveAlert(c.Request().Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, a)
}

type UserSearchResult struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
}

func (h *Handler) SearchUsers(c *echo.Context) error {
	uidVal := c.Get("user_uid")
	appUID, ok := uidVal.(string)
	if !ok || appUID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized session"})
	}

	q := c.QueryParam("q")
	if len(q) < 2 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "search query must be at least 2 characters"})
	}

	users, err := h.userService.SearchUsers(c.Request().Context(), q, 10)
	if err != nil {
		slog.Error("[user] Error searching users", "error", err, "query", q)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to search users"})
	}

	var results []UserSearchResult
	for _, u := range users {
		results = append(results, UserSearchResult{
			ID:          u.ID,
			Email:       u.Email,
			DisplayName: u.DisplayName,
		})
	}

	return c.JSON(http.StatusOK, results)
}

// createconversation handles POST /api/conversations
func (h *Handler) CreateConversation(c *echo.Context) error {
	uidVal := c.Get("user_uid")
	appUID, ok := uidVal.(string)
	if !ok || appUID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized session"})
	}

	var req requests.CreateConversationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if h.userService == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user service unavailable"})
	}

	dbUser, err := h.userService.GetUserByFirebaseUID(c.Request().Context(), appUID)
	if err != nil || dbUser == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "user not found"})
	}

	conv, err := h.apps.CreateConversation(c.Request().Context(), req.TeamID, dbUser.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, conv)
}

// listteamconversations handles GET /api/teams/:team_id/conversations
func (h *Handler) ListTeamConversations(c *echo.Context) error {
	teamIDStr := c.Param("team_id")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid team id"})
	}

	limit := 0
	if l := c.QueryParam("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	convs, err := h.apps.ListTeamConversations(c.Request().Context(), teamID, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if convs == nil {
		convs = []models.Conversation{}
	}
	return c.JSON(http.StatusOK, convs)
}

// getconversationmessages handles GET /api/conversations/:id/messages
func (h *Handler) GetConversationMessages(c *echo.Context) error {
	convIDStr := c.Param("id")
	convID, err := uuid.Parse(convIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid conversation id"})
	}

	msgs, err := h.apps.ListMessagesByConversation(c.Request().Context(), convID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if msgs == nil {
		msgs = []models.Message{}
	}
	return c.JSON(http.StatusOK, msgs)
}
