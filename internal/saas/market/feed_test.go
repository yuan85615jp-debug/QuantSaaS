package market

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/store"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type fakeProv struct {
	bars []BarInput
	err  error
}

func (f fakeProv) Name() string { return "fake" }
func (f fakeProv) FetchKlines(ctx context.Context, symbol, interval string, limit int) ([]BarInput, error) {
	return f.bars, f.err
}

func testFeedDB(t *testing.T) *store.DB {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, gdb.AutoMigrate(store.AllModels()...))
	return &store.DB{DB: gdb}
}

func TestFeedSyncSymbol(t *testing.T) {
	db := testFeedDB(t)
	svc := New(db)
	now := time.Now().UnixMilli()
	prov := fakeProv{bars: []BarInput{
		{OpenTime: now - 60_000, Open: 4.5, High: 4.6, Low: 4.4, Close: 4.55, Volume: 100},
		{OpenTime: now, Open: 4.55, High: 4.6, Low: 4.5, Close: 4.58, Volume: 120},
	}}
	feed := NewFeed(svc, prov, nil, zap.NewNop(), FeedConfig{
		Enabled: true, Interval: "1m", Limit: 50, Every: time.Hour,
	})
	n, err := feed.SyncSymbol(context.Background(), "510300", "1m", 50)
	require.NoError(t, err)
	require.Equal(t, 2, n)

	px, _, err := svc.LastClose("510300")
	require.NoError(t, err)
	require.InDelta(t, 4.58, px, 1e-9)

	feed2 := NewFeed(svc, prov, nil, zap.NewNop(), FeedConfig{Enabled: false})
	feed2.Start()
	feed2.Stop()
}
