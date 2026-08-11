package server

import (
	"fmt"
	"time"
)

type Config struct {
	Name            string
	Port            int
	ShutdownTimeout time.Duration
}

func (c Config) Addr() string {
	if c.Port == 0 {
		return ":8080"
	}
	return fmt.Sprintf(":%d", c.Port)
}

func (c Config) Timeout() time.Duration {
	if c.ShutdownTimeout == 0 {
		return 10 * time.Second
	}
	return c.ShutdownTimeout
}

