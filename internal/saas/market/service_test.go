package market

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/store"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testDB(t *testing.T) *store.DB {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, gdb.AutoMigrate(store.AllModels()...))
	return &store.DB{DB: gdb}
}

func TestImportAndList(t *testing.T) {
	db := testDB(t)
	svc := New(db)
	base := time.Now().Add(-10 * time.Minute).UnixMilli()
	n, err := svc.ImportBars("510300", "1m", []BarInput{
		{OpenTime: base, Open: 4.5, High: 4.52, Low: 4.48, Close: 4.50, Volume: 100},
		{OpenTime: base + 60_000, Open: 4.50, High: 4.51, Low: 4.49, Close: 4.49, Volume: 110},
	})
	require.NoError(t, err)
	require.Equal(t, 2, n)

	n2, err := svc.ImportBars("510300", "1m", []BarInput{
		{OpenTime: base + 60_000, Open: 4.50, High: 4.51, Low: 4.49, Close: 4.48, Volume: 120},
	})
	require.NoError(t, err)
	require.Equal(t, 1, n2)

	rows, err := svc.ListBars("510300", "1m", 10)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, 4.48, rows[1].Close)

	px, _, err := svc.LastClose("510300")
	require.NoError(t, err)
	require.Equal(t, 4.48, px)
}

func TestSeedSynthetic(t *testing.T) {
	db := testDB(t)
	svc := New(db)
	n, err := svc.SeedSynthetic(SeedOpts{Symbol: "510300", Bars: 50, StartPx: 4.5})
	require.NoError(t, err)
	require.Equal(t, 50, n)
	rows, err := svc.ListBars("510300", "1m", 50)
	require.NoError(t, err)
	require.Len(t, rows, 50)
	require.True(t, rows[len(rows)-1].Close > 0)
}
