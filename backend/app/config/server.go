package config

import (
	"github.com/WillieBam/support_copilot/backend/utils/server"
)

type IServer = server.IServer

func NewServerConfig(name string) *server.ServerConfig {
	c := Get()
	return server.NewServerConfig(name, c.Http.Port, c.Http.ShutdownTimeOut)
}
