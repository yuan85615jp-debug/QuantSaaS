package store

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ---------- User & Auth ----------

type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Email        string         `gorm:"uniqueIndex;size:255;not null" json:"email"`
	PasswordHash string         `gorm:"size:255;not null" json:"-"`
	DisplayName  string         `gorm:"size:128" json:"display_name"`
	Role         string         `gorm:"size:32;default:user" json:"role"`
	Plan         string         `gorm:"size:32;default:free" json:"plan"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

type StrategyTemplate struct {
	ID        string         `gorm:"primaryKey;size:64" json:"id"`
	Name      string         `gorm:"size:128;not null" json:"name"`
	Version   string         `gorm:"size:32;not null" json:"version"`
	IsSpot    bool           `gorm:"not null;default:true" json:"is_spot"`
	Manifest  datatypes.JSON `gorm:"type:jsonb" json:"manifest"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type InstanceStatus string

const (
	InstanceRunning InstanceStatus = "RUNNING"
	InstanceStopped InstanceStatus = "STOPPED"
	InstanceError   InstanceStatus = "ERROR"
)

type StrategyInstance struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	UserID         uint           `gorm:"index;not null" json:"user_id"`
	TemplateID     string         `gorm:"size:64;index;not null" json:"template_id"`
	Symbol         string         `gorm:"size:32;not null" json:"symbol"`
	Status         InstanceStatus `gorm:"size:16;index;not null;default:STOPPED" json:"status"`
	CapitalQuota   float64        `gorm:"not null;default:0" json:"capital_quota"`
	ChampionGeneID *uint          `gorm:"index" json:"champion_gene_id,omitempty"`
	LastError      string         `gorm:"type:text" json:"last_error,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	User           User             `gorm:"foreignKey:UserID" json:"-"`
	Template       StrategyTemplate `gorm:"foreignKey:TemplateID" json:"-"`
}

type PortfolioState struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	InstanceID           uint      `gorm:"uniqueIndex;not null" json:"instance_id"`
	CNYBalance           float64   `gorm:"not null;default:0" json:"cny_balance"`
	DeadHold             float64   `gorm:"not null;default:0" json:"dead_hold"`
	FloatHold            float64   `gorm:"not null;default:0" json:"float_hold"`
	ColdSealedHold       float64   `gorm:"not null;default:0" json:"cold_sealed_hold"`
	TotalEquity          float64   `gorm:"not null;default:0" json:"total_equity"`
	LastProcessedBarTime int64     `gorm:"not null;default:0" json:"last_processed_bar_time"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type RuntimeState struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	InstanceID uint           `gorm:"uniqueIndex;not null" json:"instance_id"`
	Snapshot   datatypes.JSON `gorm:"type:jsonb" json:"snapshot"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type LotType string

const (
	LotDeadStack  LotType = "DEAD_STACK"
	LotFloating   LotType = "FLOATING"
	LotColdSealed LotType = "COLD_SEALED"
)

type SpotLot struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	InstanceID   uint      `gorm:"index;not null" json:"instance_id"`
	LotType      LotType   `gorm:"size:16;index;not null" json:"lot_type"`
	Amount       float64   `gorm:"not null" json:"amount"`
	CostPrice    float64   `gorm:"not null" json:"cost_price"`
	IsColdSealed bool      `gorm:"not null;default:false" json:"is_cold_sealed"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type TradeAction string

const (
	ActionBuy  TradeAction = "BUY"
	ActionSell TradeAction = "SELL"
)

type TradeEngine string

const (
	EngineMacro TradeEngine = "MACRO"
	EngineMicro TradeEngine = "MICRO"
)

type TradeRecord struct {
	ID            uint        `gorm:"primaryKey" json:"id"`
	InstanceID    uint        `gorm:"index;not null" json:"instance_id"`
	ClientOrderID string      `gorm:"uniqueIndex;size:64;not null" json:"client_order_id"`
	Action        TradeAction `gorm:"size:8;not null" json:"action"`
	Engine        TradeEngine `gorm:"size:16;not null" json:"engine"`
	Symbol        string      `gorm:"size:32;not null" json:"symbol"`
	FilledQty     float64     `gorm:"not null;default:0" json:"filled_qty"`
	FilledPrice   float64     `gorm:"not null;default:0" json:"filled_price"`
	Fee           float64     `gorm:"not null;default:0" json:"fee"`
	CreatedAt     time.Time   `json:"created_at"`
}

