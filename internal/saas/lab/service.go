package lab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/store"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrNotFound     = errors.New("evolution task not found")
	ErrNotAllowed   = errors.New("evolution not allowed for this app_role")
	ErrInvalidInput = errors.New("invalid input")
	ErrBadStatus    = errors.New("invalid task status for operation")
)

type Service struct {
	db     *store.DB
	log    *zap.Logger
	Runner Runner
}

type Runner interface {
	Run(ctx context.Context, task *store.EvolutionTask) (*RunResult, error)
}

type RunResult struct {
	GeneID      uint            `json:"gene_id"`
	Score       float64         `json:"score"`
	MaxDrawdown float64         `json:"max_drawdown"`
	Generations int             `json:"generations"`
	ParamPack   json.RawMessage `json:"param_pack"`
	History     json.RawMessage `json:"history,omitempty"`
}

func NewService(db *store.DB, log *zap.Logger, runner Runner) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{db: db, log: log, Runner: runner}
}

type CreateTaskRequest struct {
	StrategyID string          `json:"strategy_id"`
	Symbol     string          `json:"symbol"`
	Config     json.RawMessage `json:"config"`
}

func (s *Service) CreateTask(req CreateTaskRequest) (*store.EvolutionTask, error) {
	sid := strings.TrimSpace(req.StrategyID)
	sym := strings.ToUpper(strings.TrimSpace(req.Symbol))
	if sid == "" {
		sid = "lunar"
	}
	if sym == "" {
		return nil, fmt.Errorf("%w: symbol required", ErrInvalidInput)
	}
	cfg := req.Config
	if len(cfg) == 0 {
		cfg = json.RawMessage(`{}`)
	}
	task := store.EvolutionTask{
		StrategyID: sid, Symbol: sym, Status: store.EvoPending, Progress: 0, Config: datatypes.JSON(cfg),
	}
	if err := s.db.Create(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *Service) GetTask(id uint) (*store.EvolutionTask, error) {
	var t store.EvolutionTask
	if err := s.db.First(&t, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (s *Service) ListTasks(limit int) ([]store.EvolutionTask, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	var list []store.EvolutionTask
	err := s.db.Order("id DESC").Limit(limit).Find(&list).Error
	return list, err
}

func (s *Service) StartTask(ctx context.Context, id uint) (*store.EvolutionTask, error) {
	task, err := s.GetTask(id)
	if err != nil {
		return nil, err
	}
	if task.Status != store.EvoPending && task.Status != store.EvoFailed {
		return nil, fmt.Errorf("%w: status=%s", ErrBadStatus, task.Status)
	}
	if s.Runner == nil {
		return nil, fmt.Errorf("lab runner not configured")
	}
	now := time.Now()
	if err := s.db.Model(task).Updates(map[string]any{
		"status": store.EvoRunning, "progress": 0.01, "error_msg": "", "updated_at": now,
	}).Error; err != nil {
		return nil, err
	}
	task.Status = store.EvoRunning
	task.Progress = 0.01
	go s.runAsync(id)
	return task, nil
}

func (s *Service) runAsync(id uint) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	task, err := s.GetTask(id)
	if err != nil {
		s.log.Warn("lab task missing", zap.Uint("id", id), zap.Error(err))
		return
	}
	res, err := s.Runner.Run(ctx, task)
	if err != nil {
		_ = s.db.Model(&store.EvolutionTask{}).Where("id = ?", id).Updates(map[string]any{
			"status": store.EvoFailed, "error_msg": err.Error(), "progress": 1, "updated_at": time.Now(),
		}).Error
		s.log.Warn("lab task failed", zap.Uint("id", id), zap.Error(err))
		return
	}
	raw, _ := json.Marshal(res)
	_ = s.db.Model(&store.EvolutionTask{}).Where("id = ?", id).Updates(map[string]any{
		"status": store.EvoCompleted, "result": datatypes.JSON(raw), "progress": 1, "error_msg": "", "updated_at": time.Now(),
	}).Error
	s.log.Info("lab task completed", zap.Uint("id", id), zap.Uint("gene_id", res.GeneID), zap.Float64("score", res.Score))
}
