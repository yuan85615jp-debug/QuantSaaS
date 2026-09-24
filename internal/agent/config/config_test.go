package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadAndRedact(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.agent.yaml")
	raw := `
agent_id: test-agent
saas:
  base_url: http://localhost:8080
  email: a@b.c
  password: secret-pass
  token: secret-token
broker:
  driver: paper
  api_key: key-xyz
  api_secret: sec-xyz
  initial_cash: 50000
`
	require.NoError(t, os.WriteFile(path, []byte(raw), 0o600))
	cfg, err := Load(path)
	require.NoError(t, err)
	require.Equal(t, "test-agent", cfg.AgentID)
	require.Equal(t, "secret-pass", cfg.SaaS.Password)
	red := cfg.Redacted()
	require.Equal(t, "***", red.SaaS.Password)
	require.Equal(t, "***", red.Broker.APIKey)
	require.Equal(t, "test-agent", red.AgentID)
}

func TestEnvOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.agent.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
agent_id: a1
broker:
  driver: paper
`), 0o600))
	t.Setenv("QS_AGENT_ID", "from-env")
	t.Setenv("QS_BROKER_API_KEY", "env-key")
	cfg, err := Load(path)
	require.NoError(t, err)
	require.Equal(t, "from-env", cfg.AgentID)
	require.Equal(t, "env-key", cfg.Broker.APIKey)
}
