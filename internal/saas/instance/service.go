package instance

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/quant"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/store"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/strategies/lunar"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/strategy"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotFound       = errors.New("instance not found")
	ErrTemplate       = errors.New("strategy template not found")
	ErrInvalidInput   = errors.New("invalid input")
	ErrAlreadyRunning = errors.New("instance already running")
	ErrNotRunning     = errors.New("instance is not running")
	ErrForbidden      = errors.New("forbidden")
)

// FillInput is the SaaS-side application of one Agent FillReport.
// ClientOrderID is the idempotency key: a second call with the same id is a no-op.
type FillInput struct {
	ClientOrderID string
	InstanceID    uint
	Symbol        string
	Action        store.TradeAction
	Engine        store.TradeEngine
	Qty           float64
	Price         float64
	Fee           float64
	Status        string // filled | partial | failed
	ErrorMsg      string
}

// CreateRequest is the input for spinning up a new strategy instance.
type CreateRequest struct {
	UserID       uint
	TemplateID   string
	Symbol       string
	CapitalQuota float64 // initial CNY cash on the ledger
}

// Service owns StrategyInstance + PortfolioState lifecycle on the SaaS side.
type Service struct {
	db *store.DB
}

func NewService(db *store.DB) *Service {
	return &Service{db: db}
}

// EnsureTemplates upserts built-in strategy templates (idempotent).
func (s *Service) EnsureTemplates() error {
	strat, ok := strategy.Get(lunar.StrategyID)
	if !ok {
		strat = lunar.NewDefault()
	}
	m := strat.Manifest()
	raw, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	tpl := store.StrategyTemplate{
		ID:       m.ID,
		Name:     m.Name,
		Version:  m.Version,
		IsSpot:   m.IsSpot,
		Manifest: datatypes.JSON(raw),
	}
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "version", "is_spot", "manifest", "updated_at"}),
	}).Create(&tpl).Error
}

