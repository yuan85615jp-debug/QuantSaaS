package instance

import "github.com/yuan85615jp-debug/QuantSaaS/internal/saas/store"

// ListRunning returns all instances in RUNNING status (ticker / live loop).
func (s *Service) ListRunning() ([]store.StrategyInstance, error) {
	var list []store.StrategyInstance
	if err := s.db.Where("status = ?", store.InstanceRunning).Order("id ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
