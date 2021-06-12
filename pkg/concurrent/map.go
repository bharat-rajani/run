package concurrent

import "sync"

// ConcurrentMap is an interface representing the functionality of hashmap
// which can be used for concurrent reads and writes.
//
// A ConcurrentMap must be safe for concurrent use by multiple
// goroutines.
type ConcurrentMap interface {
	Load(interface{}) (interface{}, bool)
	Store(key, value interface{})
	LoadOrStore(key, value interface{}) (actual interface{}, loaded bool)
	LoadAndDelete(key interface{}) (value interface{}, loaded bool)
	Delete(interface{})
	Range(func(key, value interface{}) (shouldContinue bool))
}

// RWMutexMap is like a Go map[interface{}]interface{} but is safe for concurrent use
// by multiple goroutines .
// Loads, stores, and deletes run in amortized constant time.
//
// It is essentially a Go map paired with a separate RWMutex.
//
// You may look out for sync.Map from go library which provides better performance
// under certain cases like "less write many reads".
type RWMutexMap struct {
	mu       sync.RWMutex
	internal map[interface{}]interface{}
}

func NewRWMutexMap() *RWMutexMap {
	return &RWMutexMap{
		mu:       sync.RWMutex{},
		internal: make(map[interface{}]interface{}),
	}
}

func (m *RWMutexMap) Load(key interface{}) (value interface{}, ok bool) {
	m.mu.RLock()
	value, ok = m.internal[key]
	m.mu.RUnlock()
	return
}

func (m *RWMutexMap) Store(key, value interface{}) {
	m.mu.Lock()
	if m.internal == nil {
		m.internal = make(map[interface{}]interface{})
	}
	m.internal[key] = value
	m.mu.Unlock()
}

func (m *RWMutexMap) LoadOrStore(key, value interface{}) (actual interface{}, loaded bool) {
	m.mu.Lock()
	actual, loaded = m.internal[key]
	if !loaded {
		actual = value
		if m.internal == nil {
			m.internal = make(map[interface{}]interface{})
		}
		m.internal[key] = value
	}
	m.mu.Unlock()
	return actual, loaded
}

func (m *RWMutexMap) LoadAndDelete(key interface{}) (value interface{}, loaded bool) {
	m.mu.Lock()
	value, loaded = m.internal[key]
	if !loaded {
		m.mu.Unlock()
		return nil, false
	}
	delete(m.internal, key)
	m.mu.Unlock()
	return value, loaded
}

func (m *RWMutexMap) Delete(key interface{}) {
	m.mu.Lock()
	delete(m.internal, key)
	m.mu.Unlock()
}

func (m *RWMutexMap) Range(f func(key, value interface{}) (shouldContinue bool)) {
	m.mu.RLock()
	keys := make([]interface{}, 0, len(m.internal))
	for k := range m.internal {
		keys = append(keys, k)
	}
	m.mu.RUnlock()

	for _, k := range keys {
		v, ok := m.Load(k)
		if !ok {
			continue
		}
		if !f(k, v) {
			break
		}
	}
}
