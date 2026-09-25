package lab

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/store"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type fakeRunner struct{}

func (fakeRunner) Run(ctx context.Context, task *store.EvolutionTask) (*RunResult, error) {
	return &RunResult{GeneID: 1, Score: 0.12, MaxDrawdown: 0.05, Generations: 3, ParamPack: json.RawMessage(`{}`)}, nil
}

func testLabDB(t *testing.T) *store.DB {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, gdb.AutoMigrate(store.AllModels()...))
	return &store.DB{DB: gdb}
}

func TestCreateAndStartTask(t *testing.T) {
	db := testLabDB(t)
	svc := NewService(db, zap.NewNop(), fakeRunner{})
	task, err := svc.CreateTask(CreateTaskRequest{Symbol: "510300", StrategyID: "lunar"})
	require.NoError(t, err)
	require.Equal(t, store.EvoPending, task.Status)

	started, err := svc.StartTask(context.Background(), task.ID)
	require.NoError(t, err)
	require.Equal(t, store.EvoRunning, started.Status)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got, err := svc.GetTask(task.ID)
		require.NoError(t, err)
		if got.Status == store.EvoCompleted {
			require.NotEmpty(t, got.Result)
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("task did not complete")
}
