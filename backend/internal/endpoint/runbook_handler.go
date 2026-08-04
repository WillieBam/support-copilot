package endpoint

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/WillieBam/support_copilot/backend/types/requests"
	"github.com/WillieBam/support_copilot/backend/types/responses"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

// CreateRunbook handles post /internal/teams/:team_id/runbooks
func (h *Handler) CreateRunbook(c *echo.Context) error {
	teamID, err := uuid.Parse(c.Param("team_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid team_id"})
	}

	var req requests.CreateRunbookRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	if strings.TrimSpace(req.Title) == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "title is required"})
	}
	if strings.TrimSpace(req.Content) == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "content is required"})
	}

	rb, err := h.teamService.CreateRunbook(c.Request().Context(), teamID, req.IncidentID, req.Title, req.Content)
	if err != nil {
		slog.Error("[runbook] CreateRunbook failed", "team_id", teamID, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, rb)
}

// UpdateRunbook handles patch /internal/runbooks/:id
func (h *Handler) UpdateRunbook(c *echo.Context) error {
	runbookID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid runbook id"})
	}

	var req requests.UpdateRunbookRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	rb, err := h.teamService.UpdateRunbook(c.Request().Context(), runbookID, req.Title, req.Content)
	if err != nil {
		slog.Error("[runbook] UpdateRunbook failed", "runbook_id", runbookID, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, rb)
}

// DeprecateRunbook handles patch /internal/runbooks/:id/deprecate
func (h *Handler) DeprecateRunbook(c *echo.Context) error {
	runbookID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid runbook id"})
	}

	rb, err := h.teamService.DeprecateRunbook(c.Request().Context(), runbookID)
	if err != nil {
		slog.Error("[runbook] DeprecateRunbook failed", "runbook_id", runbookID, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, rb)
}

// GetRunbook handles get /internal/runbooks/:id
func (h *Handler) GetRunbook(c *echo.Context) error {
	runbookID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid runbook id"})
	}

	rb, err := h.teamService.GetRunbook(c.Request().Context(), runbookID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "runbook not found"})
	}
	return c.JSON(http.StatusOK, rb)
}

// ListRunbooks handles get /internal/teams/:team_id/runbooks?status=active
func (h *Handler) ListRunbooks(c *echo.Context) error {
	teamID, err := uuid.Parse(c.Param("team_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid team_id"})
	}

	status := c.QueryParam("status")
	if status == "" {
		status = "active"
	}

	runbooks, err := h.teamService.ListRunbooks(c.Request().Context(), teamID, status)
	if err != nil {
		slog.Error("[runbook] ListRunbooks failed", "team_id", teamID, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, runbooks)
}

// ListIncidentsInternal handles GET /internal/teams/:team_id/incidents
func (h *Handler) ListIncidentsInternal(c *echo.Context) error {
	teamID, err := uuid.Parse(c.Param("team_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid team_id"})
	}

	incidents, err := h.teamService.ListTeamIncidents(c.Request().Context(), teamID)
	if err != nil {
		slog.Error("[runbook] ListIncidentsInternal failed", "team_id", teamID, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, incidents)
}

// ─── incident context handler ─────────────────────────────────────────────────

// relativeTime formats a past time as a human-readable string like "2h ago"
func relativeTime(t time.Time) string {
	dur := time.Since(t)
	if dur < 0 {
		dur = 0
	}
	if dur < time.Minute {
		return "just now"
	}
	if dur < time.Hour {
		return fmt.Sprintf("%dm ago", int(math.Round(dur.Minutes())))
	}
	if dur < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(math.Round(dur.Hours())))
	}
	return fmt.Sprintf("%dd ago", int(math.Round(dur.Hours()/24)))
}

// otelKeyMap normalises verbose opentelemetry metric key names to short readable names
var otelKeyMap = map[string]string{
	"container.cpu.usage":              "cpu_usage_pct",
	"system.cpu.system":                "cpu_system_pct",
	"runtime.go.mem_stats.total_alloc": "mem_alloc_bytes",
	"error_rate":                       "error_rate",
	"trace.grpc.server.request.hits":   "grpc_hits",
}

