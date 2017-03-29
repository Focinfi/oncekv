package mock

import (
	"sync"
	"time"

	"github.com/Focinfi/oncekv/config"
)

// DefaultWatchPeriod for value changing period
var DefaultWatchPeriod = time.Millisecond * 500

// Meta a meta.Meta mock
type Meta struct {
	sync.RWMutex
	data map[string]string
}

// DefaultMeta for default mock meta
var DefaultMeta = &Meta{data: map[string]string{}}

// Get gets the value of the given key
func (m *Meta) Get(key string) (string, error) {
	m.RLock()
	defer m.RUnlock()

	val, ok := m.data[key]
	if !ok {
		return "", config.ErrDataNotFound
	}

	return val, nil
}

// Put the key/value pair
func (m *Meta) Put(key string, value string) error {
	m.Lock()
	defer m.Unlock()

	m.data[key] = value
	return nil
}

// WatchModify watch the modification event of the value the given key
// and execute the do
func (m *Meta) WatchModify(key string, do func()) {
	tikcer := time.NewTicker(DefaultWatchPeriod)
	for {
		select {
		case <-tikcer.C:
			do()
		}
	}
}
