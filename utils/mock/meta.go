package mock

import (
	"sync"
	"time"

	"github.com/Focinfi/oncekv/meta"
)

// Meta a meta.Meta mock
type Meta struct {
	sync.RWMutex
	data map[string]string
}

// NewMeta returns a new Meta
func NewMeta() *Meta {
	return &Meta{data: map[string]string{}}
}

// Get gets the value of the given key
func (m *Meta) Get(key string) (string, error) {
	m.RLock()
	defer m.RUnlock()

	val, ok := m.data[key]
	if !ok {
		return "", meta.ErrDataNotFound
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
	tikcer := time.NewTicker(time.Millisecond * 500)
	for {
		select {
		case <-tikcer.C:
			do()
		}
	}
}