// parseAndCleanseMetrics converts raw metric json into a compact map
//   - normalises otel key names to short readable names
//   - drops near-zero float values (< 0 01) to reduce llm token noise
//   - converts large byte values to megabytes
func parseAndCleanseMetrics(raw string) map[string]any {
	if raw == "" {
		return map[string]any{}
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return map[string]any{"raw": raw}
	}

	cleaned := make(map[string]any, len(parsed))
	for k, v := range parsed {
		f, isFloat := v.(float64)
		if isFloat && f < 0.01 {
			continue // drop near-zero noise
		}

		// convert large byte values -> megabytes
		shortKey := k
		if isFloat && strings.Contains(k, "alloc") && f > 1_000_000 {
			v = math.Round(f/1_000_000*10) / 10
			shortKey = strings.Replace(k, "total_alloc", "alloc_mb", 1)
		}

		if norm, ok := otelKeyMap[shortKey]; ok {
			cleaned[norm] = v
		} else if norm, ok := otelKeyMap[k]; ok {
			cleaned[norm] = v
		} else {
			// use the last segment of dot-separated otel key as a short name
			parts := strings.Split(k, ".")
			cleaned[parts[len(parts)-1]] = v
		}
	}
	return cleaned
}

// GetIncidentContext handles get /internal/incidents/:id/context
// returns a cleansed, llm-optimised incident context - timeline capped at 3 entries
// metrics noise-filtered, timestamps as relative strings
func (h *Handler) GetIncidentContext(c *echo.Context) error {
	incidentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid incident id"})
	}

	inc, alerts, err := h.teamService.GetIncidentContext(c.Request().Context(), incidentID)
	if err != nil {
		slog.Error("[runbook] GetIncidentContext failed", "incident_id", incidentID, "error", err)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "incident not found"})
	}

	// build cleansed alert list
	cleansedAlerts := make([]responses.CleansedAlert, 0, len(alerts))
	for _, a := range alerts {
		cleansedAlerts = append(cleansedAlerts, responses.CleansedAlert{
			Service:    a.ServiceName,
			Severity:   a.Severity,
			Received:   relativeTime(a.ReceivedAt),
			KeyMetrics: parseAndCleanseMetrics(a.Metrics),
		})
	}

	// build timeline - cap at 3 most recent transitions (history is already asc)
	history := inc.History
	timeline := make([]responses.TimelineEntry, 0, 4)
	if len(history) > 3 {
		omitted := len(history) - 3
		timeline = append(timeline, responses.TimelineEntry{
			Note: fmt.Sprintf("... %d earlier transition(s) omitted", omitted),
		})
		history = history[len(history)-3:]
	}
	for _, h := range history {
		from := h.PreviousStatus
		if from == "" {
			from = "—"
		}
		timeline = append(timeline, responses.TimelineEntry{
			At:   relativeTime(h.UpdatedAt),
			From: from,
			To:   h.NewStatus,
			Note: h.Details,
		})
	}

	// fetch active runbooks tied to this incident so llm knows what already exists
	existingRunbooks, _ := h.teamService.ListRunbooks(c.Request().Context(), inc.TeamID, "active")
	summaries := make([]responses.RunbookSummary, 0)
	for _, rb := range existingRunbooks {
		if rb.IncidentID == inc.IncidentID {
			summaries = append(summaries, responses.RunbookSummary{
				ID:     rb.ID.String(),
				Title:  rb.Title,
				Status: rb.Status,
			})
		}
	}

	return c.JSON(http.StatusOK, responses.IncidentContextResponse{
		IncidentID:       inc.ID.String(),
		Title:            inc.Title,
		Status:           inc.Status,
		Age:              relativeTime(inc.AssignedAt),
		Details:          inc.Details,
		Alerts:           cleansedAlerts,
		Timeline:         timeline,
		ExistingRunbooks: summaries,
	})
}
