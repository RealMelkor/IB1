package util

import (
	"sync"
)

type SafeMap[T any] struct {
        data    map[string]T
        mutex   sync.Mutex
}

func (m *SafeMap[any]) Set(key string, value any) {
        m.mutex.Lock()
        m.data[key] = value
        m.mutex.Unlock()
}

func (m *SafeMap[any]) Get(key string) (any, bool) {
        m.mutex.Lock()
        v, ok := m.data[key]
        m.mutex.Unlock()
        return v, ok
}

func (m *SafeMap[any]) Delete(key string) {
        m.mutex.Lock()
        delete(m.data, key)
        m.mutex.Unlock()
}

func (m *SafeMap[any]) Clear() {
        m.mutex.Lock()
        m.data = map[string]any{}
        m.mutex.Unlock()
}

func (m *SafeMap[any]) Length() int {
        m.mutex.Lock()
        v := len(m.data)
        m.mutex.Unlock()
        return v
}

func (m *SafeMap[any]) Init() {
        m.data = map[string]any{}
        m.mutex = sync.Mutex{}
}

func (m *SafeMap[any]) Iter(f func(string, any)(any,bool)) {
        m.mutex.Lock()
	for k, v := range m.data {
		n, keep := f(k, v)
		if keep == false {
			delete(m.data, k)
		} else {
			m.data[k] = n
		}
	}
        m.mutex.Unlock()
}
