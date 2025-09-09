package util

import (
	"sync"
)

type SafeMap[T, T2 any] struct {
	data sync.Map
	empty T2
}

func (m *SafeMap[T, T2]) Set(key T, value T2) {
	m.data.Store(key, value)
}

func (m *SafeMap[T, T2]) Get(key T) (T2, bool) {
	v, ok := m.data.Load(key)
	if !ok {
		return m.empty, ok
	}
	return v.(T2), ok
}

func (m *SafeMap[T, T2]) Delete(key T) {
	m.data.Delete(key)
}

func (m *SafeMap[T, T2]) Clear() {
	m.data = sync.Map{}
}

func (m *SafeMap[T, T2]) Length() int {
	i := 0
	m.data.Range(func(key, val any) bool {
		i++
		return true
	})
	return i
}

func (m *SafeMap[T, T2]) Init() {
	m.data = sync.Map{}
}

func (m *SafeMap[T, T2]) Iter(f func(T, T2) (T2, bool)) {
	m.data.Range(func(key, val any) bool {
		n, keep := f(key.(T), val.(T2))
		if keep == false {
			m.data.Delete(key)
		} else {
			m.data.Store(key, n)
		}
		return true
	})
}
