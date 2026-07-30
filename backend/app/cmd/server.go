package cmd

import (
	"context"
	"log/slog"

	"github.com/WillieBam/support_copilot/backend/app"
	"github.com/WillieBam/support_copilot/backend/app/config"
	"github.com/WillieBam/support_copilot/backend/internal/endpoint"
	"github.com/WillieBam/support_copilot/backend/middlewares"
	utilserver "github.com/WillieBam/support_copilot/backend/utils/server"
	"github.com/labstack/echo/v5"
	"github.com/spf13/cobra"
)

var supportCopilotCmd = &cobra.Command{
	Use:   "server",
	Short: "Run server",
	Long:  `Run Support Copilot Server`,
	Run:   supportCopilotExec,
}

func init() {
	rootCmd.AddCommand(supportCopilotCmd)
}

func supportCopilotExec(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	a := app.NewApp()
	h := endpoint.NewHandler(a.Service, a.AuthService, a.TeamService, a.Repository.User)

	s := utilserver.New(config.NewServerConfig("support-copilot"))

	if err := s.Start(ctx, func(e *echo.Echo) {
		e.Use(middlewares.RecoveryMiddleware())
		e.Use(middlewares.CORSMiddleware())

		e.POST("/auth/exchange", h.TokenExchangeHandler)

		// '/api' group endpoints
		apiGroup := e.Group("/api")
		apiGroup.Use(middlewares.AuthMiddleware(a.AuthService))
		apiGroup.GET("/auth/me", h.Me)
		apiGroup.POST("/alerts/ingest", h.IngestAlert)
		apiGroup.GET("/alerts/:id", h.RetrieveAlert)

		// team endpoints
		apiGroup.POST("/teams", h.CreateTeam)
		apiGroup.GET("/teams/me", h.GetTeams)
		apiGroup.DELETE("/teams/:team_id", h.DeleteTeam)
		apiGroup.GET("/teams/:team_id/members", h.GetTeamMembers)
		apiGroup.POST("/teams/:team_id/members", h.AddTeamMember)
		apiGroup.DELETE("/teams/:team_id/members/:user_id", h.RemoveTeamMember)
		apiGroup.POST("/teams/:team_id/incidents", h.AssignTeamIncident)
		apiGroup.GET("/teams/:team_id/incidents", h.GetTeamIncidents)
		apiGroup.GET("/incidents/:id", h.GetTeamIncident)
		apiGroup.PUT("/incidents/:id", h.UpdateTeamIncidentStatus)
		apiGroup.GET("/teams/:team_id/instruction", h.GetTeamInstruction)
		apiGroup.POST("/teams/:team_id/instruction", h.SaveTeamInstruction)
		apiGroup.GET("/users/search", h.SearchUsers)

		// conversation endpoints
		apiGroup.POST("/conversations", h.CreateConversation)
		apiGroup.GET("/teams/:team_id/conversations", h.ListTeamConversations)
		apiGroup.GET("/conversations/:id/messages", h.GetConversationMessages)

		// '/query' group endpoints
		g := e.Group("/query")
		g.Use(middlewares.AuthMiddleware(a.AuthService))
		g.POST("/chat", h.Query)

		// '/internal' group endpoints — protected by x-internal-api-key for MCP server calls
		internalGroup := e.Group("/internal", middlewares.InternalAPIKeyMiddleware())
		internalGroup.POST("/teams/:team_id/runbooks", h.CreateRunbook)
		internalGroup.PATCH("/runbooks/:id", h.UpdateRunbook)
		internalGroup.PATCH("/runbooks/:id/deprecate", h.DeprecateRunbook)
		internalGroup.GET("/runbooks/:id", h.GetRunbook)
		internalGroup.GET("/teams/:team_id/runbooks", h.ListRunbooks)
		internalGroup.GET("/teams/:team_id/incidents", h.GetTeamIncidents)
		internalGroup.GET("/incidents/:id/context", h.GetIncidentContext)

		// serve the React SPA with a client-side routing fallback
		// see static.go for the full routing strategy
		clientDir := config.Get().ClientDir
		if clientDir != "" {
			registerSPAStatic(e, clientDir)
		}
	}); err != nil {
		slog.Error("server gave up", "err", err)
	}
}
