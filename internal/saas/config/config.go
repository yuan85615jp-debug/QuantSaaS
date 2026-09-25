package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type AppRole string

const (
	RoleSaaS AppRole = "saas"
	RoleLab  AppRole = "lab"
	RoleDev  AppRole = "dev"
)

type Config struct {
	AppRole  AppRole        `yaml:"app_role"`
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	JWT      JWTConfig      `yaml:"jwt"`
	Market   MarketConfig   `yaml:"market"`
}

type ServerConfig struct {
	HTTPAddr string `yaml:"http_addr"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
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
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type JWTConfig struct {
	Secret     string `yaml:"secret"`
	ExpireHour int    `yaml:"expire_hour"`
}

type MarketConfig struct {
	Enabled  bool     `yaml:"enabled"`
	Provider string   `yaml:"provider"`
	Symbols  []string `yaml:"symbols"`
	Interval string   `yaml:"interval"`
	Limit    int      `yaml:"limit"`
	EverySec int      `yaml:"every_sec"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

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
	if v := os.Getenv("QS_MARKET_ENABLED"); v != "" {
		cfg.Market.Enabled = v == "1" || strings.EqualFold(v, "true")
	}
	if v := os.Getenv("QS_MARKET_SYMBOLS"); v != "" {
		parts := strings.Split(v, ",")
		cfg.Market.Symbols = nil
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				cfg.Market.Symbols = append(cfg.Market.Symbols, p)
			}
		}
	}
	if v := os.Getenv("QS_MARKET_INTERVAL"); v != "" {
		cfg.Market.Interval = v
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
	if c.Market.Provider == "" {
		c.Market.Provider = "eastmoney"
	}
	if c.Market.Interval == "" {
		c.Market.Interval = "1m"
	}
	if c.Market.Limit <= 0 {
		c.Market.Limit = 120
	}
	if c.Market.EverySec <= 0 {
		c.Market.EverySec = 60
	}
	if len(c.Market.Symbols) == 0 {
		c.Market.Symbols = []string{"510300"}
	}
	return nil
}

func (c *Config) AllowEvolution() bool {
	return c.AppRole == RoleLab || c.AppRole == RoleDev
}

func (c *Config) AllowTradeDispatch() bool {
	return c.AppRole == RoleSaaS || c.AppRole == RoleDev
}
