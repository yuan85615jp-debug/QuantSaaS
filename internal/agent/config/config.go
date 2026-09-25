package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	AgentID   string       `yaml:"agent_id"`
	SaaS      SaaSConfig   `yaml:"saas"`
	Broker    BrokerConfig `yaml:"broker"`
	Instances []uint       `yaml:"instances"`
}

type SaaSConfig struct {
	BaseURL  string `yaml:"base_url"`
	Email    string `yaml:"email"`
	Password string `yaml:"password"`
	Token    string `yaml:"token"`
}

type BrokerConfig struct {
	Driver         string  `yaml:"driver"`
	APIKey         string  `yaml:"api_key"`
	APISecret      string  `yaml:"api_secret"`
	AccountID      string  `yaml:"account_id"`
	CommissionRate float64 `yaml:"commission_rate"`
	StampTaxRate   float64 `yaml:"stamp_tax_rate"`
	InitialCash    float64 `yaml:"initial_cash"`
	LotStep        float64 `yaml:"lot_step"`
	LotMin         float64 `yaml:"lot_min"`
	BaseURL        string  `yaml:"base_url"`
	DryRun         bool    `yaml:"dry_run"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read agent config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse agent config: %w", err)
	}
	if v := os.Getenv("QS_AGENT_PASSWORD"); v != "" {
		cfg.SaaS.Password = v
	}
	if v := os.Getenv("QS_AGENT_TOKEN"); v != "" {
		cfg.SaaS.Token = v
	}
	if v := os.Getenv("QS_BROKER_API_KEY"); v != "" {
		cfg.Broker.APIKey = v
	}
	if v := os.Getenv("QS_BROKER_API_SECRET"); v != "" {
		cfg.Broker.APISecret = v
	}
	if v := os.Getenv("QS_AGENT_ID"); v != "" {
		cfg.AgentID = v
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.AgentID) == "" {
		return fmt.Errorf("agent_id required")
	}
	if c.Broker.Driver == "" {
		c.Broker.Driver = "paper"
	}
	switch c.Broker.Driver {
	case "paper":
	case "live":
		if !c.Broker.DryRun && c.Broker.APIKey == "" {
			return fmt.Errorf("broker.api_key required for live (or set dry_run: true / QS_BROKER_API_KEY)")
		}
	default:
		return fmt.Errorf("unsupported broker.driver %q (paper|live)", c.Broker.Driver)
	}
	if c.Broker.CommissionRate < 0 {
		c.Broker.CommissionRate = 0.0003
	}
	if c.Broker.StampTaxRate < 0 {
		c.Broker.StampTaxRate = 0.001
	}
	if c.Broker.InitialCash <= 0 {
		c.Broker.InitialCash = 100_000
	}
	if c.Broker.LotStep <= 0 {
		c.Broker.LotStep = 100
	}
	if c.Broker.LotMin <= 0 {
		c.Broker.LotMin = 100
	}
	return nil
}

func (c *Config) Redacted() Config {
	out := *c
	out.SaaS.Password = mask(out.SaaS.Password)
	out.SaaS.Token = mask(out.SaaS.Token)
	out.Broker.APIKey = mask(out.Broker.APIKey)
	out.Broker.APISecret = mask(out.Broker.APISecret)
	return out
}

func mask(s string) string {
	if s == "" {
		return ""
	}
	return "***"
}
