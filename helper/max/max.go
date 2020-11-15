package max

import "sort"

type Extractor interface{
	sort.Interface
	GetLastElement(i int) interface{}
}

func Max(maxExtractor Extractor) interface{} {
	sort.Sort(maxExtractor.(sort.Interface))
	return maxExtractor.GetLastElement(maxExtractor.Len()-1)
}