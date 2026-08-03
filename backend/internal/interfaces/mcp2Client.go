package interfaces

import (
	"context"

	"github.com/WillieBam/support_copilot/backend/types/requests"
)

// imcptwoclient is the contract for invoking mcp server 2 knowledge base tools
type IMCP2Client interface {
	CreateRunbook(ctx context.Context, args requests.MCP2CreateRunbookArgs) (string, error)
	UpdateRunbook(ctx context.Context, args requests.MCP2UpdateRunbookArgs) (string, error)
	DeprecateRunbook(ctx context.Context, args requests.MCP2DeprecateRunbookArgs) (string, error)
	GetRunbook(ctx context.Context, args requests.MCP2GetRunbookArgs) (string, error)
	ListRunbooks(ctx context.Context, args requests.MCP2ListRunbooksArgs) (string, error)
	GetIncident(ctx context.Context, args requests.MCP2GetIncidentArgs) (string, error)
	ListIncidents(ctx context.Context, args requests.MCP2ListIncidentsArgs) (string, error)
}
