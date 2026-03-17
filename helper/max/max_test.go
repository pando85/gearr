package max

import (
	"testing"

	"github.com/google/uuid"
)

type testItem struct {
	id    int
	value int
}

type testItems []*testItem

func (t testItems) Len() int                         { return len(t) }
func (t testItems) Less(i, j int) bool               { return t[i].value < t[j].value }
func (t testItems) Swap(i, j int)                    { t[i], t[j] = t[j], t[i] }
func (t testItems) GetLastElement(i int) interface{} { return t[i] }

func TestMax(t *testing.T) {
	tests := []struct {
		name     string
		items    testItems
		expected int
	}{
		{
			name:     "single element",
			items:    testItems{{id: 1, value: 42}},
			expected: 42,
		},
		{
			name:     "multiple elements",
			items:    testItems{{id: 1, value: 10}, {id: 2, value: 30}, {id: 3, value: 20}},
			expected: 30,
		},
		{
			name:     "already sorted",
			items:    testItems{{id: 1, value: 1}, {id: 2, value: 2}, {id: 3, value: 3}},
			expected: 3,
		},
		{
			name:     "reverse sorted",
			items:    testItems{{id: 1, value: 3}, {id: 2, value: 2}, {id: 3, value: 1}},
			expected: 3,
		},
		{
			name:     "negative values",
			items:    testItems{{id: 1, value: -10}, {id: 2, value: -5}, {id: 3, value: -20}},
			expected: -5,
		},
		{
			name:     "all same values",
			items:    testItems{{id: 1, value: 5}, {id: 2, value: 5}, {id: 3, value: 5}},
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Max(tt.items).(*testItem)
			if result.value != tt.expected {
				t.Errorf("Max() = %d, want %d", result.value, tt.expected)
			}
		})
	}
}

func TestMaxEmpty(t *testing.T) {
	var items testItems
	result := Max(items)
	if result != nil {
		t.Errorf("Max() = %v, want nil", result)
	}
}

type taskEvents []*taskEvent

type taskEvent struct {
	eventID int
	id      uuid.UUID
}

func (t taskEvents) Len() int                         { return len(t) }
func (t taskEvents) Less(i, j int) bool               { return t[i].eventID < t[j].eventID }
func (t taskEvents) Swap(i, j int)                    { t[i], t[j] = t[j], t[i] }
func (t taskEvents) GetLastElement(i int) interface{} { return t[i] }

func TestMaxWithTaskEvents(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New()

	tests := []struct {
		name         string
		items        taskEvents
		expectedID   int
		expectedUUID uuid.UUID
	}{
		{
			name:         "find max eventID",
			items:        taskEvents{{eventID: 1, id: id1}, {eventID: 5, id: id2}, {eventID: 3, id: id3}},
			expectedID:   5,
			expectedUUID: id2,
		},
		{
			name:         "single event",
			items:        taskEvents{{eventID: 10, id: id1}},
			expectedID:   10,
			expectedUUID: id1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Max(tt.items).(*taskEvent)
			if result.eventID != tt.expectedID {
				t.Errorf("Max() eventID = %d, want %d", result.eventID, tt.expectedID)
			}
			if result.id != tt.expectedUUID {
				t.Errorf("Max() id = %v, want %v", result.id, tt.expectedUUID)
			}
		})
	}
}
