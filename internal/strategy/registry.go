package strategy

import "sync"

type Factory func() Strategy

var (
	mu        sync.RWMutex
	factories = map[string]Factory{}
)

func Register(id string, f Factory) {
	mu.Lock()
	defer mu.Unlock()
	factories[id] = f
}

func Get(id string) (Strategy, bool) {
	mu.RLock()
	defer mu.RUnlock()
	f, ok := factories[id]
	if !ok {
		return nil, false
	}
	return f(), true
}

func List() []string {
	mu.RLock()
	defer mu.RUnlock()
	ids := make([]string, 0, len(factories))
	for id := range factories {
		ids = append(ids, id)
	}
	return ids
}
