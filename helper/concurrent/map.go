package concurrent

import "sync"

type Map struct {
	sync.RWMutex
	items map[string]interface{}
}

// concurrent map item
type MapItem struct {
	Key   string
	Value interface{}
}
func (cm *Map) Set(key string, value interface{}) {
	cm.Lock()
	defer cm.Unlock()

	cm.items[key] = value
}

func (cm *Map) Get(key string) (interface{}, bool) {
	cm.Lock()
	defer cm.Unlock()

	value, ok := cm.items[key]

	return value, ok
}

func (cm *Map) Iter() <-chan MapItem {
	c := make(chan MapItem)

	f := func() {
		cm.Lock()
		defer cm.Unlock()

		for k, v := range cm.items {
			c <- MapItem{k, v}
		}
		close(c)
	}
	go f()

	return c
}