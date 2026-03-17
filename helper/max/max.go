package max

import "sort"

type Extractor interface {
	sort.Interface
	GetLastElement(i int) interface{}
}

func Max(maxExtractor Extractor) interface{} {
	n := maxExtractor.Len()
	if n == 0 {
		return nil
	}

	maxIdx := 0
	for i := 1; i < n; i++ {
		if maxExtractor.Less(maxIdx, i) {
			maxIdx = i
		}
	}
	return maxExtractor.GetLastElement(maxIdx)
}
