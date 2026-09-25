package ticker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestTickerDisabledStartStop(t *testing.T) {
	tk := New(nil, nil, nil, zap.NewNop(), Config{Enabled: false, Interval: time.Second})
	tk.Start()
	tk.Stop() // should not hang
	require.True(t, true)
}