// Create allocates a STOPPED instance and an empty portfolio with capital_quota as cash.
func (s *Service) Create(req CreateRequest) (*store.StrategyInstance, error) {
	req.TemplateID = strings.TrimSpace(req.TemplateID)
	req.Symbol = strings.ToUpper(strings.TrimSpace(req.Symbol))
	if req.UserID == 0 || req.TemplateID == "" || req.Symbol == "" {
		return nil, fmt.Errorf("%w: user_id, template_id, symbol required", ErrInvalidInput)
	}
	if req.CapitalQuota < 0 {
		return nil, fmt.Errorf("%w: capital_quota must be >= 0", ErrInvalidInput)
	}

	var tpl store.StrategyTemplate
	if err := s.db.First(&tpl, "id = ?", req.TemplateID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTemplate
		}
		return nil, err
	}

	var out *store.StrategyInstance
	err := s.db.Transaction(func(tx *gorm.DB) error {
		inst := store.StrategyInstance{
			UserID:       req.UserID,
			TemplateID:   req.TemplateID,
			Symbol:       req.Symbol,
			Status:       store.InstanceStopped,
			CapitalQuota: req.CapitalQuota,
		}
		if geneID, ok, gerr := s.lookupChampionID(tx, req.TemplateID, req.Symbol); gerr != nil {
			return gerr
		} else if ok {
			inst.ChampionGeneID = &geneID
		}
		if err := tx.Create(&inst).Error; err != nil {
			return err
		}
		port := store.PortfolioState{
			InstanceID:  inst.ID,
			CNYBalance:  req.CapitalQuota,
			TotalEquity: req.CapitalQuota,
		}
		if err := tx.Create(&port).Error; err != nil {
			return err
		}
		rt := store.RuntimeState{
			InstanceID: inst.ID,
			Snapshot:   datatypes.JSON([]byte(`{}`)),
		}
		if err := tx.Create(&rt).Error; err != nil {
			return err
		}
		out = &inst
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) lookupChampionID(tx *gorm.DB, strategyID, symbol string) (uint, bool, error) {
	var g store.GeneRecord
	err := tx.Where("strategy_id = ? AND role = ? AND symbol = ?", strategyID, store.GeneChampion, symbol).
		Order("activated_at DESC, id DESC").First(&g).Error
	if err == nil {
		return g.ID, true, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, false, err
	}
	err = tx.Where("strategy_id = ? AND role = ?", strategyID, store.GeneChampion).
		Order("activated_at DESC, id DESC").First(&g).Error
	if err == nil {
		return g.ID, true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, false, nil
	}
	return 0, false, err
}

func (s *Service) Get(id uint, userID uint) (*store.StrategyInstance, error) {
	var inst store.StrategyInstance
	q := s.db.Where("id = ?", id)
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	if err := q.First(&inst).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &inst, nil
}

func (s *Service) ListByUser(userID uint) ([]store.StrategyInstance, error) {
	var list []store.StrategyInstance
	if err := s.db.Where("user_id = ?", userID).Order("id ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (s *Service) Start(id uint, userID uint) (*store.StrategyInstance, error) {
	inst, err := s.Get(id, userID)
	if err != nil {
		return nil, err
	}
	if inst.Status == store.InstanceRunning {
		return nil, ErrAlreadyRunning
	}
	if geneID, ok, gerr := s.lookupChampionID(s.db.DB, inst.TemplateID, inst.Symbol); gerr != nil {
		return nil, gerr
	} else if ok {
		inst.ChampionGeneID = &geneID
	}
	inst.Status = store.InstanceRunning
	inst.LastError = ""
	if err := s.db.Model(inst).Updates(map[string]interface{}{
		"status":           inst.Status,
		"champion_gene_id": inst.ChampionGeneID,
		"last_error":       "",
	}).Error; err != nil {
		return nil, err
	}
	return inst, nil
}

func (s *Service) Stop(id uint, userID uint) (*store.StrategyInstance, error) {
	inst, err := s.Get(id, userID)
	if err != nil {
		return nil, err
	}
	if inst.Status == store.InstanceStopped {
		return nil, ErrNotRunning
	}
	inst.Status = store.InstanceStopped
	if err := s.db.Model(inst).Update("status", store.InstanceStopped).Error; err != nil {
		return nil, err
	}
	return inst, nil
}

func (s *Service) MarkError(id uint, msg string) error {
	return s.db.Model(&store.StrategyInstance{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     store.InstanceError,
		"last_error": msg,
	}).Error
}

func (s *Service) GetPortfolio(instanceID uint) (*store.PortfolioState, error) {
	var p store.PortfolioState
	if err := s.db.Where("instance_id = ?", instanceID).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (s *Service) ApplyRelease(instanceID uint, qty float64, price float64) error {
	if qty <= 0 {
		return fmt.Errorf("%w: release qty must be > 0", ErrInvalidInput)
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var p store.PortfolioState
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("instance_id = ?", instanceID).First(&p).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if qty > p.DeadHold {
			qty = p.DeadHold
		}
		if qty <= 0 {
			return nil
		}
		p.DeadHold -= qty
		p.FloatHold += qty
		if price > 0 {
			p.TotalEquity = p.CNYBalance + (p.DeadHold+p.FloatHold+p.ColdSealedHold)*price
		}
		return tx.Save(&p).Error
	})
}

func (s *Service) ApplyFill(instanceID uint, action store.TradeAction, engine store.TradeEngine, qty, price, fee float64) error {
	return s.applyPortfolioFill(s.db.DB, instanceID, action, engine, qty, price, fee)
}

// ApplyFillReport persists TradeRecord + SpotExecution and updates the ledger.
// Idempotent on ClientOrderID: if a TradeRecord already exists for that id, returns nil
// without mutating the portfolio again (safe under Agent WS redelivery).
func (s *Service) ApplyFillReport(in FillInput) error {
	in.ClientOrderID = strings.TrimSpace(in.ClientOrderID)
	if in.ClientOrderID == "" {
		return fmt.Errorf("%w: client_order_id required", ErrInvalidInput)
	}
	if in.InstanceID == 0 {
		return fmt.Errorf("%w: instance_id required", ErrInvalidInput)
	}
	status := strings.ToLower(strings.TrimSpace(in.Status))
	if status == "" {
		status = "filled"
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		var existing store.TradeRecord
		err := tx.Where("client_order_id = ?", in.ClientOrderID).First(&existing).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var exec store.SpotExecution
		execErr := tx.Where("client_order_id = ?", in.ClientOrderID).First(&exec).Error
		if execErr == nil && (exec.Status == store.ExecFilled || exec.Status == store.ExecFailed) {
			return nil
		}
		if execErr != nil && !errors.Is(execErr, gorm.ErrRecordNotFound) {
			return execErr
		}

		symbol := strings.ToUpper(strings.TrimSpace(in.Symbol))
		if symbol == "" {
			var inst store.StrategyInstance
			if e := tx.Select("symbol").First(&inst, in.InstanceID).Error; e == nil {
				symbol = inst.Symbol
			}
		}

		if status == "failed" {
			return s.upsertExecution(tx, in, symbol, store.ExecFailed, 0, 0, 0)
		}

		if in.Qty <= 0 || in.Price <= 0 {
			return fmt.Errorf("%w: qty and price must be > 0", ErrInvalidInput)
		}
		if in.Action != store.ActionBuy && in.Action != store.ActionSell {
			return fmt.Errorf("%w: unknown action %s", ErrInvalidInput, in.Action)
		}
		if in.Engine != store.EngineMacro && in.Engine != store.EngineMicro {
			in.Engine = store.EngineMicro
		}

		if err := s.applyPortfolioFill(tx, in.InstanceID, in.Action, in.Engine, in.Qty, in.Price, in.Fee); err != nil {
			return err
		}

		rec := store.TradeRecord{
			InstanceID:    in.InstanceID,
			ClientOrderID: in.ClientOrderID,
			Action:        in.Action,
			Engine:        in.Engine,
			Symbol:        symbol,
			FilledQty:     in.Qty,
			FilledPrice:   in.Price,
			Fee:           in.Fee,
		}
		if err := tx.Create(&rec).Error; err != nil {
			if isUniqueViolation(err) {
				return nil
			}
			return err
		}

		execStatus := store.ExecFilled
		return s.upsertExecution(tx, in, symbol, execStatus, in.Qty, in.Price, in.Fee)
	})
}

func (s *Service) applyPortfolioFill(tx *gorm.DB, instanceID uint, action store.TradeAction, engine store.TradeEngine, qty, price, fee float64) error {
	if qty <= 0 || price <= 0 {
		return fmt.Errorf("%w: qty and price must be > 0", ErrInvalidInput)
	}
	var p store.PortfolioState
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("instance_id = ?", instanceID).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	notional := qty * price
	switch action {
	case store.ActionBuy:
		cost := notional + fee
		if cost > p.CNYBalance {
			return fmt.Errorf("%w: insufficient cash (need %.2f have %.2f)", ErrInvalidInput, cost, p.CNYBalance)
		}
		p.CNYBalance -= cost
		if engine == store.EngineMacro {
			p.DeadHold += qty
		} else {
			p.FloatHold += qty
		}
	case store.ActionSell:
		if qty > p.FloatHold {
			return fmt.Errorf("%w: insufficient float hold", ErrInvalidInput)
		}
		p.FloatHold -= qty
		p.CNYBalance += notional - fee
	default:
		return fmt.Errorf("%w: unknown action %s", ErrInvalidInput, action)
	}
	p.TotalEquity = p.CNYBalance + (p.DeadHold+p.FloatHold+p.ColdSealedHold)*price
	return tx.Save(&p).Error
}

func (s *Service) upsertExecution(tx *gorm.DB, in FillInput, symbol string, status store.ExecutionStatus, filledQty, filledPrice, fee float64) error {
	var exec store.SpotExecution
	err := tx.Where("client_order_id = ?", in.ClientOrderID).First(&exec).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		exec = store.SpotExecution{
			InstanceID:    in.InstanceID,
			ClientOrderID: in.ClientOrderID,
			Status:        status,
			Action:        in.Action,
			Symbol:        symbol,
			RequestQty:    in.Qty,
			FilledQty:     filledQty,
			FilledPrice:   filledPrice,
			Fee:           fee,
			ErrorMsg:      in.ErrorMsg,
		}
		return tx.Create(&exec).Error
	}
	exec.Status = status
	exec.FilledQty = filledQty
	exec.FilledPrice = filledPrice
	exec.Fee = fee
	if in.ErrorMsg != "" {
		exec.ErrorMsg = in.ErrorMsg
	}
	if in.Action != "" {
		exec.Action = in.Action
	}
	if symbol != "" {
		exec.Symbol = symbol
	}
	return tx.Save(&exec).Error
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate")
}