type ExecutionStatus string

const (
	ExecPending ExecutionStatus = "pending"
	ExecFilled  ExecutionStatus = "filled"
	ExecFailed  ExecutionStatus = "failed"
)

type SpotExecution struct {
	ID            uint            `gorm:"primaryKey" json:"id"`
	InstanceID    uint            `gorm:"index;not null" json:"instance_id"`
	ClientOrderID string          `gorm:"uniqueIndex;size:64;not null" json:"client_order_id"`
	Status        ExecutionStatus `gorm:"size:16;index;not null;default:pending" json:"status"`
	Action        TradeAction     `gorm:"size:8;not null" json:"action"`
	Symbol        string          `gorm:"size:32;not null" json:"symbol"`
	RequestQty    float64         `gorm:"not null" json:"request_qty"`
	FilledQty     float64         `gorm:"not null;default:0" json:"filled_qty"`
	FilledPrice   float64         `gorm:"not null;default:0" json:"filled_price"`
	Fee           float64         `gorm:"not null;default:0" json:"fee"`
	ErrorMsg      string          `gorm:"type:text" json:"error_msg,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type AuditLog struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	EventType string         `gorm:"size:64;index;not null" json:"event_type"`
	Payload   datatypes.JSON `gorm:"type:jsonb" json:"payload"`
	CreatedAt time.Time      `json:"created_at"`
}

type GeneRole string

const (
	GeneChallenger GeneRole = "challenger"
	GeneChampion   GeneRole = "champion"
	GeneRetired    GeneRole = "retired"
)

type GeneRecord struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	StrategyID   string         `gorm:"size:64;index;not null" json:"strategy_id"`
	Symbol       string         `gorm:"size:32;index" json:"symbol"`
	Role         GeneRole       `gorm:"size:16;index;not null" json:"role"`
	ParamPack    datatypes.JSON `gorm:"type:jsonb;not null" json:"param_pack"`
	ScoreTotal   float64        `gorm:"not null;default:0" json:"score_total"`
	MaxDrawdown  float64        `gorm:"not null;default:0" json:"max_drawdown"`
	WindowScores datatypes.JSON `gorm:"type:jsonb" json:"window_scores,omitempty"`
	ActivatedAt  *time.Time     `json:"activated_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type EvolutionTaskStatus string

const (
	EvoPending   EvolutionTaskStatus = "pending"
	EvoRunning   EvolutionTaskStatus = "running"
	EvoCompleted EvolutionTaskStatus = "completed"
	EvoFailed    EvolutionTaskStatus = "failed"
)

type EvolutionTask struct {
	ID         uint                `gorm:"primaryKey" json:"id"`
	StrategyID string              `gorm:"size:64;index;not null" json:"strategy_id"`
	Symbol     string              `gorm:"size:32" json:"symbol"`
	Status     EvolutionTaskStatus `gorm:"size:16;index;not null;default:pending" json:"status"`
	Progress   float64             `gorm:"not null;default:0" json:"progress"`
	Config     datatypes.JSON      `gorm:"type:jsonb" json:"config"`
	Result     datatypes.JSON      `gorm:"type:jsonb" json:"result,omitempty"`
	ErrorMsg   string              `gorm:"type:text" json:"error_msg,omitempty"`
	CreatedAt  time.Time           `json:"created_at"`
	UpdatedAt  time.Time           `json:"updated_at"`
}

type KLine struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Symbol    string    `gorm:"size:32;uniqueIndex:idx_kline_unique;not null" json:"symbol"`
	Interval  string    `gorm:"size:8;uniqueIndex:idx_kline_unique;not null" json:"interval"`
	OpenTime  int64     `gorm:"uniqueIndex:idx_kline_unique;not null" json:"open_time"`
	Open      float64   `gorm:"not null" json:"open"`
	High      float64   `gorm:"not null" json:"high"`
	Low       float64   `gorm:"not null" json:"low"`
	Close     float64   `gorm:"not null" json:"close"`
	Volume    float64   `gorm:"not null;default:0" json:"volume"`
	CreatedAt time.Time `json:"created_at"`
}

func AllModels() []interface{} {
	return []interface{}{
		&User{}, &StrategyTemplate{}, &StrategyInstance{}, &PortfolioState{}, &RuntimeState{},
		&SpotLot{}, &TradeRecord{}, &SpotExecution{}, &AuditLog{}, &GeneRecord{}, &EvolutionTask{}, &KLine{},
	}
}
