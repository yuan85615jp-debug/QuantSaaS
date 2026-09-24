package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// AppRole controls which capabilities are enabled.
// saas: live management + trade dispatch (no GA write)
// lab:  GA + backtest only (no trade dispatch)
// dev:  everything open
type AppRole string

const (
	RoleSaaS AppRole = "saas"
	RoleLab  AppRole = "lab"
	RoleDev  AppRole = "dev"
)

// Config is the root configuration loaded from config.yaml + env overrides.
type Config struct {
	AppRole  AppRole        `yaml:"app_role"`
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	JWT      JWTConfig      `yaml:"jwt"`
}

type ServerConfig struct {
	HTTPAddr string `yaml:"http_addr"` // e.g. ":8080"
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"` // prefer env: QS_DB_PASSWORD
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

func (d DatabaseConfig) DSN() string {
	ssl := d.SSLMode
	if ssl == "" {
		ssl = "disable"
	}
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, ssl,
	)
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`     // e.g. "127.0.0.1:6379"
	Password string `yaml:"password"` // prefer env: QS_REDIS_PASSWORD
	DB       int    `yaml:"db"`
}

type JWTConfig struct {
	Secret     string `yaml:"secret"` // prefer env: QS_JWT_SECRET
	ExpireHour int    `yaml:"expire_hour"`
}

// Load reads config from path, then overlays environment variables for secrets.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Env overrides for secrets (never commit real values)
	if v := os.Getenv("QS_APP_ROLE"); v != "" {
		cfg.AppRole = AppRole(strings.ToLower(v))
	}
	if v := os.Getenv("QS_DB_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("QS_DB_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v := os.Getenv("QS_REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if v := os.Getenv("QS_REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("QS_JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}
	if v := os.Getenv("QS_HTTP_ADDR"); v != "" {
		cfg.Server.HTTPAddr = v
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	switch c.AppRole {
	case RoleSaaS, RoleLab, RoleDev:
	default:
		return fmt.Errorf("invalid app_role %q, want saas|lab|dev", c.AppRole)
	}
	if c.JWT.Secret == "" || c.JWT.Secret == "CHANGE_ME" {
		return fmt.Errorf("jwt.secret must be set via config or QS_JWT_SECRET")
	}
	if c.JWT.ExpireHour <= 0 {
		c.JWT.ExpireHour = 72
	}
	if c.Server.HTTPAddr == "" {
		c.Server.HTTPAddr = ":8080"
	}
	return nil
}

// AllowEvolution is true when GA task create/run is permitted.
func (c *Config) AllowEvolution() bool {
	return c.AppRole == RoleLab || c.AppRole == RoleDev
}

// AllowTradeDispatch is true when TradeCommand may be sent to agents.
func (c *Config) AllowTradeDispatch() bool {
	return c.AppRole == RoleSaaS || c.AppRole == RoleDev
}