// RecordPendingExecution writes a pending SpotExecution when a TradeCommand is dispatched.
// Idempotent on client_order_id (existing row is left unchanged).
func (s *Service) RecordPendingExecution(instanceID uint, clientOrderID, symbol string, action store.TradeAction, requestQty float64) error {
	clientOrderID = strings.TrimSpace(clientOrderID)
	if clientOrderID == "" || instanceID == 0 {
		return fmt.Errorf("%w: client_order_id and instance_id required", ErrInvalidInput)
	}
	var existing store.SpotExecution
	err := s.db.Where("client_order_id = ?", clientOrderID).First(&existing).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	exec := store.SpotExecution{
		InstanceID:    instanceID,
		ClientOrderID: clientOrderID,
		Status:        store.ExecPending,
		Action:        action,
		Symbol:        strings.ToUpper(strings.TrimSpace(symbol)),
		RequestQty:    requestQty,
	}
	if err := s.db.Create(&exec).Error; err != nil {
		if isUniqueViolation(err) {
			return nil
		}
		return err
	}
	return nil
}

func ToQuantPortfolio(p *store.PortfolioState) quant.Portfolio {
	if p == nil {
		return quant.Portfolio{}
	}
	return quant.Portfolio{
		Cash:           p.CNYBalance,
		DeadHold:       p.DeadHold,
		FloatHold:      p.FloatHold,
		ColdSealedHold: p.ColdSealedHold,
	}
}

