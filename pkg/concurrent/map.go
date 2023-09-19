package concurrent

import "sync"

// ConcurrentMap is an interface representing the functionality of hashmap
// which can be used for concurrent reads and writes.
//
// A ConcurrentMap must be safe for concurrent use by multiple
// goroutines.
type ConcurrentMap interface {
	Load(key any) (value any, ok bool)
	Store(key any, value any)
	LoadOrStore(key any, value any) (actual any, loaded bool)
	LoadAndDelete(key any) (value any, loaded bool)
	Delete(key any)
	Range(func(key any, value any) (shouldContinue bool))
}

// RWMutexMap is a concurrency-safe alternative to a Go map[any]any,
// allowing for concurrent operations like loading, storing, and deleting
// with amortized constant time. It combines a Go map with a separate RWMutex.
//
// Note: For improved performance, consider using the sync.Map from the Go
// standard library, especially in scenarios with fewer write operations
// and many read operations.
type RWMutexMap struct {
	mu       sync.RWMutex
	internal map[any]any
}

// NewRWMutexMap creates and initializes a new instance of RWMutexMap, a concurrency-safe map
// with an underlying RWMutex to manage concurrent access.
//
// Example usage:
//
//	myMap := NewRWMutexMap()
//	myMap.Store(key, value)
//	val, found := myMap.Load(key)
//
// To ensure proper initialization, always use NewRWMutexMap to create a new instance
// instead of manually creating RWMutexMap instances.
//
// Returns:
//   - A pointer to a newly created RWMutexMap.
func NewRWMutexMap() *RWMutexMap {
	return &RWMutexMap{
		mu:       sync.RWMutex{},
		internal: make(map[any]any),
	}
}

// Load retrieves the value associated with the specified key from the RWMutexMap
// in a thread-safe manner. It acquires a read lock on the underlying RWMutex to
// allow concurrent reads without blocking. If the key is found, it returns the
// associated value and true; otherwise, it returns a zero value and false.
//
// Parameters:
//   - key: The key for which the associated value is to be retrieved.
//
// Returns:
//   - value: The value associated with the key (if found).
//   - ok: A boolean indicating whether the key was found in the map.
func (m *RWMutexMap) Load(key any) (value any, ok bool) {
	m.mu.RLock()
	value, ok = m.internal[key]
	m.mu.RUnlock()
	return
}

func (m *RWMutexMap) Store(key any, value any) {
	m.mu.Lock()
	if m.internal == nil {
		m.internal = make(map[any]any)
	}
	m.internal[key] = value
	m.mu.Unlock()
}

func (m *RWMutexMap) LoadOrStore(key any, value any) (actual any, loaded bool) {
	m.mu.Lock()
	actual, loaded = m.internal[key]
	if !loaded {
		actual = value
		if m.internal == nil {
			m.internal = make(map[any]any)
		}
		m.internal[key] = value
	}
	m.mu.Unlock()
	return actual, loaded
}

func (m *RWMutexMap) LoadAndDelete(key any) (value any, ok bool) {
	m.mu.Lock()
	value, ok = m.internal[key]
	if !ok {
		m.mu.Unlock()
		var zeroVal any
		return zeroVal, false
	}
	delete(m.internal, key)
	m.mu.Unlock()
	return value, ok
}

func (m *RWMutexMap) Delete(key any) {
	m.mu.Lock()
	delete(m.internal, key)
	m.mu.Unlock()
}

func (m *RWMutexMap) Range(f func(key any, value any) (shouldContinue bool)) {
	m.mu.RLock()
	keys := make([]any, 0, len(m.internal))
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
func (m *RWMutexMap) UpdateRange(f func(val any) any) {
	m.mu.Lock()
	for k, v := range m.internal {
		m.internal[k] = f(v)
	}
	m.mu.Unlock()
	return
}
