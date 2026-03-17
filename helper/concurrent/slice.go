package concurrent

import "sync"

type Slice[T any] struct {
	sync.RWMutex
	items []T
}

type SliceItem[T any] struct {
	Index int
	Value T
}

func NewSlice[T any]() *Slice[T] {
	return &Slice[T]{
		items: make([]T, 0),
	}
}

func (cs *Slice[T]) Append(item T) {
	cs.Lock()
	defer cs.Unlock()

	cs.items = append(cs.items, item)
}

func (cs *Slice[T]) Iter() <-chan SliceItem[T] {
	c := make(chan SliceItem[T], 10)

	f := func() {
		cs.Lock()
		defer cs.Unlock()
		for index, value := range cs.items {
			c <- SliceItem[T]{index, value}
		}
		close(c)
	}
	go f()

	return c
}

func (cs *Slice[T]) Delete(item T) {
	cs.Lock()
	defer cs.Unlock()
	foundIndex := -1
	for index, value := range cs.items {
		if any(value) == any(item) {
			foundIndex = index
			break
		}
	}
	if foundIndex != -1 && len(cs.items) > 0 {
		cs.items[foundIndex] = cs.items[len(cs.items)-1]
		var zero T
		cs.items[len(cs.items)-1] = zero
		cs.items = cs.items[:len(cs.items)-1]
	}
}

func (cs *Slice[T]) Len() int {
	cs.RLock()
	defer cs.RUnlock()
	return len(cs.items)
}
