package util

import (
	"sync"
)

type SafeObj[T any] struct {
	obj     T
	mutex   sync.Mutex
	updated bool
	Reload  func() (T, error)
}

func (m *SafeObj[T]) Get() (T, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if !m.updated {
		obj, err := m.Reload()
		if err != nil {
			return m.obj, err
		}
		m.obj = obj
		m.updated = true
	}
	v := m.obj
	return v, nil
}

func (m *SafeObj[T]) Refresh() {
	m.mutex.Lock()
	m.updated = false
	m.mutex.Unlock()
}
