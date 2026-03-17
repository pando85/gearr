package concurrent

import (
	"sync"
	"testing"
)

func TestMap_SetAndGet(t *testing.T) {
	m := NewMap()

	m.Set("key1", "value1")
	m.Set("key2", 123)

	val, ok := m.Get("key1")
	if !ok {
		t.Error("key1 should exist")
	}
	if val != "value1" {
		t.Errorf("key1 = %v, want value1", val)
	}

	val, ok = m.Get("key2")
	if !ok {
		t.Error("key2 should exist")
	}
	if val != 123 {
		t.Errorf("key2 = %v, want 123", val)
	}

	_, ok = m.Get("nonexistent")
	if ok {
		t.Error("nonexistent key should not exist")
	}
}

func TestMap_Overwrite(t *testing.T) {
	m := NewMap()

	m.Set("key", "value1")
	m.Set("key", "value2")

	val, ok := m.Get("key")
	if !ok {
		t.Error("key should exist")
	}
	if val != "value2" {
		t.Errorf("key = %v, want value2", val)
	}
}

func TestMap_Iter(t *testing.T) {
	m := NewMap()

	m.Set("key1", "value1")
	m.Set("key2", "value2")
	m.Set("key3", "value3")

	count := 0
	for item := range m.Iter() {
		count++
		if item.Key != "key1" && item.Key != "key2" && item.Key != "key3" {
			t.Errorf("unexpected key: %s", item.Key)
		}
	}

	if count != 3 {
		t.Errorf("Iter returned %d items, want 3", count)
	}
}

func TestMap_ConcurrentAccess(t *testing.T) {
	m := NewMap()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := string(rune('a' + i%26))
			m.Set(key, i)
		}(i)
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := string(rune('a' + i%26))
			m.Get(key)
		}(i)
	}

	wg.Wait()
}

func TestSlice_Append(t *testing.T) {
	var s Slice

	s.Append("item1")
	s.Append("item2")
	s.Append("item3")

	count := 0
	for item := range s.Iter() {
		count++
		if item.Index < 0 || item.Index > 2 {
			t.Errorf("unexpected index: %d", item.Index)
		}
	}

	if count != 3 {
		t.Errorf("Iter returned %d items, want 3", count)
	}
}

func TestSlice_Delete(t *testing.T) {
	var s Slice

	s.Append("item1")
	s.Append("item2")
	s.Append("item3")

	s.Delete("item2")

	count := 0
	for item := range s.Iter() {
		count++
		if item.Value == "item2" {
			t.Error("item2 should have been deleted")
		}
	}

	if count != 2 {
		t.Errorf("Iter returned %d items, want 2", count)
	}
}

func TestSlice_DeleteNonExistent(t *testing.T) {
	var s Slice

	s.Append("item1")
	originalCount := 0
	for range s.Iter() {
		originalCount++
	}

	s.Delete("nonexistent")

	count := 0
	for range s.Iter() {
		count++
	}

	if count != originalCount {
		t.Errorf("count after deleting nonexistent = %d, want %d", count, originalCount)
	}
}

func TestSlice_ConcurrentAccess(t *testing.T) {
	var s Slice
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.Append(i)
		}(i)
	}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.Delete(i)
		}(i)
	}

	wg.Wait()
}
