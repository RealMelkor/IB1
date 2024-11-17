package web

import (
	"sync"
)

type safeMap[T any] struct {
        data    map[string]T
        mutex   sync.Mutex
}

func (m *safeMap[any]) set(key string, value any) {
        m.mutex.Lock()
        m.data[key] = value
        m.mutex.Unlock()
}

func (m *safeMap[any]) get(key string) (any, bool) {
        m.mutex.Lock()
        v, ok := m.data[key]
        m.mutex.Unlock()
        return v, ok
}

func (m *safeMap[any]) clear() {
        m.mutex.Lock()
        m.data = map[string]any{}
        m.mutex.Unlock()
}

func (m *safeMap[any]) length() int {
        m.mutex.Lock()
        v := len(m.data)
        m.mutex.Unlock()
        return v
}

func (m *safeMap[any]) Init() {
        m.data = map[string]any{}
        m.mutex = sync.Mutex{}
}
