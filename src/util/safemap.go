package util

import (
	"sync"
)

type SafeMap[T any] struct {
	data	sync.Map
}

func (m *SafeMap[T]) Set(key string, value T) {
        m.data.Store(key, value)
}

func (m *SafeMap[T]) Get(key string) (T, bool) {
	v, ok := m.data.Load(key)
	return v.(T), ok
}

func (m *SafeMap[T]) Delete(key string) {
	m.data.Delete(key)
}

func (m *SafeMap[T]) Clear() {
	m.data = sync.Map{}
}

func (m *SafeMap[T]) Length() int {
	i := 0
	m.data.Range(func(key, val any)bool{
		i++
		return true
	})
        return i
}

func (m *SafeMap[T]) Init() {
        m.data = sync.Map{}
}

func (m *SafeMap[T]) Iter(f func(string, T)(T, bool)) {
	m.data.Range(func(key, val any)bool {
		n, keep := f(key.(string), val.(T))
		if keep == false {
			m.data.Delete(key)
		} else {
			m.data.Store(key, n)
		}
		return true
	})
}
