package market

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/store"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Service manages K-line storage used by the live ticker.
type Service struct {
	db *store.DB
}

func New(db *store.DB) *Service {
	return &Service{db: db}
}

// BarInput is one OHLCV row for import.
type BarInput struct {
	OpenTime int64   `json:"open_time"` // unix ms
	Open     float64 `json:"open"`
	High     float64 `json:"high"`
	Low      float64 `json:"low"`
	Close    float64 `json:"close"`
	Volume   float64 `json:"volume"`
}

// ImportBars upserts bars for symbol+interval (idempotent on unique index).
func (s *Service) ImportBars(symbol, interval string, bars []BarInput) (int, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	interval = strings.TrimSpace(interval)
	if interval == "" {
		interval = "1m"
	}
	if symbol == "" {
		return 0, fmt.Errorf("symbol required")
	}
	if len(bars) == 0 {
		return 0, nil
	}

	rows := make([]store.KLine, 0, len(bars))
	for _, b := range bars {
		if b.OpenTime <= 0 || b.Close <= 0 {
			continue
		}
		h, l := b.High, b.Low
		if h <= 0 {
			h = b.Close
		}
		if l <= 0 {
			l = b.Close
		}
		o := b.Open
		if o <= 0 {
			o = b.Close
		}
		rows = append(rows, store.KLine{
			Symbol:   symbol,
			Interval: interval,
			OpenTime: b.OpenTime,
			Open:     o,
			High:     h,
			Low:      l,
			Close:    b.Close,
			Volume:   b.Volume,
		})
	}
	if len(rows) == 0 {
		return 0, fmt.Errorf("no valid bars")
	}

	err := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "symbol"}, {Name: "interval"}, {Name: "open_time"}},
		DoUpdates: clause.AssignmentColumns([]string{"open", "high", "low", "close", "volume"}),
	}).CreateInBatches(rows, 200).Error
	if err != nil {
		return 0, err
	}
	return len(rows), nil
}

// ListBars returns the most recent n bars in chronological order.
func (s *Service) ListBars(symbol, interval string, n int) ([]store.KLine, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if interval == "" {
		interval = "1m"
	}
	if n <= 0 {
		n = 120
	}
	if n > 5000 {
		n = 5000
	}
	var rows []store.KLine
	err := s.db.Where("symbol = ? AND interval = ?", symbol, interval).
		Order("open_time DESC").Limit(n).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	return rows, nil
}

// LastClose returns the latest close price for symbol (any interval preferred 1m).
func (s *Service) LastClose(symbol string) (float64, int64, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	var row store.KLine
	err := s.db.Where("symbol = ?", symbol).Order("open_time DESC").First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, 0, fmt.Errorf("no klines for %s", symbol)
		}
		return 0, 0, err
	}
	return row.Close, row.OpenTime, nil
}

// SeedOpts controls synthetic bar generation for demos.
type SeedOpts struct {
	Symbol    string
	Interval  string
	Bars      int
	StartPx   float64
	Drift     float64
	Vol       float64
	EndTimeMs int64
}

// SeedSynthetic writes a random-walk series ending near now (demo-friendly decline).
func (s *Service) SeedSynthetic(opts SeedOpts) (int, error) {
	if opts.Symbol == "" {
		opts.Symbol = "510300"
	}
	if opts.Interval == "" {
		opts.Interval = "1m"
	}
	if opts.Bars <= 0 {
		opts.Bars = 200
	}
	if opts.StartPx <= 0 {
		opts.StartPx = 4.50
	}
	if opts.Drift == 0 {
		opts.Drift = -0.00015
	}
	if opts.Vol <= 0 {
		opts.Vol = 0.0015
	}
	end := opts.EndTimeMs
	if end <= 0 {
		end = time.Now().UnixMilli()
	}
	stepMs := intervalToMs(opts.Interval)

	bars := make([]BarInput, 0, opts.Bars)
	px := opts.StartPx
	startOpen := end - int64(opts.Bars-1)*stepMs
	for i := 0; i < opts.Bars; i++ {
		noise := math.Sin(float64(i)*0.37)*opts.Vol + math.Cos(float64(i)*0.19)*opts.Vol*0.5
		px = px * math.Exp(opts.Drift+noise)
		if px < 0.01 {
			px = 0.01
		}
		ot := startOpen + int64(i)*stepMs
		hi := px * (1 + math.Abs(noise)*0.5)
		lo := px * (1 - math.Abs(noise)*0.5)
		bars = append(bars, BarInput{
			OpenTime: ot,
			Open:     px,
			High:     hi,
			Low:      lo,
			Close:    px,
			Volume:   1000 + float64(i%50)*10,
		})
	}
	return s.ImportBars(opts.Symbol, opts.Interval, bars)
}

func intervalToMs(iv string) int64 {
	switch strings.ToLower(strings.TrimSpace(iv)) {
	case "1m":
		return 60_000
	case "5m":
		return 5 * 60_000
	case "15m":
		return 15 * 60_000
	case "1h", "60m":
		return 60 * 60_000
	case "1d", "d":
		return 24 * 60 * 60_000
	default:
		return 60_000
	}
}
