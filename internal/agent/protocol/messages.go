package protocol

// Message envelope for SaaS ↔ Agent (WebSocket in Phase 8; in-process for Phase 7 tests).
type Type string

const (
	TypeHello        Type = "hello"
	TypeHeartbeat    Type = "heartbeat"
	TypeTradeCommand Type = "trade_command"
	TypeFillReport   Type = "fill_report"
	TypeDeltaReport  Type = "delta_report"
	TypeError        Type = "error"
)

// Envelope is the wire format; Payload is type-specific JSON.
type Envelope struct {
	Type      Type   `json:"type"`
	RequestID string `json:"request_id,omitempty"`
	Payload   []byte `json:"payload"` // raw JSON of the typed body
}

// Hello is sent by Agent after auth to register which instances it serves.
type Hello struct {
	AgentID     string `json:"agent_id"`
	Version     string `json:"version"`
	InstanceIDs []uint `json:"instance_ids,omitempty"`
}

// Heartbeat keeps the session alive.
type Heartbeat struct {
	AgentID string `json:"agent_id"`
	TsMs    int64  `json:"ts_ms"`
}

// TradeCommand is issued by SaaS after Step(); Agent must NOT interpret strategy.
// API keys never appear in this message.
type TradeCommand struct {
	ClientOrderID string  `json:"client_order_id"`
	InstanceID    uint    `json:"instance_id"`
	Symbol        string  `json:"symbol"`
	Side          string  `json:"side"`   // BUY | SELL
	Engine        string  `json:"engine"` // MACRO | MICRO
	Qty           float64 `json:"qty"`
	OrderType     string  `json:"order_type"` // MARKET (default)
}

// FillReport is the execution result for one TradeCommand.
type FillReport struct {
	ClientOrderID string  `json:"client_order_id"`
	InstanceID    uint    `json:"instance_id"`
	Symbol        string  `json:"symbol"`
	Side          string  `json:"side"`
	Engine        string  `json:"engine"`
	FilledQty     float64 `json:"filled_qty"`
	FilledPrice   float64 `json:"filled_price"`
	Fee           float64 `json:"fee"`
	Status        string  `json:"status"` // filled | failed | partial
	ErrorMsg      string  `json:"error_msg,omitempty"`
	TsMs          int64   `json:"ts_ms"`
}

// DeltaReport is a broker-side position snapshot (Agent has no Dead/Float notion).
// SaaS maps this into its three-stack ledger via ApplyFill history.
type DeltaReport struct {
	InstanceID uint    `json:"instance_id"`
	Symbol     string  `json:"symbol"`
	Cash       float64 `json:"cash"`
	Shares     float64 `json:"shares"`
	TsMs       int64   `json:"ts_ms"`
}

// ErrorBody is a generic protocol error.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
