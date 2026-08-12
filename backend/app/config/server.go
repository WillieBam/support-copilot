package config

import (
	"github.com/WillieBam/support_copilot/backend/utils/server"
)

// NewServerConfig builds a server config from current app settings
func NewServerConfig(name string) server.Config {
	c := Get()
	return server.Config{
		Name:            name,
		Port:            c.Http.Port,
		ShutdownTimeout: c.Http.ShutdownTimeOut,
	}
}

