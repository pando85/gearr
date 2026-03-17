package concurrent

import "sync"

type Map[K comparable, V any] struct {
	sync.RWMutex
	items map[K]V
}

type MapItem[K comparable, V any] struct {
	Key   K
	Value V
}

func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		items: make(map[K]V),
	}
}

func (cm *Map[K, V]) Set(key K, value V) {
	cm.Lock()
	defer cm.Unlock()

	cm.items[key] = value
}

func (cm *Map[K, V]) Get(key K) (V, bool) {
	cm.RLock()
	defer cm.RUnlock()

	value, ok := cm.items[key]

	return value, ok
}

func (cm *Map[K, V]) Delete(key K) {
	cm.Lock()
	defer cm.Unlock()

	delete(cm.items, key)
}

func (cm *Map[K, V]) Iter() <-chan MapItem[K, V] {
	c := make(chan MapItem[K, V])

	f := func() {
		cm.Lock()
		defer cm.Unlock()

		for k, v := range cm.items {
			c <- MapItem[K, V]{k, v}
		}
		close(c)
	}
	go f()

	return c
}
