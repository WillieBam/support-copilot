package server

import (
	"strconv"
	"time"
)

type IServer interface {
	Name() string
	Port() string
	GetShutdownTimeOutDuration() time.Duration
}

type ServerConfig struct {
	ConfigName              string
	ServerPort              int
	ShutdownTimeoutDuration time.Duration
}

func NewServerConfig(name string, port int, shutdownTimeout time.Duration) *ServerConfig {
	return &ServerConfig{
		ConfigName:              name,
		ServerPort:              port,
		ShutdownTimeoutDuration: shutdownTimeout,
	}
}

func (c *ServerConfig) Name() string {
	return c.ConfigName
}

func (c *ServerConfig) Port() string {
	if c.ServerPort == 0 {
		return "8080"
	}

	return strconv.Itoa(c.ServerPort)
}

func (c *ServerConfig) GetShutdownTimeOutDuration() time.Duration {
	if c.ShutdownTimeoutDuration == 0 {
		return 10 * time.Second
	}
	return c.ShutdownTimeoutDuration
}
