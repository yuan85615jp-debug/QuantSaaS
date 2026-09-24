package instance

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/store"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/strategies/lunar"
	_ "github.com/yuan85615jp-debug/QuantSaaS/internal/strategies/lunar" // register
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testDB(t *testing.T) *store.DB {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open("file:" + t.Name() + "?mode=memory&cache=shared"), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, gdb.AutoMigrate(store.AllModels()...))
	return &store.DB{DB: gdb}
}

func TestEnsureTemplatesAndCreateStartStop(t *testing.T) {
	db := testDB(t)
	svc := NewService(db)

	require.NoError(t, svc.EnsureTemplates())

	var tpl store.StrategyTemplate
	require.NoError(t, db.First(&tpl, "id = ?", lunar.StrategyID).Error)
	require.Equal(t, "lunar", tpl.ID)
	require.True(t, tpl.IsSpot)

	u := store.User{Email: "a@b.c", PasswordHash: "x", Role: "user"}
	require.NoError(t, db.Create(&u).Error)

	inst, err := svc.Create(CreateRequest{
		UserID: u.ID, TemplateID: lunar.StrategyID, Symbol: "510300", CapitalQuota: 100_000,
	})
	require.NoError(t, err)
	require.Equal(t, store.InstanceStopped, inst.Status)
	require.Nil(t, inst.ChampionGeneID)

	port, err := svc.GetPortfolio(inst.ID)
	require.NoError(t, err)
	require.InDelta(t, 100_000, port.CNYBalance, 1e-9)
	require.Equal(t, 0.0, port.DeadHold)

	running, err := svc.Start(inst.ID, u.ID)
	require.NoError(t, err)
	require.Equal(t, store.InstanceRunning, running.Status)

	_, err = svc.Start(inst.ID, u.ID)
	require.ErrorIs(t, err, ErrAlreadyRunning)

	stopped, err := svc.Stop(inst.ID, u.ID)
	require.NoError(t, err)
	require.Equal(t, store.InstanceStopped, stopped.Status)

	list, err := svc.ListByUser(u.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestApplyReleaseAndFill(t *testing.T) {
	db := testDB(t)
	svc := NewService(db)
	require.NoError(t, svc.EnsureTemplates())
	u := store.User{Email: "c@d.e", PasswordHash: "x"}
	require.NoError(t, db.Create(&u).Error)

	inst, err := svc.Create(CreateRequest{
		UserID: u.ID, TemplateID: lunar.StrategyID, Symbol: "518880", CapitalQuota: 50_000,
	})
	require.NoError(t, err)

	require.NoError(t, svc.ApplyFill(inst.ID, store.ActionBuy, store.EngineMacro, 1000, 10, 30))
	port, err := svc.GetPortfolio(inst.ID)
	require.NoError(t, err)
	require.InDelta(t, 50_000-10_000-30, port.CNYBalance, 1e-6)
	require.InDelta(t, 1000, port.DeadHold, 1e-9)
	require.Equal(t, 0.0, port.FloatHold)

	require.NoError(t, svc.ApplyRelease(inst.ID, 400, 10))
	port, err = svc.GetPortfolio(inst.ID)
	require.NoError(t, err)
	require.InDelta(t, 600, port.DeadHold, 1e-9)
	require.InDelta(t, 400, port.FloatHold, 1e-9)

	require.NoError(t, svc.ApplyFill(inst.ID, store.ActionSell, store.EngineMicro, 200, 11, 5))
	port, err = svc.GetPortfolio(inst.ID)
	require.NoError(t, err)
	require.InDelta(t, 200, port.FloatHold, 1e-9)
	require.InDelta(t, 50_000-10030+2200-5, port.CNYBalance, 1e-6)

	qp := ToQuantPortfolio(port)
	require.InDelta(t, port.CNYBalance, qp.Cash, 1e-9)
	require.InDelta(t, 600, qp.DeadHold, 1e-9)
}

func TestChampionBindAndPromote(t *testing.T) {
	db := testDB(t)
	svc := NewService(db)
	require.NoError(t, svc.EnsureTemplates())
	u := store.User{Email: "g@h.i", PasswordHash: "x"}
	require.NoError(t, db.Create(&u).Error)

	params := lunar.DefaultParams()
	params.Kp = 2.5
	raw, err := json.Marshal(params)
	require.NoError(t, err)
	now := time.Now()
	gene := store.GeneRecord{
		StrategyID: lunar.StrategyID,
		Symbol:     "510300",
		Role:       store.GeneChallenger,
		ParamPack:  raw,
		ScoreTotal: 1.23,
	}
	require.NoError(t, db.Create(&gene).Error)

	require.NoError(t, svc.PromoteGene(gene.ID))
	var g store.GeneRecord
	require.NoError(t, db.First(&g, gene.ID).Error)
	require.Equal(t, store.GeneChampion, g.Role)
	require.NotNil(t, g.ActivatedAt)
	_ = now

	inst, err := svc.Create(CreateRequest{
		UserID: u.ID, TemplateID: lunar.StrategyID, Symbol: "510300", CapitalQuota: 10_000,
	})
	require.NoError(t, err)
	require.NotNil(t, inst.ChampionGeneID)
	require.Equal(t, gene.ID, *inst.ChampionGeneID)

	pack, err := svc.LoadChampionParams(inst)
	require.NoError(t, err)
	var loaded lunar.Params
	require.NoError(t, json.Unmarshal(pack, &loaded))
	require.InDelta(t, 2.5, loaded.Kp, 1e-9)

	inst2, err := svc.Create(CreateRequest{
		UserID: u.ID, TemplateID: lunar.StrategyID, Symbol: "159915", CapitalQuota: 1,
	})
	require.NoError(t, err)
	pack2, err := svc.LoadChampionParams(inst2)
	require.NoError(t, err)
	require.NotEmpty(t, pack2)
}

func TestOwnership(t *testing.T) {
	db := testDB(t)
	svc := NewService(db)
	require.NoError(t, svc.EnsureTemplates())
	u1 := store.User{Email: "1@x.y", PasswordHash: "x"}
	u2 := store.User{Email: "2@x.y", PasswordHash: "x"}
	require.NoError(t, db.Create(&u1).Error)
	require.NoError(t, db.Create(&u2).Error)

	inst, err := svc.Create(CreateRequest{
		UserID: u1.ID, TemplateID: lunar.StrategyID, Symbol: "510300", CapitalQuota: 1,
	})
	require.NoError(t, err)

	_, err = svc.Get(inst.ID, u2.ID)
	require.ErrorIs(t, err, ErrNotFound)
}
