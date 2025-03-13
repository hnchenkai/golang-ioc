package ioc

import "sync"

type IMap[K string, T any] interface {
	Clear()
	CompareAndDelete(key K, old K) (deleted bool)
	CompareAndSwap(key K, old K, new K) (swapped bool)
	Delete(key K)
	Load(key K) (value *T, ok bool)
	LoadAndDelete(key K) (value *T, loaded bool)
	LoadOrNew(key K, f func(key K) *T) (actual *T, loaded bool)
	LoadOrStore(key K, value *T) (actual *T, loaded bool)
	// Range calls f sequentially for each key and value present in the map. If f returns false, range stops the iteration.
	Range(f func(key K, value *T) bool)
	Store(key K, value *T)
	Swap(key K, value *T) (previous *T, loaded bool)
}

type Map[K string, T any] struct {
	m sync.Map
}

func NewMap[K string, T any]() *Map[K, T] {
	return &Map[K, T]{}
}

func (m *Map[K, T]) Clear() {
	// m.m.Clear()
}
func (m *Map[K, T]) CompareAndDelete(key K, old K) (deleted bool) {
	return m.m.CompareAndDelete(key, old)
}
func (m *Map[K, T]) CompareAndSwap(key K, old K, new K) (swapped bool) {
	return m.m.CompareAndSwap(key, old, new)
}
func (m *Map[K, T]) Delete(key K) {
	m.m.Delete(key)
}
func (m *Map[K, T]) Load(key K) (value *T, ok bool) {
	v, ok := m.m.Load(key)
	if ok {
		return v.(*T), ok
	}
	return nil, false
}
func (m *Map[K, T]) LoadAndDelete(key K) (value *T, loaded bool) {
	v, ok := m.m.LoadAndDelete(key)
	if v != nil {
		return v.(*T), ok
	}
	return nil, false
}
func (m *Map[K, T]) LoadOrStore(key K, value *T) (actual *T, loaded bool) {
	v, ok := m.m.LoadOrStore(key, value)
	return v.(*T), ok
}

func (m *Map[K, T]) LoadOrNew(key K, f func(key K) *T) (actual *T, loaded bool) {
	if v, ok := m.m.Load(key); ok {
		return v.(*T), ok
	} else {
		v, ok := m.m.LoadOrStore(key, f(key))
		return v.(*T), ok
	}
}
func (m *Map[K, T]) Range(f func(key K, value *T) bool) {
	m.m.Range(func(key, value any) bool {
		return f(key.(K), value.(*T))
	})
}
func (m *Map[K, T]) Store(key K, value *T) {
	m.m.Store(key, value)
}

func (m *Map[K, T]) Len() int {
	count := 0
	m.Range(func(key K, value *T) bool {
		count++
		return true
	})
	return count
}

func (m *Map[K, T]) Swap(key K, value *T) (previous *T, loaded bool) {
	v, ok := m.m.Swap(key, value)
	if v != nil {
		return v.(*T), ok
	}
	return nil, false
}
