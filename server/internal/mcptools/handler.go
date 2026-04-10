package mcptools

import (
	serverproto "github.com/eberle1080/mcp-protocol/server"
	"github.com/eberle1080/repo-depot/server/internal/service"
)

// RegisterAll registers all repo-depot MCP tools on the given registry.
func RegisterAll(h *serverproto.DefaultHandler, svc *service.RepodepotService) error {
	reg := h.Registry

	if err := registerProjectTools(reg, svc); err != nil {
		return err
	}

	if err := registerPRTools(reg, svc); err != nil {
		return err
	}

	if err := registerGtTools(reg, svc); err != nil {
		return err
	}

	if err := registerSyncTools(reg, svc); err != nil {
		return err
	}

	if err := registerBuildTools(reg, svc); err != nil {
		return err
	}

	return nil
}