func (s *Service) LoadChampionParams(inst *store.StrategyInstance) (json.RawMessage, error) {
	if inst == nil {
		return nil, ErrInvalidInput
	}
	if inst.ChampionGeneID != nil {
		var g store.GeneRecord
		if err := s.db.First(&g, *inst.ChampionGeneID).Error; err == nil && len(g.ParamPack) > 0 {
			return json.RawMessage(g.ParamPack), nil
		}
	}
	if geneID, ok, err := s.lookupChampionID(s.db.DB, inst.TemplateID, inst.Symbol); err != nil {
		return nil, err
	} else if ok {
		var g store.GeneRecord
		if err := s.db.First(&g, geneID).Error; err == nil && len(g.ParamPack) > 0 {
			return json.RawMessage(g.ParamPack), nil
		}
	}
	raw, err := json.Marshal(lunar.DefaultParams())
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func (s *Service) PromoteGene(geneID uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var g store.GeneRecord
		if err := tx.First(&g, geneID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		now := time.Now()
		q := tx.Model(&store.GeneRecord{}).
			Where("strategy_id = ? AND role = ?", g.StrategyID, store.GeneChampion)
		if g.Symbol != "" {
			q = q.Where("symbol = ? OR symbol = '' OR symbol IS NULL", g.Symbol)
		}
		if err := q.Update("role", store.GeneRetired).Error; err != nil {
			return err
		}
		g.Role = store.GeneChampion
		g.ActivatedAt = &now
		return tx.Save(&g).Error
	})
}
