package config

import (
	"log"
	"log/slog"
	"strings"
	"time"

	"github.com/spf13/viper"
)

var (
	cfg    *viper.Viper
	config *Config
)

type IConfig interface {
	GetString(key string) string
}

type Config struct {
	ClientDir string `mapstructure:"client_dir"`

	Http struct {
		Port            int
		ShutdownTimeOut time.Duration
	}

	Database struct {
		Host     string
		Port     int
		User     string
		Password string
		Name     string
	}

	Auth struct {
		Enabled      bool
		TOTPRequired bool   `mapstructure:"totp_required"`
		JWTSecret    string `mapstructure:"jwt_secret"`
	}

	Firebase struct {
		ProjectID          string `mapstructure:"project_id"`
		ServiceAccountPath string `mapstructure:"service_account_path"`
	}

	Ollama struct {
		Model     string
		BaseURL   string        `mapstructure:"base_url"`
		KeepAlive string        `mapstructure:"keep_alive"`
		Timeout   time.Duration `mapstructure:"timeout"`
		NumCtx    int           `mapstructure:"num_ctx"`
	}

	MCP1 struct {
		Host string
		Port string
	}

	MCP2 struct {
		Host string
		Port string
	}
}

func Get() *Config {
	if config != nil {
		return config
	}
	newConfig()
	config = &Config{}
	if err := cfg.Unmarshal(config); err != nil {
		log.Printf("Failed to unmarshal config: %v", err)
	}
	return config
}

func newConfig() IConfig {
	if cfg != nil {
		return cfg
	}
	cfg = viper.New()
	cfg.SetDefault("client_dir", "frontend/dist")
	cfg.SetDefault("http.port", 8080)
	cfg.SetDefault("http.shutdownTimeOut", 10*time.Second)
	cfg.SetDefault("database.host", "localhost")
	cfg.SetDefault("database.port", 5432)
	cfg.SetDefault("database.user", "postgres")
	cfg.SetDefault("database.password", "supportcopilot")
	cfg.SetDefault("database.name", "copilot")
	cfg.SetDefault("auth.enabled", false)
	cfg.SetDefault("auth.totp_required", false)
	cfg.SetDefault("auth.jwt_secret", "local_development_fallback_secret_key_32_bytes_long")
	cfg.SetDefault("firebase.project_id", "")
	cfg.SetDefault("firebase.service_account_path", "backend/app/config/serviceAccountKey.json")
	cfg.SetDefault("ollama.model", "llama3.2:latest")
	cfg.SetDefault("ollama.base_url", "http://localhost:11434")
	cfg.SetDefault("ollama.keep_alive", "30m")
	cfg.SetDefault("ollama.timeout", 5*time.Minute)
	cfg.SetDefault("ollama.num_ctx", 2048)
	cfg.SetDefault("mcp1.host", "localhost")
	cfg.SetDefault("mcp1.port", 9000)
	cfg.SetDefault("mcp2.host", "localhost")
	cfg.SetDefault("mcp2.port", 9000)
	cfg.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	cfg.AutomaticEnv()
	cfg.SetConfigName("config")
	cfg.SetConfigType("yaml")
	cfg.AddConfigPath(".")
	cfg.AddConfigPath("./config")

	if err := cfg.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			slog.Error("Failed to read config", "err", err)
		}
	}

	return cfg
}
