package config

import (
	"log"
	"log/slog"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
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
	ClientDir      string `mapstructure:"client_dir"`
	InternalAPIKey string `mapstructure:"internal_api_key"`

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

	LLM struct {
		Provider      string        `mapstructure:"provider"`
		Model         string        `mapstructure:"model"`
		APIKey        string        `mapstructure:"api_key"`
		BaseURL       string        `mapstructure:"base_url"`
		KeepAlive     string        `mapstructure:"keep_alive"`
		Timeout       time.Duration `mapstructure:"timeout"`
		NumCtx        int           `mapstructure:"num_ctx"`
		TLSSkipVerify bool          `mapstructure:"tls_skip_verify"`
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
	if err := cfg.Unmarshal(config, func(dc *mapstructure.DecoderConfig) {
		dc.WeaklyTypedInput = true
		dc.DecodeHook = mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		)
	}); err != nil {
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
	cfg.SetDefault("llm.provider", "ollama")
	cfg.SetDefault("llm.model", "llama3.2:latest")
	cfg.SetDefault("llm.api_key", "")
	cfg.SetDefault("llm.base_url", "http://localhost:11434")
	cfg.SetDefault("llm.keep_alive", "30m")
	cfg.SetDefault("llm.timeout", 5*time.Minute)
	cfg.SetDefault("llm.num_ctx", 2048)
	cfg.SetDefault("llm.tls_skip_verify", false)
	cfg.SetDefault("mcp1.host", "localhost")
	cfg.SetDefault("mcp1.port", 9000)
	cfg.SetDefault("mcp2.host", "localhost")
	cfg.SetDefault("mcp2.port", 9000)
	cfg.BindEnv("llm.provider", "LLM_PROVIDER")
	cfg.BindEnv("llm.model", "LLM_MODEL")
	cfg.BindEnv("llm.base_url", "LLM_BASE_URL")
	cfg.BindEnv("llm.api_key", "LLM_API_KEY", "GEMINI_API_KEY", "OPENAI_API_KEY")
	cfg.BindEnv("llm.tls_skip_verify", "LLM_TLS_SKIP_VERIFY")
	cfg.BindEnv("database.user", "DATABASE_USER", "DB_USER")
	cfg.BindEnv("database.password", "DATABASE_PASSWORD", "DB_PASSWORD")
	cfg.BindEnv("database.name", "DATABASE_NAME", "DB_NAME")
	cfg.BindEnv("database.port", "DATABASE_PORT", "DB_PORT")
	cfg.BindEnv("database.host", "DATABASE_HOST", "DB_HOST")
	cfg.BindEnv("http.port", "HTTP_PORT", "SERVER_PORT")
	cfg.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	cfg.AutomaticEnv()

	// Optionally load .env if present
	envCfg := viper.New()
	envCfg.SetConfigFile(".env")
	if err := envCfg.ReadInConfig(); err == nil {
		for _, k := range envCfg.AllKeys() {
			if !cfg.IsSet(k) || cfg.GetString(k) == "" {
				cfg.Set(k, envCfg.Get(k))
			}
		}
	}

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
